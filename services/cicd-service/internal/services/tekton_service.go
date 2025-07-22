package services

import (
	"context"
	"fmt"
	"time"

	"cicd-service/internal/config"
	"cicd-service/internal/models"

	"github.com/google/uuid"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// TektonService Tekton集成服务接口
type TektonService interface {
	// Pipeline操作
	CreatePipeline(ctx context.Context, pipeline *models.Pipeline) error
	UpdatePipeline(ctx context.Context, pipeline *models.Pipeline) error
	DeletePipeline(ctx context.Context, pipelineID uuid.UUID) error
	
	// PipelineRun操作
	CreatePipelineRun(ctx context.Context, req *TektonPipelineRunRequest) error
	GetPipelineRunStatus(ctx context.Context, runID uuid.UUID) (*TektonPipelineRunStatus, error)
	CancelPipelineRun(ctx context.Context, runID uuid.UUID) error
	
	// TaskRun操作
	GetTaskRunStatus(ctx context.Context, taskRunID uuid.UUID) (*TektonTaskRunStatus, error)
	GetTaskRunLogs(ctx context.Context, taskRunID uuid.UUID) (string, error)
	
	// 事件监听
	WatchPipelineRuns(ctx context.Context) error
	WatchTaskRuns(ctx context.Context) error
	
	// 资源管理
	CreateSecret(ctx context.Context, secret *models.Secret) error
	UpdateSecret(ctx context.Context, secret *models.Secret) error
	DeleteSecret(ctx context.Context, secretID uuid.UUID) error
	
	// 健康检查
	HealthCheck(ctx context.Context) error
}

type tektonService struct {
	config        *config.Config
	k8sClient     kubernetes.Interface
	tektonClient  tektonclient.Interface
	restConfig    *rest.Config
	runService    PipelineRunService
}

// NewTektonService 创建Tekton服务实例
func NewTektonService(cfg *config.Config, runService PipelineRunService) (TektonService, error) {
	var restConfig *rest.Config
	var err error

	if cfg.Kubernetes.InCluster {
		// 集群内配置
		restConfig, err = rest.InClusterConfig()
	} else {
		// 集群外配置
		if cfg.Kubernetes.ConfigPath != "" {
			restConfig, err = clientcmd.BuildConfigFromFlags("", cfg.Kubernetes.ConfigPath)
		} else {
			// 使用默认kubeconfig路径
			restConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				clientcmd.NewDefaultClientConfigLoadingRules(),
				&clientcmd.ConfigOverrides{},
			).ClientConfig()
		}
	}

	if err != nil {
		return nil, fmt.Errorf("创建Kubernetes配置失败: %w", err)
	}

	// 创建客户端
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	tektonClient, err := tektonclient.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("创建Tekton客户端失败: %w", err)
	}

	return &tektonService{
		config:       cfg,
		k8sClient:    k8sClient,
		tektonClient: tektonClient,
		restConfig:   restConfig,
		runService:   runService,
	}, nil
}

// TektonPipelineRunRequest Tekton流水线运行请求
type TektonPipelineRunRequest struct {
	Name           string                 `json:"name"`
	PipelineID     uuid.UUID              `json:"pipeline_id"`
	RunID          uuid.UUID              `json:"run_id"`
	Parameters     map[string]interface{} `json:"parameters"`
	Timeout        int                    `json:"timeout"`
	Workspace      string                 `json:"workspace"`
	ServiceAccount string                 `json:"service_account"`
}

// TektonPipelineRunStatus Tekton流水线运行状态
type TektonPipelineRunStatus struct {
	Status      string                 `json:"status"`
	Message     string                 `json:"message"`
	StartTime   *time.Time             `json:"start_time"`
	EndTime     *time.Time             `json:"end_time"`
	TaskRuns    map[string]interface{} `json:"task_runs"`
}

// TektonTaskRunStatus Tekton任务运行状态
type TektonTaskRunStatus struct {
	Status    string     `json:"status"`
	Message   string     `json:"message"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	Steps     []StepStatus `json:"steps"`
}

// StepStatus 步骤状态
type StepStatus struct {
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	Message   string     `json:"message"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
}

// CreatePipeline 创建Tekton Pipeline
func (s *tektonService) CreatePipeline(ctx context.Context, pipeline *models.Pipeline) error {
	tektonPipeline := s.buildTektonPipeline(pipeline)
	
	_, err := s.tektonClient.TektonV1beta1().
		Pipelines(s.config.Kubernetes.Namespace).
		Create(ctx, tektonPipeline, metav1.CreateOptions{})
	
	if err != nil {
		return fmt.Errorf("创建Tekton Pipeline失败: %w", err)
	}
	
	return nil
}

