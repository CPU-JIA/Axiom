package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Pipeline 流水线模型
type Pipeline struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ProjectID    uuid.UUID      `json:"project_id" gorm:"type:uuid;not null;index"`
	Name         string         `json:"name" gorm:"size:255;not null"`
	Description  *string        `json:"description" gorm:"type:text"`
	Status       string         `json:"status" gorm:"size:20;not null;default:active"` // active, disabled, archived
	Config       PipelineConfig `json:"config" gorm:"embedded"`
	Triggers     datatypes.JSON `json:"triggers" gorm:"type:jsonb;default:'[]'"`        // 触发器配置
	Variables    datatypes.JSON `json:"variables" gorm:"type:jsonb;default:'{}'"`       // 管道变量
	LastRunID    *uuid.UUID     `json:"last_run_id" gorm:"type:uuid"`
	LastRunAt    *time.Time     `json:"last_run_at"`
	CreatedAt    time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt    *time.Time     `json:"deleted_at,omitempty" gorm:"index"`

	// 关联关系
	Project     *Project       `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	PipelineRuns []PipelineRun `json:"pipeline_runs,omitempty" gorm:"foreignKey:PipelineID"`
	Tasks        []Task        `json:"tasks,omitempty" gorm:"foreignKey:PipelineID"`
}

// PipelineConfig 流水线配置
type PipelineConfig struct {
	Timeout         int      `json:"timeout" gorm:"default:3600"`                    // 超时时间(秒)
	Retries         int      `json:"retries" gorm:"default:0"`                       // 重试次数
	Workspace       string   `json:"workspace" gorm:"size:255;default:/workspace"`   // 工作空间路径
	ServiceAccount  string   `json:"service_account" gorm:"size:255"`                // K8s服务账户
	NodeSelector    string   `json:"node_selector" gorm:"size:500"`                  // 节点选择器
	ResourceLimits  string   `json:"resource_limits" gorm:"size:500"`                // 资源限制
	EnableCache     bool     `json:"enable_cache" gorm:"default:true"`               // 启用缓存
	CacheKeys       []string `json:"cache_keys" gorm:"type:text[]"`                  // 缓存键
	NotificationChannels []string `json:"notification_channels" gorm:"type:text[]"` // 通知渠道
}

// Task 任务模型
type Task struct {
	ID           uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	PipelineID   uuid.UUID    `json:"pipeline_id" gorm:"type:uuid;not null;index"`
	Name         string       `json:"name" gorm:"size:255;not null"`
	Description  *string      `json:"description" gorm:"type:text"`
	Type         string       `json:"type" gorm:"size:50;not null"`           // build, test, deploy, custom
	Image        string       `json:"image" gorm:"size:255;not null"`         // 容器镜像
	Command      []string     `json:"command" gorm:"type:text[]"`             // 执行命令
	Args         []string     `json:"args" gorm:"type:text[]"`                // 命令参数
	WorkingDir   *string      `json:"working_dir" gorm:"size:255"`            // 工作目录
	Env          datatypes.JSON `json:"env" gorm:"type:jsonb;default:'{}'"`   // 环境变量
	Volumes      datatypes.JSON `json:"volumes" gorm:"type:jsonb;default:'[]'"` // 挂载卷
	DependsOn    []string     `json:"depends_on" gorm:"type:text[]"`          // 依赖任务
	Condition    *string      `json:"condition" gorm:"size:255"`              // 执行条件
	Order        int          `json:"order" gorm:"not null;default:0"`        // 执行顺序
	Timeout      int          `json:"timeout" gorm:"default:1800"`            // 任务超时(秒)
	Retries      int          `json:"retries" gorm:"default:0"`               // 重试次数
	CreatedAt    time.Time    `json:"created_at" gorm:"not null"`
	UpdatedAt    time.Time    `json:"updated_at" gorm:"not null"`

	// 关联关系
	Pipeline *Pipeline `json:"pipeline,omitempty" gorm:"foreignKey:PipelineID"`
	TaskRuns []TaskRun `json:"task_runs,omitempty" gorm:"foreignKey:TaskID"`
}

// PipelineRun 流水线运行记录
type PipelineRun struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	PipelineID    uuid.UUID      `json:"pipeline_id" gorm:"type:uuid;not null;index"`
	RunNumber     int            `json:"run_number" gorm:"not null"`                     // 运行序号
	Status        string         `json:"status" gorm:"size:20;not null;default:pending"` // pending, running, succeeded, failed, cancelled, timeout
	TriggerType   string         `json:"trigger_type" gorm:"size:50;not null"`          // manual, webhook, schedule, api
	TriggerBy     *uuid.UUID     `json:"trigger_by" gorm:"type:uuid"`                   // 触发用户
	TriggerData   datatypes.JSON `json:"trigger_data" gorm:"type:jsonb;default:'{}'"`   // 触发数据
	StartedAt     *time.Time     `json:"started_at"`
	FinishedAt    *time.Time     `json:"finished_at"`
	Duration      *int           `json:"duration"`                                       // 执行时长(秒)
	LogsPath      *string        `json:"logs_path" gorm:"size:512"`                     // 日志文件路径
	ArtifactsPath *string        `json:"artifacts_path" gorm:"size:512"`                // 产物路径
	ResourceUsage datatypes.JSON `json:"resource_usage" gorm:"type:jsonb;default:'{}'"`// 资源使用统计
	ErrorMessage  *string        `json:"error_message" gorm:"type:text"`                // 错误信息
	CreatedAt     time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"not null"`

	// 关联关系
	Pipeline *Pipeline `json:"pipeline,omitempty" gorm:"foreignKey:PipelineID"`
	TaskRuns []TaskRun `json:"task_runs,omitempty" gorm:"foreignKey:PipelineRunID"`
}

