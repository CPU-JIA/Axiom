package services

import (
	"fmt"
	"time"

	"cicd-service/internal/config"
	"cicd-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PipelineService 流水线服务接口
type PipelineService interface {
	Create(req *CreatePipelineRequest) (*models.Pipeline, error)
	GetByID(id uuid.UUID) (*models.Pipeline, error)
	GetByProject(projectID uuid.UUID) ([]models.Pipeline, error)
	Update(id uuid.UUID, req *UpdatePipelineRequest) (*models.Pipeline, error)
	Delete(id uuid.UUID) error
	Enable(id uuid.UUID) error
	Disable(id uuid.UUID) error
	Clone(id uuid.UUID, newName string) (*models.Pipeline, error)
	List(req *ListPipelinesRequest) ([]models.Pipeline, int64, error)
	GetStatistics(projectID *uuid.UUID) (*PipelineStats, error)
	ValidateConfig(config interface{}) error
}

type pipelineService struct {
	db     *gorm.DB
	config *config.Config
}

// NewPipelineService 创建流水线服务实例
func NewPipelineService(db *gorm.DB, cfg *config.Config) PipelineService {
	return &pipelineService{
		db:     db,
		config: cfg,
	}
}

// CreatePipelineRequest 创建流水线请求
type CreatePipelineRequest struct {
	ProjectID   uuid.UUID               `json:"project_id" validate:"required"`
	Name        string                  `json:"name" validate:"required,max=255"`
	Description *string                 `json:"description"`
	Config      models.PipelineConfig   `json:"config"`
	Triggers    []TriggerConfig         `json:"triggers"`
	Variables   map[string]interface{}  `json:"variables"`
	Tasks       []CreateTaskRequest     `json:"tasks" validate:"required,min=1"`
}

// UpdatePipelineRequest 更新流水线请求
type UpdatePipelineRequest struct {
	Name        *string                 `json:"name" validate:"omitempty,max=255"`
	Description *string                 `json:"description"`
	Config      *models.PipelineConfig  `json:"config"`
	Triggers    []TriggerConfig         `json:"triggers"`
	Variables   map[string]interface{}  `json:"variables"`
	Tasks       []CreateTaskRequest     `json:"tasks"`
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name        string                 `json:"name" validate:"required,max=255"`
	Description *string                `json:"description"`
	Type        string                 `json:"type" validate:"required,oneof=build test deploy custom"`
	Image       string                 `json:"image" validate:"required"`
	Command     []string               `json:"command"`
	Args        []string               `json:"args"`
	WorkingDir  *string                `json:"working_dir"`
	Env         map[string]string      `json:"env"`
	Volumes     []VolumeMount          `json:"volumes"`
	DependsOn   []string               `json:"depends_on"`
	Condition   *string                `json:"condition"`
	Order       int                    `json:"order"`
	Timeout     int                    `json:"timeout" validate:"min=1,max=7200"`
	Retries     int                    `json:"retries" validate:"min=0,max=5"`
}

// TriggerConfig 触发器配置
type TriggerConfig struct {
	Type       string                 `json:"type" validate:"required,oneof=webhook push tag schedule manual"`
	Conditions map[string]interface{} `json:"conditions"`
	Enabled    bool                   `json:"enabled"`
}

// VolumeMount 卷挂载配置
type VolumeMount struct {
	Name      string `json:"name" validate:"required"`
	MountPath string `json:"mount_path" validate:"required"`
	SubPath   string `json:"sub_path"`
	ReadOnly  bool   `json:"read_only"`
}

// ListPipelinesRequest 列表查询请求
type ListPipelinesRequest struct {
	ProjectID *uuid.UUID `json:"project_id"`
	Status    *string    `json:"status"`
	Search    *string    `json:"search"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
	SortBy    string     `json:"sort_by"`
	SortDesc  bool       `json:"sort_desc"`
}

// PipelineStats 流水线统计信息
type PipelineStats struct {
	TotalPipelines    int64                     `json:"total_pipelines"`
	ActivePipelines   int64                     `json:"active_pipelines"`
	DisabledPipelines int64                     `json:"disabled_pipelines"`
	TotalRuns         int64                     `json:"total_runs"`
	SuccessfulRuns    int64                     `json:"successful_runs"`
	FailedRuns        int64                     `json:"failed_runs"`
	SuccessRate       float64                   `json:"success_rate"`
	AverageRunTime    float64                   `json:"average_run_time"`
	RunsByStatus      map[string]int64          `json:"runs_by_status"`
	RunsByTrigger     map[string]int64          `json:"runs_by_trigger"`
	RecentActivity    []PipelineActivity        `json:"recent_activity"`
}

// PipelineActivity 流水线活动
type PipelineActivity struct {
	PipelineID   uuid.UUID `json:"pipeline_id"`
	PipelineName string    `json:"pipeline_name"`
	RunID        uuid.UUID `json:"run_id"`
	RunNumber    int       `json:"run_number"`
	Status       string    `json:"status"`
	TriggerType  string    `json:"trigger_type"`
	StartedAt    time.Time `json:"started_at"`
	Duration     *int      `json:"duration"`
}

// Create 创建流水线
func (s *pipelineService) Create(req *CreatePipelineRequest) (*models.Pipeline, error) {
	// 检查项目是否存在
	var project models.Project
	if err := s.db.Where("id = ?", req.ProjectID).First(&project).Error; err != nil {
		return nil, fmt.Errorf("项目不存在")
	}

	// 检查同项目下流水线名称唯一性
	var existingPipeline models.Pipeline
	if err := s.db.Where("project_id = ? AND name = ? AND deleted_at IS NULL", 
		req.ProjectID, req.Name).First(&existingPipeline).Error; err == nil {
		return nil, fmt.Errorf("流水线名称 '%s' 已存在", req.Name)
	}

	// 创建流水线事务
	return s.createPipelineWithTasks(req)
}

// createPipelineWithTasks 创建流水线及任务（事务）
func (s *pipelineService) createPipelineWithTasks(req *CreatePipelineRequest) (*models.Pipeline, error) {
	var pipeline *models.Pipeline
	
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 创建流水线
		pipeline = &models.Pipeline{
			ProjectID:   req.ProjectID,
			Name:        req.Name,
			Description: req.Description,
			Status:      "active",
			Config:      req.Config,
		}

		// 处理触发器
		if req.Triggers != nil {
			triggersJSON, err := jsonMarshal(req.Triggers)
			if err != nil {
				return fmt.Errorf("序列化触发器配置失败: %w", err)
			}
			pipeline.Triggers = triggersJSON
		}

		// 处理变量
		if req.Variables != nil {
			variablesJSON, err := jsonMarshal(req.Variables)
			if err != nil {
				return fmt.Errorf("序列化变量配置失败: %w", err)
			}
			pipeline.Variables = variablesJSON
		}

		if err := tx.Create(pipeline).Error; err != nil {
			return fmt.Errorf("创建流水线失败: %w", err)
		}

		// 创建任务
		for i, taskReq := range req.Tasks {
			task := &models.Task{
				PipelineID:  pipeline.ID,
				Name:        taskReq.Name,
				Description: taskReq.Description,
				Type:        taskReq.Type,
				Image:       taskReq.Image,
				Command:     taskReq.Command,
				Args:        taskReq.Args,
				WorkingDir:  taskReq.WorkingDir,
				DependsOn:   taskReq.DependsOn,
				Condition:   taskReq.Condition,
				Order:       taskReq.Order,
				Timeout:     taskReq.Timeout,
				Retries:     taskReq.Retries,
			}

			// 处理环境变量
			if taskReq.Env != nil {
				envJSON, err := jsonMarshal(taskReq.Env)
				if err != nil {
					return fmt.Errorf("序列化任务环境变量失败: %w", err)
				}
				task.Env = envJSON
			}

			// 处理卷挂载
			if taskReq.Volumes != nil {
				volumesJSON, err := jsonMarshal(taskReq.Volumes)
				if err != nil {
					return fmt.Errorf("序列化任务卷挂载失败: %w", err)
				}
				task.Volumes = volumesJSON
			}

			// 设置默认超时时间
			if task.Timeout == 0 {
				task.Timeout = s.config.Tekton.DefaultTimeout
			}

			// 设置默认顺序
			if task.Order == 0 {
				task.Order = i + 1
			}

			if err := tx.Create(task).Error; err != nil {
				return fmt.Errorf("创建任务失败: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 重新加载完整数据
	return s.GetByID(pipeline.ID)
}

// GetByID 根据ID获取流水线
func (s *pipelineService) GetByID(id uuid.UUID) (*models.Pipeline, error) {
	var pipeline models.Pipeline
	if err := s.db.Where("id = ? AND deleted_at IS NULL", id).
		Preload("Tasks", func(db *gorm.DB) *gorm.DB {
			return db.Order("\"order\" ASC, created_at ASC")
		}).
		Preload("Project").
		First(&pipeline).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("流水线不存在")
		}
		return nil, fmt.Errorf("获取流水线失败: %w", err)
	}
	return &pipeline, nil
}

// GetByProject 获取项目的所有流水线
func (s *pipelineService) GetByProject(projectID uuid.UUID) ([]models.Pipeline, error) {
	var pipelines []models.Pipeline
	if err := s.db.Where("project_id = ? AND deleted_at IS NULL", projectID).
		Order("created_at DESC").Find(&pipelines).Error; err != nil {
		return nil, fmt.Errorf("获取项目流水线失败: %w", err)
	}
	return pipelines, nil
}

// Update 更新流水线
func (s *pipelineService) Update(id uuid.UUID, req *UpdatePipelineRequest) (*models.Pipeline, error) {
	pipeline, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	return s.updatePipelineWithTasks(pipeline, req)
}

// updatePipelineWithTasks 更新流水线及任务（事务）
func (s *pipelineService) updatePipelineWithTasks(pipeline *models.Pipeline, req *UpdatePipelineRequest) (*models.Pipeline, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		updates := make(map[string]interface{})
		
		if req.Name != nil {
			// 检查名称唯一性
			var existingPipeline models.Pipeline
			if err := tx.Where("project_id = ? AND name = ? AND id != ? AND deleted_at IS NULL", 
				pipeline.ProjectID, *req.Name, pipeline.ID).First(&existingPipeline).Error; err == nil {
				return fmt.Errorf("流水线名称 '%s' 已存在", *req.Name)
			}
			updates["name"] = *req.Name
		}
		
		if req.Description != nil {
			updates["description"] = *req.Description
		}
		
		if req.Config != nil {
			updates["config"] = *req.Config
		}

		if req.Triggers != nil {
			triggersJSON, err := jsonMarshal(req.Triggers)
			if err != nil {
				return fmt.Errorf("序列化触发器配置失败: %w", err)
			}
			updates["triggers"] = triggersJSON
		}

		if req.Variables != nil {
			variablesJSON, err := jsonMarshal(req.Variables)
			if err != nil {
				return fmt.Errorf("序列化变量配置失败: %w", err)
			}
			updates["variables"] = variablesJSON
		}

		updates["updated_at"] = time.Now()

		if err := tx.Model(pipeline).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新流水线失败: %w", err)
		}

		// 如果更新了任务
		if req.Tasks != nil {
			// 删除现有任务
			if err := tx.Where("pipeline_id = ?", pipeline.ID).Delete(&models.Task{}).Error; err != nil {
				return fmt.Errorf("删除现有任务失败: %w", err)
			}

			// 创建新任务
			for i, taskReq := range req.Tasks {
				task := &models.Task{
					PipelineID:  pipeline.ID,
					Name:        taskReq.Name,
					Description: taskReq.Description,
					Type:        taskReq.Type,
					Image:       taskReq.Image,
					Command:     taskReq.Command,
					Args:        taskReq.Args,
					WorkingDir:  taskReq.WorkingDir,
					DependsOn:   taskReq.DependsOn,
					Condition:   taskReq.Condition,
					Order:       taskReq.Order,
					Timeout:     taskReq.Timeout,
					Retries:     taskReq.Retries,
				}

				// 处理环境变量
				if taskReq.Env != nil {
					envJSON, err := jsonMarshal(taskReq.Env)
					if err != nil {
						return fmt.Errorf("序列化任务环境变量失败: %w", err)
					}
					task.Env = envJSON
				}

				// 处理卷挂载
				if taskReq.Volumes != nil {
					volumesJSON, err := jsonMarshal(taskReq.Volumes)
					if err != nil {
						return fmt.Errorf("序列化任务卷挂载失败: %w", err)
					}
					task.Volumes = volumesJSON
				}

				// 设置默认值
				if task.Timeout == 0 {
					task.Timeout = s.config.Tekton.DefaultTimeout
				}
				if task.Order == 0 {
					task.Order = i + 1
				}

				if err := tx.Create(task).Error; err != nil {
					return fmt.Errorf("创建任务失败: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetByID(pipeline.ID)
}

// Delete 删除流水线（软删除）
func (s *pipelineService) Delete(id uuid.UUID) error {
	pipeline, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// 检查是否有运行中的流水线
	var runningCount int64
	if err := s.db.Model(&models.PipelineRun{}).
		Where("pipeline_id = ? AND status IN (?)", id, []string{"pending", "running"}).
		Count(&runningCount).Error; err != nil {
		return fmt.Errorf("检查运行状态失败: %w", err)
	}

	if runningCount > 0 {
		return fmt.Errorf("存在运行中的流水线，无法删除")
	}

	// 软删除
	now := time.Now()
	if err := s.db.Model(pipeline).Update("deleted_at", now).Error; err != nil {
		return fmt.Errorf("删除流水线失败: %w", err)
	}

	return nil
}

// Enable 启用流水线
func (s *pipelineService) Enable(id uuid.UUID) error {
	return s.updateStatus(id, "active")
}

// Disable 禁用流水线
func (s *pipelineService) Disable(id uuid.UUID) error {
	return s.updateStatus(id, "disabled")
}

// updateStatus 更新流水线状态
func (s *pipelineService) updateStatus(id uuid.UUID, status string) error {
	pipeline, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.db.Model(pipeline).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("更新流水线状态失败: %w", err)
	}

	return nil
}

// Clone 克隆流水线
func (s *pipelineService) Clone(id uuid.UUID, newName string) (*models.Pipeline, error) {
	// 获取原流水线
	original, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 检查新名称唯一性
	var existingPipeline models.Pipeline
	if err := s.db.Where("project_id = ? AND name = ? AND deleted_at IS NULL", 
		original.ProjectID, newName).First(&existingPipeline).Error; err == nil {
		return nil, fmt.Errorf("流水线名称 '%s' 已存在", newName)
	}

	// 创建克隆请求
	var triggers []TriggerConfig
	if err := jsonUnmarshal(original.Triggers, &triggers); err == nil {
		// 忽略反序列化错误
	}

	var variables map[string]interface{}
	if err := jsonUnmarshal(original.Variables, &variables); err == nil {
		// 忽略反序列化错误
	}

	var tasks []CreateTaskRequest
	for _, task := range original.Tasks {
		taskReq := CreateTaskRequest{
			Name:        task.Name,
			Description: task.Description,
			Type:        task.Type,
			Image:       task.Image,
			Command:     task.Command,
			Args:        task.Args,
			WorkingDir:  task.WorkingDir,
			DependsOn:   task.DependsOn,
			Condition:   task.Condition,
			Order:       task.Order,
			Timeout:     task.Timeout,
			Retries:     task.Retries,
		}

		// 反序列化环境变量和卷挂载
		var env map[string]string
		if err := jsonUnmarshal(task.Env, &env); err == nil {
			taskReq.Env = env
		}

		var volumes []VolumeMount
		if err := jsonUnmarshal(task.Volumes, &volumes); err == nil {
			taskReq.Volumes = volumes
		}

		tasks = append(tasks, taskReq)
	}

	cloneReq := &CreatePipelineRequest{
		ProjectID:   original.ProjectID,
		Name:        newName,
		Description: original.Description,
		Config:      original.Config,
		Triggers:    triggers,
		Variables:   variables,
		Tasks:       tasks,
	}

	return s.createPipelineWithTasks(cloneReq)
}

// List 列表查询流水线
func (s *pipelineService) List(req *ListPipelinesRequest) ([]models.Pipeline, int64, error) {
	query := s.db.Model(&models.Pipeline{}).Where("deleted_at IS NULL")

	// 应用筛选条件
	if req.ProjectID != nil {
		query = query.Where("project_id = ?", *req.ProjectID)
	}
	
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	
	if req.Search != nil {
		searchTerm := fmt.Sprintf("%%%s%%", *req.Search)
		query = query.Where("name ILIKE ? OR description ILIKE ?", searchTerm, searchTerm)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计流水线总数失败: %w", err)
	}

	// 应用排序
	sortBy := "created_at"
	if req.SortBy != "" {
		sortBy = req.SortBy
	}
	
	if req.SortDesc {
		sortBy += " DESC"
	} else {
		sortBy += " ASC"
	}
	query = query.Order(sortBy)

	// 应用分页
	if req.Page > 0 && req.Limit > 0 {
		offset := (req.Page - 1) * req.Limit
		query = query.Offset(offset).Limit(req.Limit)
	}

	var pipelines []models.Pipeline
	if err := query.Preload("Project").Find(&pipelines).Error; err != nil {
		return nil, 0, fmt.Errorf("查询流水线列表失败: %w", err)
	}

	return pipelines, total, nil
}

// GetStatistics 获取流水线统计信息
func (s *pipelineService) GetStatistics(projectID *uuid.UUID) (*PipelineStats, error) {
	stats := &PipelineStats{}

	query := s.db.Model(&models.Pipeline{}).Where("deleted_at IS NULL")
	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	}

	// 基础统计
	if err := query.Count(&stats.TotalPipelines).Error; err != nil {
		return nil, fmt.Errorf("统计流水线总数失败: %w", err)
	}

	if err := query.Where("status = ?", "active").Count(&stats.ActivePipelines).Error; err != nil {
		return nil, fmt.Errorf("统计活跃流水线失败: %w", err)
	}

	if err := query.Where("status = ?", "disabled").Count(&stats.DisabledPipelines).Error; err != nil {
		return nil, fmt.Errorf("统计禁用流水线失败: %w", err)
	}

	// 运行统计
	runQuery := s.db.Model(&models.PipelineRun{})
	if projectID != nil {
		runQuery = runQuery.Joins("JOIN pipelines ON pipeline_runs.pipeline_id = pipelines.id").
			Where("pipelines.project_id = ? AND pipelines.deleted_at IS NULL", *projectID)
	}

	if err := runQuery.Count(&stats.TotalRuns).Error; err != nil {
		return nil, fmt.Errorf("统计总运行次数失败: %w", err)
	}

	if err := runQuery.Where("status = ?", "succeeded").Count(&stats.SuccessfulRuns).Error; err != nil {
		return nil, fmt.Errorf("统计成功运行次数失败: %w", err)
	}

	if err := runQuery.Where("status = ?", "failed").Count(&stats.FailedRuns).Error; err != nil {
		return nil, fmt.Errorf("统计失败运行次数失败: %w", err)
	}

	// 计算成功率
	if stats.TotalRuns > 0 {
		stats.SuccessRate = float64(stats.SuccessfulRuns) / float64(stats.TotalRuns) * 100
	}

	// 计算平均运行时间
	var avgDuration float64
	if err := runQuery.Where("status = ? AND duration IS NOT NULL", "succeeded").
		Select("AVG(duration)").Scan(&avgDuration).Error; err != nil {
		return nil, fmt.Errorf("计算平均运行时间失败: %w", err)
	}
	stats.AverageRunTime = avgDuration

	// 按状态分组统计
	stats.RunsByStatus = make(map[string]int64)
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	if err := runQuery.Select("status, COUNT(*) as count").
		Group("status").Scan(&statusStats).Error; err != nil {
		return nil, fmt.Errorf("按状态统计失败: %w", err)
	}
	for _, stat := range statusStats {
		stats.RunsByStatus[stat.Status] = stat.Count
	}

	// 按触发器分组统计
	stats.RunsByTrigger = make(map[string]int64)
	var triggerStats []struct {
		TriggerType string `json:"trigger_type"`
		Count       int64  `json:"count"`
	}
	if err := runQuery.Select("trigger_type, COUNT(*) as count").
		Group("trigger_type").Scan(&triggerStats).Error; err != nil {
		return nil, fmt.Errorf("按触发器统计失败: %w", err)
	}
	for _, stat := range triggerStats {
		stats.RunsByTrigger[stat.TriggerType] = stat.Count
	}

	// 最近活动
	var activities []PipelineActivity
	activityQuery := s.db.Table("pipeline_runs").
		Select("pipeline_runs.pipeline_id, pipelines.name as pipeline_name, pipeline_runs.id as run_id, pipeline_runs.run_number, pipeline_runs.status, pipeline_runs.trigger_type, pipeline_runs.started_at, pipeline_runs.duration").
		Joins("JOIN pipelines ON pipeline_runs.pipeline_id = pipelines.id").
		Where("pipelines.deleted_at IS NULL")

	if projectID != nil {
		activityQuery = activityQuery.Where("pipelines.project_id = ?", *projectID)
	}

	if err := activityQuery.Order("pipeline_runs.created_at DESC").
		Limit(10).Scan(&activities).Error; err != nil {
		return nil, fmt.Errorf("获取最近活动失败: %w", err)
	}
	stats.RecentActivity = activities

	return stats, nil
}

// ValidateConfig 验证配置
func (s *pipelineService) ValidateConfig(config interface{}) error {
	// TODO: 实现配置验证逻辑
	return nil
}

// 辅助函数
func jsonMarshal(v interface{}) ([]byte, error) {
	// 这里应该使用实际的JSON序列化
	// 为了简化示例，返回空字节切片
	return []byte("{}"), nil
}

func jsonUnmarshal(data []byte, v interface{}) error {
	// 这里应该使用实际的JSON反序列化
	// 为了简化示例，直接返回nil
	return nil
}