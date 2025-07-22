package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Project 项目模型
type Project struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	TenantID    uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name        string         `json:"name" gorm:"size:255;not null"`
	Key         string         `json:"key" gorm:"size:10;not null"`
	Description *string        `json:"description" gorm:"type:text"`
	ManagerID   *uuid.UUID     `json:"manager_id" gorm:"type:uuid"`
	Status      string         `json:"status" gorm:"size:20;not null;default:active"`
	Settings    datatypes.JSON `json:"settings" gorm:"type:jsonb;default:'{}'"`
	CreatedAt   time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt   *time.Time     `json:"deleted_at,omitempty" gorm:"index"`

	// 关联关系
	Manager *User           `json:"manager,omitempty" gorm:"foreignKey:ManagerID"`
	Members []ProjectMember `json:"members,omitempty" gorm:"foreignKey:ProjectID"`
	Tasks   []Task          `json:"tasks,omitempty" gorm:"foreignKey:ProjectID"`
	Sprints []Sprint        `json:"sprints,omitempty" gorm:"foreignKey:ProjectID"`
}

// User 用户模型 (简化版，主要用于关联)
type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Email     string    `json:"email" gorm:"size:255;uniqueIndex;not null"`
	FullName  *string   `json:"full_name" gorm:"size:255"`
	AvatarURL *string   `json:"avatar_url" gorm:"size:1024"`
}

// ProjectMember 项目成员模型
type ProjectMember struct {
	ProjectID uuid.UUID `json:"project_id" gorm:"type:uuid;primary_key"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;primary_key"`
	RoleID    uuid.UUID `json:"role_id" gorm:"type:uuid;not null"`
	AddedAt   time.Time `json:"added_at" gorm:"not null"`
	AddedBy   uuid.UUID `json:"added_by" gorm:"type:uuid"`

	// 关联关系
	User    *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
}

// Task 任务模型
type Task struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ProjectID      uuid.UUID      `json:"project_id" gorm:"type:uuid;not null;index"`
	TaskNumber     int64          `json:"task_number" gorm:"not null"`
	Title          string         `json:"title" gorm:"size:512;not null"`
	Description    *string        `json:"description" gorm:"type:text"`
	StatusID       *uuid.UUID     `json:"status_id" gorm:"type:uuid"`
	AssigneeID     *uuid.UUID     `json:"assignee_id" gorm:"type:uuid;index"`
	CreatorID      uuid.UUID      `json:"creator_id" gorm:"type:uuid;not null"`
	ParentTaskID   *uuid.UUID     `json:"parent_task_id" gorm:"type:uuid"`
	SprintID       *uuid.UUID     `json:"sprint_id" gorm:"type:uuid;index"`
	DueDate        *time.Time     `json:"due_date"`
	Priority       string         `json:"priority" gorm:"size:20;not null;default:medium"`
	StoryPoints    *int           `json:"story_points"`
	Tags           datatypes.JSON `json:"tags" gorm:"type:jsonb;default:'[]'"`
	CustomFields   datatypes.JSON `json:"custom_fields" gorm:"type:jsonb;default:'{}'"`
	CreatedAt      time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt      time.Time      `json:"updated_at" gorm:"not null"`

	// 关联关系
	Project    *Project    `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Status     *TaskStatus `json:"status,omitempty" gorm:"foreignKey:StatusID"`
	Assignee   *User       `json:"assignee,omitempty" gorm:"foreignKey:AssigneeID"`
	Creator    *User       `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	ParentTask *Task       `json:"parent_task,omitempty" gorm:"foreignKey:ParentTaskID"`
	SubTasks   []Task      `json:"sub_tasks,omitempty" gorm:"foreignKey:ParentTaskID"`
	Sprint     *Sprint     `json:"sprint,omitempty" gorm:"foreignKey:SprintID"`
	Comments   []Comment   `json:"comments,omitempty" gorm:"polymorphic:Parent"`
}