// buildTektonPipeline 构建Tekton Pipeline对象
func (s *tektonService) buildTektonPipeline(pipeline *models.Pipeline) *tektonv1beta1.Pipeline {
	tasks := make([]tektonv1beta1.PipelineTask, 0, len(pipeline.Tasks))
	
	for _, task := range pipeline.Tasks {
		pipelineTask := tektonv1beta1.PipelineTask{
			Name: task.Name,
			TaskSpec: &tektonv1beta1.EmbeddedTask{
				TaskSpec: tektonv1beta1.TaskSpec{
					Steps: []tektonv1beta1.Step{
						{
							Name:       "main",
							Image:      task.Image,
							Command:    task.Command,
							Args:       task.Args,
							WorkingDir: task.WorkingDir,
						},
					},
				},
			},
		}
		
		// 添加环境变量
		if task.Env != nil {
			var envVars []corev1.EnvVar
			// TODO: 解析task.Env JSON并转换为EnvVar切片
			pipelineTask.TaskSpec.TaskSpec.Steps[0].Env = envVars
		}
		
		// 添加依赖关系
		if len(task.DependsOn) > 0 {
			pipelineTask.RunAfter = task.DependsOn
		}
		
		tasks = append(tasks, pipelineTask)
	}
	
	return &tektonv1beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pipeline-%s", pipeline.ID.String()[:8]),
			Namespace: s.config.Kubernetes.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "euclid-cicd",
				"app.kubernetes.io/component":  "pipeline",
				"euclid.io/pipeline-id":       pipeline.ID.String(),
				"euclid.io/project-id":        pipeline.ProjectID.String(),
			},
		},
		Spec: tektonv1beta1.PipelineSpec{
			Tasks: tasks,
			Workspaces: []tektonv1beta1.PipelineWorkspaceDeclaration{
				{
					Name:        "source",
					Description: "Source code workspace",
				},
			},
		},
	}
}

// CreatePipelineRun 创建Tekton PipelineRun
func (s *tektonService) CreatePipelineRun(ctx context.Context, req *TektonPipelineRunRequest) error {
	pipelineRef := &tektonv1beta1.PipelineRef{
		Name: fmt.Sprintf("pipeline-%s", req.PipelineID.String()[:8]),
	}
	
	// 构建参数
	var params []tektonv1beta1.Param
	for key, value := range req.Parameters {
		params = append(params, tektonv1beta1.Param{
			Name: key,
			Value: tektonv1beta1.ArrayOrString{
				Type:      tektonv1beta1.ParamTypeString,
				StringVal: fmt.Sprintf("%v", value),
			},
		})
	}
	
	// 设置超时时间
	timeout := &metav1.Duration{
		Duration: time.Duration(req.Timeout) * time.Second,
	}
	
	pipelineRun := &tektonv1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: s.config.Kubernetes.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "euclid-cicd",
				"app.kubernetes.io/component": "pipelinerun",
				"euclid.io/pipeline-id":      req.PipelineID.String(),
				"euclid.io/run-id":           req.RunID.String(),
			},
		},
		Spec: tektonv1beta1.PipelineRunSpec{
			PipelineRef: pipelineRef,
			Params:      params,
			Timeout:     timeout,
			Workspaces: []tektonv1beta1.WorkspaceBinding{
				{
					Name: "source",
					VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
		},
	}
	
	// 设置ServiceAccount
	if req.ServiceAccount != "" {
		pipelineRun.Spec.ServiceAccountName = req.ServiceAccount
	} else {
		pipelineRun.Spec.ServiceAccountName = s.config.Kubernetes.ServiceAccount
	}
	
	_, err := s.tektonClient.TektonV1beta1().
		PipelineRuns(s.config.Kubernetes.Namespace).
		Create(ctx, pipelineRun, metav1.CreateOptions{})
	
	if err != nil {
		return fmt.Errorf("创建Tekton PipelineRun失败: %w", err)
	}
	
	return nil
}