// TaskRun 任务运行记录
type TaskRun struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	PipelineRunID uuid.UUID      `json:"pipeline_run_id" gorm:"type:uuid;not null;index"`
	TaskID        uuid.UUID      `json:"task_id" gorm:"type:uuid;not null;index"`
	Name          string         `json:"name" gorm:"size:255;not null"`
	Status        string         `json:"status" gorm:"size:20;not null;default:pending"` // pending, running, succeeded, failed, cancelled, skipped
	StartedAt     *time.Time     `json:"started_at"`
	FinishedAt    *time.Time     `json:"finished_at"`
	Duration      *int           `json:"duration"`                                       // 执行时长(秒)
	ExitCode      *int           `json:"exit_code"`                                      // 退出码
	LogsPath      *string        `json:"logs_path" gorm:"size:512"`                     // 日志文件路径
	PodName       *string        `json:"pod_name" gorm:"size:255"`                      // K8s Pod名称
	NodeName      *string        `json:"node_name" gorm:"size:255"`                     // K8s Node名称
	ResourceUsage datatypes.JSON `json:"resource_usage" gorm:"type:jsonb;default:'{}'"`// 资源使用统计
	ErrorMessage  *string        `json:"error_message" gorm:"type:text"`                // 错误信息
	RetryCount    int            `json:"retry_count" gorm:"default:0"`                  // 重试次数
	CreatedAt     time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"not null"`

	// 关联关系
	PipelineRun *PipelineRun `json:"pipeline_run,omitempty" gorm:"foreignKey:PipelineRunID"`
	Task        *Task        `json:"task,omitempty" gorm:"foreignKey:TaskID"`
}

// BuildCache 构建缓存模型
type BuildCache struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ProjectID   uuid.UUID  `json:"project_id" gorm:"type:uuid;not null;index"`
	Key         string     `json:"key" gorm:"size:255;not null;uniqueIndex"`         // 缓存键
	Path        string     `json:"path" gorm:"size:512;not null"`                    // 缓存路径
	Size        int64      `json:"size" gorm:"default:0"`                            // 缓存大小(bytes)
	HitCount    int        `json:"hit_count" gorm:"default:0"`                       // 命中次数
	Checksum    string     `json:"checksum" gorm:"size:64"`                          // 校验和
	Metadata    datatypes.JSON `json:"metadata" gorm:"type:jsonb;default:'{}'"`      // 元数据
	ExpiresAt   *time.Time `json:"expires_at"`                                       // 过期时间
	LastUsedAt  *time.Time `json:"last_used_at"`                                     // 最后使用时间
	CreatedAt   time.Time  `json:"created_at" gorm:"not null"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"not null"`
}

// Secret 密钥管理模型
type Secret struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ProjectID uuid.UUID      `json:"project_id" gorm:"type:uuid;not null;index"`
	Name      string         `json:"name" gorm:"size:255;not null"`
	Type      string         `json:"type" gorm:"size:50;not null"`                     // generic, docker-registry, ssh-auth
	Data      datatypes.JSON `json:"data" gorm:"type:jsonb;not null"`                  // 加密数据
	UsageCount int           `json:"usage_count" gorm:"default:0"`                     // 使用次数
	LastUsedAt *time.Time    `json:"last_used_at"`                                     // 最后使用时间
	CreatedBy  uuid.UUID     `json:"created_by" gorm:"type:uuid;not null"`
	CreatedAt  time.Time     `json:"created_at" gorm:"not null"`
	UpdatedAt  time.Time     `json:"updated_at" gorm:"not null"`
}

// Environment 环境管理模型
type Environment struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ProjectID    uuid.UUID      `json:"project_id" gorm:"type:uuid;not null;index"`
	Name         string         `json:"name" gorm:"size:255;not null"`
	Type         string         `json:"type" gorm:"size:50;not null"`                     // development, staging, production
	Status       string         `json:"status" gorm:"size:20;not null;default:active"`    // active, inactive, maintenance
	Namespace    string         `json:"namespace" gorm:"size:255;not null"`               // K8s命名空间
	Config       datatypes.JSON `json:"config" gorm:"type:jsonb;default:'{}'"`            // 环境配置
	Secrets      []string       `json:"secrets" gorm:"type:text[]"`                       // 关联密钥
	Variables    datatypes.JSON `json:"variables" gorm:"type:jsonb;default:'{}'"`         // 环境变量
	ProtectionRules datatypes.JSON `json:"protection_rules" gorm:"type:jsonb;default:'{}'"` // 保护规则
	CreatedBy    uuid.UUID      `json:"created_by" gorm:"type:uuid;not null"`
	CreatedAt    time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt    *time.Time     `json:"deleted_at,omitempty" gorm:"index"`
}

// Project 项目模型 (简化版)
type Project struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Name     string    `json:"name" gorm:"size:255;not null"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null"`
}