// TaskStatus 任务状态模型
type TaskStatus struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	TenantID     uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name         string    `json:"name" gorm:"size:50;not null"`
	Category     string    `json:"category" gorm:"size:20;not null"` // todo, in_progress, done
	Color        string    `json:"color" gorm:"size:7;default:#6B7280"`
	DisplayOrder int       `json:"display_order" gorm:"not null"`
	IsDefault    bool      `json:"is_default" gorm:"default:false"`
}

// Sprint 迭代模型
type Sprint struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ProjectID   uuid.UUID  `json:"project_id" gorm:"type:uuid;not null;index"`
	Name        string     `json:"name" gorm:"size:255;not null"`
	Goal        *string    `json:"goal" gorm:"type:text"`
	StartDate   time.Time  `json:"start_date" gorm:"not null"`
	EndDate     time.Time  `json:"end_date" gorm:"not null"`
	Status      string     `json:"status" gorm:"size:20;not null;default:planned"` // planned, active, completed
	CreatedAt   time.Time  `json:"created_at" gorm:"not null"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"not null"`

	// 关联关系
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Tasks   []Task   `json:"tasks,omitempty" gorm:"foreignKey:SprintID"`
}

// Comment 评论模型 (多态关联)
type Comment struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	TenantID         uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	AuthorID         uuid.UUID `json:"author_id" gorm:"type:uuid;not null"`
	Content          string    `json:"content" gorm:"type:text;not null"`
	ParentEntityType string    `json:"parent_entity_type" gorm:"size:50;not null"`
	ParentEntityID   uuid.UUID `json:"parent_entity_id" gorm:"type:uuid;not null"`
	CreatedAt        time.Time `json:"created_at" gorm:"not null"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"not null"`

	// 关联关系
	Author *User `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
}

// Milestone 里程碑模型
type Milestone struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ProjectID   uuid.UUID  `json:"project_id" gorm:"type:uuid;not null;index"`
	Title       string     `json:"title" gorm:"size:255;not null"`
	Description *string    `json:"description" gorm:"type:text"`
	DueDate     *time.Time `json:"due_date"`
	Status      string     `json:"status" gorm:"size:20;not null;default:open"` // open, closed
	Progress    int        `json:"progress" gorm:"default:0"`                   // 0-100
	CreatedAt   time.Time  `json:"created_at" gorm:"not null"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"not null"`

	// 关联关系
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
}

// ProjectSettings 项目设置结构
type ProjectSettings struct {
	TaskNumberPrefix    string            `json:"task_number_prefix"`    // 任务编号前缀
	AllowGuestComments  bool              `json:"allow_guest_comments"`  // 允许访客评论
	AutoArchiveSprints  bool              `json:"auto_archive_sprints"`  // 自动归档迭代
	DefaultAssignee     *uuid.UUID        `json:"default_assignee"`      // 默认分配人
	NotificationSettings map[string]bool  `json:"notification_settings"` // 通知设置
	WorkflowSettings    WorkflowSettings  `json:"workflow_settings"`     // 工作流设置
}

// WorkflowSettings 工作流设置
type WorkflowSettings struct {
	AutoMoveToInProgress bool     `json:"auto_move_to_in_progress"` // 开始工作时自动移到进行中
	RequireCommentOnMove bool     `json:"require_comment_on_move"`  // 状态变更时要求评论
	AllowedTransitions   []string `json:"allowed_transitions"`      // 允许的状态转换
}

// BeforeCreate GORM钩子：创建前
func (p *Project) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}

// BeforeCreate GORM钩子：任务创建前
func (t *Task) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return
}

// BeforeCreate GORM钩子：迭代创建前
func (s *Sprint) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}

// TableName 指定表名
func (Project) TableName() string {
	return "projects"
}

func (Task) TableName() string {
	return "tasks"
}

func (TaskStatus) TableName() string {
	return "task_statuses"
}

func (Sprint) TableName() string {
	return "sprints"
}

func (Comment) TableName() string {
	return "comments"
}

func (Milestone) TableName() string {
	return "milestones"
}

func (ProjectMember) TableName() string {
	return "project_members"
}

func (User) TableName() string {
	return "users"
}