// GetPipelineRunStatus 获取PipelineRun状态
func (s *tektonService) GetPipelineRunStatus(ctx context.Context, runID uuid.UUID) (*TektonPipelineRunStatus, error) {
	// 通过Label查找PipelineRun
	labelSelector := fmt.Sprintf("euclid.io/run-id=%s", runID.String())
	
	pipelineRuns, err := s.tektonClient.TektonV1beta1().
		PipelineRuns(s.config.Kubernetes.Namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
	
	if err != nil {
		return nil, fmt.Errorf("获取PipelineRun失败: %w", err)
	}
	
	if len(pipelineRuns.Items) == 0 {
		return nil, fmt.Errorf("PipelineRun不存在")
	}
	
	pipelineRun := pipelineRuns.Items[0]
	
	status := &TektonPipelineRunStatus{
		Status: string(pipelineRun.Status.GetCondition(tektonv1beta1.PipelineRunSucceeded).Type),
	}
	
	if pipelineRun.Status.StartTime != nil {
		status.StartTime = &pipelineRun.Status.StartTime.Time
	}
	
	if pipelineRun.Status.CompletionTime != nil {
		status.EndTime = &pipelineRun.Status.CompletionTime.Time
	}
	
	// 获取TaskRun状态
	if pipelineRun.Status.TaskRuns != nil {
		status.TaskRuns = make(map[string]interface{})
		for name, taskRun := range pipelineRun.Status.TaskRuns {
			status.TaskRuns[name] = map[string]interface{}{
				"status":     string(taskRun.Status.GetCondition(tektonv1beta1.TaskRunSucceeded).Type),
				"start_time": taskRun.Status.StartTime,
				"end_time":   taskRun.Status.CompletionTime,
			}
		}
	}
	
	return status, nil
}

// CancelPipelineRun 取消PipelineRun
func (s *tektonService) CancelPipelineRun(ctx context.Context, runID uuid.UUID) error {
	// 通过Label查找PipelineRun
	labelSelector := fmt.Sprintf("euclid.io/run-id=%s", runID.String())
	
	pipelineRuns, err := s.tektonClient.TektonV1beta1().
		PipelineRuns(s.config.Kubernetes.Namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
	
	if err != nil {
		return fmt.Errorf("获取PipelineRun失败: %w", err)
	}
	
	if len(pipelineRuns.Items) == 0 {
		return fmt.Errorf("PipelineRun不存在")
	}
	
	pipelineRun := &pipelineRuns.Items[0]
	
	// 设置取消状态
	pipelineRun.Spec.Status = tektonv1beta1.PipelineRunSpecStatusCancelled
	
	_, err = s.tektonClient.TektonV1beta1().
		PipelineRuns(s.config.Kubernetes.Namespace).
		Update(ctx, pipelineRun, metav1.UpdateOptions{})
	
	if err != nil {
		return fmt.Errorf("取消PipelineRun失败: %w", err)
	}
	
	return nil
}

// WatchPipelineRuns 监听PipelineRun事件
func (s *tektonService) WatchPipelineRuns(ctx context.Context) error {
	watchlist := cache.NewListWatchFromClient(
		s.tektonClient.TektonV1beta1().RESTClient(),
		"pipelineruns",
		s.config.Kubernetes.Namespace,
		fields.Everything(),
	)
	
	_, controller := cache.NewInformer(
		watchlist,
		&tektonv1beta1.PipelineRun{},
		time.Second*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				s.handlePipelineRunEvent("ADDED", obj)
			},
			DeleteFunc: func(obj interface{}) {
				s.handlePipelineRunEvent("DELETED", obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				s.handlePipelineRunEvent("MODIFIED", newObj)
			},
		},
	)
	
	go controller.Run(ctx.Done())
	return nil
}

// handlePipelineRunEvent 处理PipelineRun事件
func (s *tektonService) handlePipelineRunEvent(eventType string, obj interface{}) {
	pipelineRun, ok := obj.(*tektonv1beta1.PipelineRun)
	if !ok {
		return
	}
	
	// 获取运行ID
	runIDStr, exists := pipelineRun.Labels["euclid.io/run-id"]
	if !exists {
		return
	}
	
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		return
	}
	
	// 更新数据库状态
	var status string
	var message *string
	
	condition := pipelineRun.Status.GetCondition(tektonv1beta1.PipelineRunSucceeded)
	if condition != nil {
		switch condition.Status {
		case corev1.ConditionTrue:
			status = "succeeded"
		case corev1.ConditionFalse:
			status = "failed"
			if condition.Message != "" {
				message = &condition.Message
			}
		default:
			status = "running"
		}
	} else {
		status = "pending"
	}
	
	// 异步更新状态
	go func() {
		if err := s.runService.UpdateStatus(runID, status, message); err != nil {
			fmt.Printf("更新PipelineRun状态失败: %v\n", err)
		}
	}()
}

// CreateSecret 创建Secret
func (s *tektonService) CreateSecret(ctx context.Context, secret *models.Secret) error {
	// TODO: 实现Secret创建逻辑
	return nil
}

// UpdateSecret 更新Secret
func (s *tektonService) UpdateSecret(ctx context.Context, secret *models.Secret) error {
	// TODO: 实现Secret更新逻辑
	return nil
}

// DeleteSecret 删除Secret
func (s *tektonService) DeleteSecret(ctx context.Context, secretID uuid.UUID) error {
	// TODO: 实现Secret删除逻辑
	return nil
}

// HealthCheck 健康检查
func (s *tektonService) HealthCheck(ctx context.Context) error {
	// 检查Kubernetes连接
	_, err := s.k8sClient.CoreV1().Namespaces().Get(ctx, s.config.Kubernetes.Namespace, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Kubernetes连接失败: %w", err)
	}
	
	// 检查Tekton连接
	_, err = s.tektonClient.TektonV1beta1().Pipelines(s.config.Tekton.Namespace).List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("Tekton连接失败: %w", err)
	}
	
	return nil
}