// User 用户模型 (简化版)
type User struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Email    string    `json:"email" gorm:"size:255;uniqueIndex;not null"`
	FullName *string   `json:"full_name" gorm:"size:255"`
}

// GORM钩子：创建前
func (p *Pipeline) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}

func (t *Task) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return
}

func (pr *PipelineRun) BeforeCreate(tx *gorm.DB) (err error) {
	if pr.ID == uuid.Nil {
		pr.ID = uuid.New()
	}
	// 自动生成运行序号
	if pr.RunNumber == 0 {
		var maxRunNumber int
		tx.Model(&PipelineRun{}).Where("pipeline_id = ?", pr.PipelineID).
			Select("COALESCE(MAX(run_number), 0)").Scan(&maxRunNumber)
		pr.RunNumber = maxRunNumber + 1
	}
	return
}

func (tr *TaskRun) BeforeCreate(tx *gorm.DB) (err error) {
	if tr.ID == uuid.Nil {
		tr.ID = uuid.New()
	}
	return
}

func (bc *BuildCache) BeforeCreate(tx *gorm.DB) (err error) {
	if bc.ID == uuid.Nil {
		bc.ID = uuid.New()
	}
	return
}

func (s *Secret) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}

func (e *Environment) BeforeCreate(tx *gorm.DB) (err error) {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return
}

// 表名指定
func (Pipeline) TableName() string {
	return "pipelines"
}

func (Task) TableName() string {
	return "tasks"
}

func (PipelineRun) TableName() string {
	return "pipeline_runs"
}

func (TaskRun) TableName() string {
	return "task_runs"
}

func (BuildCache) TableName() string {
	return "build_caches"
}

func (Secret) TableName() string {
	return "secrets"
}

func (Environment) TableName() string {
	return "environments"
}

func (Project) TableName() string {
	return "projects"
}

func (User) TableName() string {
	return "users"
}