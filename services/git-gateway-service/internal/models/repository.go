package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Repository 代码仓库模型
type Repository struct {
	ID               uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ProjectID        uuid.UUID       `json:"project_id" gorm:"type:uuid;not null;index"`
	Name             string          `json:"name" gorm:"size:255;not null"`
	Description      *string         `json:"description" gorm:"type:text"`
	Visibility       string          `json:"visibility" gorm:"size:20;not null;default:private"` // private, internal, public
	DefaultBranch    string          `json:"default_branch" gorm:"size:255;not null;default:main"`
	GitURL           string          `json:"git_url" gorm:"size:512;not null"`
	HTTPURL          string          `json:"http_url" gorm:"size:512;not null"`
	SSHURL           string          `json:"ssh_url" gorm:"size:512;not null"`
	Size             int64           `json:"size" gorm:"default:0"`                                  // 仓库大小 (bytes)
	CommitCount      int             `json:"commit_count" gorm:"default:0"`                          // 提交数量
	BranchCount      int             `json:"branch_count" gorm:"default:1"`                          // 分支数量
	TagCount         int             `json:"tag_count" gorm:"default:0"`                             // 标签数量
	Language         *string         `json:"language" gorm:"size:50"`                                // 主要语言
	Topics           datatypes.JSON  `json:"topics" gorm:"type:jsonb;default:'[]'"`                  // 主题标签
	Settings         RepositorySettings `json:"settings" gorm:"embedded"`                            // 仓库设置
	LastActivityAt   *time.Time      `json:"last_activity_at"`                                       // 最后活动时间
	CreatedAt        time.Time       `json:"created_at" gorm:"not null"`
	UpdatedAt        time.Time       `json:"updated_at" gorm:"not null"`
	DeletedAt        *time.Time      `json:"deleted_at,omitempty" gorm:"index"`

	// 关联关系
	Project         *Project         `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Branches        []Branch         `json:"branches,omitempty" gorm:"foreignKey:RepositoryID"`
	Tags            []Tag            `json:"tags,omitempty" gorm:"foreignKey:RepositoryID"`
	Webhooks        []Webhook        `json:"webhooks,omitempty" gorm:"foreignKey:RepositoryID"`
	PushEvents      []PushEvent      `json:"push_events,omitempty" gorm:"foreignKey:RepositoryID"`
	AccessKeys      []AccessKey      `json:"access_keys,omitempty" gorm:"foreignKey:RepositoryID"`
}

// RepositorySettings 仓库设置
type RepositorySettings struct {
	AllowPush           bool                `json:"allow_push" gorm:"default:true"`
	AllowForcePush      bool                `json:"allow_force_push" gorm:"default:false"`
	AllowDeletions      bool                `json:"allow_deletions" gorm:"default:false"`
	RequireSignedCommits bool                `json:"require_signed_commits" gorm:"default:false"`
	EnableLFS           bool                `json:"enable_lfs" gorm:"default:true"`
	EnableIssues        bool                `json:"enable_issues" gorm:"default:true"`
	EnableWiki          bool                `json:"enable_wiki" gorm:"default:true"`
	AutoDeleteBranch    bool                `json:"auto_delete_branch" gorm:"default:false"`
	DefaultMergeMethod  string              `json:"default_merge_method" gorm:"size:20;default:merge"` // merge, squash, rebase
}

// Project 项目模型 (简化版)
type Project struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Name     string    `json:"name" gorm:"size:255;not null"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null"`
}

// Branch 分支模型
type Branch struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	RepositoryID uuid.UUID  `json:"repository_id" gorm:"type:uuid;not null;index"`
	Name         string     `json:"name" gorm:"size:255;not null"`
	CommitSHA    string     `json:"commit_sha" gorm:"size:40;not null"`
	IsDefault    bool       `json:"is_default" gorm:"default:false"`
	IsProtected  bool       `json:"is_protected" gorm:"default:false"`
	Protection   BranchProtection `json:"protection" gorm:"embedded"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"not null"`

	// 关联关系
	Repository *Repository `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
}

// BranchProtection 分支保护设置
type BranchProtection struct {
	RequireStatusChecks       bool `json:"require_status_checks" gorm:"default:false"`
	RequireUpToDate          bool `json:"require_up_to_date" gorm:"default:false"`
	RequirePullRequest       bool `json:"require_pull_request" gorm:"default:false"`
	RequireCodeOwnerReviews  bool `json:"require_code_owner_reviews" gorm:"default:false"`
	DismissStaleReviews      bool `json:"dismiss_stale_reviews" gorm:"default:false"`
	RequiredReviewers        int  `json:"required_reviewers" gorm:"default:1"`
	RestrictPushes           bool `json:"restrict_pushes" gorm:"default:false"`
	AllowForcePushes         bool `json:"allow_force_pushes" gorm:"default:false"`
	AllowDeletions           bool `json:"allow_deletions" gorm:"default:false"`
}

// Tag 标签模型
type Tag struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	RepositoryID uuid.UUID  `json:"repository_id" gorm:"type:uuid;not null;index"`
	Name         string     `json:"name" gorm:"size:255;not null"`
	CommitSHA    string     `json:"commit_sha" gorm:"size:40;not null"`
	Message      *string    `json:"message" gorm:"type:text"`
	TaggerName   *string    `json:"tagger_name" gorm:"size:255"`
	TaggerEmail  *string    `json:"tagger_email" gorm:"size:255"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null"`

	// 关联关系
	Repository *Repository `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
}

// Webhook 网络钩子模型
type Webhook struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	RepositoryID uuid.UUID      `json:"repository_id" gorm:"type:uuid;not null;index"`
	URL          string         `json:"url" gorm:"size:1024;not null"`
	Secret       *string        `json:"secret,omitempty" gorm:"size:255"`
	ContentType  string         `json:"content_type" gorm:"size:50;not null;default:application/json"`
	Events       datatypes.JSON `json:"events" gorm:"type:jsonb;not null;default:'[]'"`
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	SSLVerify    bool           `json:"ssl_verify" gorm:"default:true"`
	LastStatus   *string        `json:"last_status" gorm:"size:20"`
	LastError    *string        `json:"last_error" gorm:"type:text"`
	LastDelivery *time.Time     `json:"last_delivery"`
	CreatedAt    time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"not null"`

	// 关联关系
	Repository *Repository      `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
	Deliveries []WebhookDelivery `json:"deliveries,omitempty" gorm:"foreignKey:WebhookID"`
}

// WebhookDelivery Webhook投递记录
type WebhookDelivery struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	WebhookID         uuid.UUID      `json:"webhook_id" gorm:"type:uuid;not null;index"`
	EventType         string         `json:"event_type" gorm:"size:50;not null"`
	DeliveryID        string         `json:"delivery_id" gorm:"size:40;not null;uniqueIndex"`
	RequestHeaders    datatypes.JSON `json:"request_headers" gorm:"type:jsonb"`
	RequestBody       datatypes.JSON `json:"request_body" gorm:"type:jsonb"`
	ResponseStatus    *int           `json:"response_status"`
	ResponseHeaders   datatypes.JSON `json:"response_headers" gorm:"type:jsonb"`
	ResponseBody      *string        `json:"response_body" gorm:"type:text"`
	Success           bool           `json:"success" gorm:"default:false"`
	AttemptCount      int            `json:"attempt_count" gorm:"default:1"`
	LastAttemptAt     time.Time      `json:"last_attempt_at" gorm:"not null"`
	NextAttemptAt     *time.Time     `json:"next_attempt_at"`
	CreatedAt         time.Time      `json:"created_at" gorm:"not null"`

	// 关联关系
	Webhook *Webhook `json:"webhook,omitempty" gorm:"foreignKey:WebhookID"`
}

// PushEvent 推送事件模型
type PushEvent struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	RepositoryID uuid.UUID      `json:"repository_id" gorm:"type:uuid;not null;index"`
	UserID       uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
	Ref          string         `json:"ref" gorm:"size:255;not null"`         // refs/heads/main
	Before       string         `json:"before" gorm:"size:40;not null"`       // 之前的commit SHA
	After        string         `json:"after" gorm:"size:40;not null"`        // 之后的commit SHA
	Forced       bool           `json:"forced" gorm:"default:false"`          // 是否强制推送
	CommitCount  int            `json:"commit_count" gorm:"default:0"`         // 提交数量
	Commits      datatypes.JSON `json:"commits" gorm:"type:jsonb"`             // 提交详情
	Pusher       PusherInfo     `json:"pusher" gorm:"embedded"`                // 推送者信息
	CreatedAt    time.Time      `json:"created_at" gorm:"not null"`

	// 关联关系
	Repository *Repository `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
}

// PusherInfo 推送者信息
type PusherInfo struct {
	Name  string `json:"name" gorm:"size:255"`
	Email string `json:"email" gorm:"size:255"`
}

// AccessKey 访问密钥模型 (用于仓库访问认证)
type AccessKey struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	RepositoryID *uuid.UUID `json:"repository_id" gorm:"type:uuid;index"`  // 可选，为空则为全局密钥
	UserID       uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	Title        string     `json:"title" gorm:"size:255;not null"`
	PublicKey    string     `json:"public_key" gorm:"type:text;not null;uniqueIndex"`
	Fingerprint  string     `json:"fingerprint" gorm:"size:64;not null"`
	KeyType      string     `json:"key_type" gorm:"size:20;not null"`      // ssh-rsa, ssh-ed25519, etc.
	AccessLevel  string     `json:"access_level" gorm:"size:20;not null;default:read"` // read, write, admin
	LastUsedAt   *time.Time `json:"last_used_at"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"not null"`

	// 关联关系
	Repository *Repository `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
}

// GitOperation Git操作审计记录
type GitOperation struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	RepositoryID uuid.UUID      `json:"repository_id" gorm:"type:uuid;not null;index"`
	UserID       uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	Operation    string         `json:"operation" gorm:"size:50;not null"`        // push, pull, clone, etc.
	Protocol     string         `json:"protocol" gorm:"size:10;not null"`         // http, ssh
	RefName      *string        `json:"ref_name" gorm:"size:255"`                 // 分支或标签名
	CommitSHA    *string        `json:"commit_sha" gorm:"size:40"`                // 提交SHA
	ClientIP     string         `json:"client_ip" gorm:"size:45;not null"`        // 客户端IP
	UserAgent    *string        `json:"user_agent" gorm:"size:512"`               // 用户代理
	Success      bool           `json:"success" gorm:"not null"`                  // 操作是否成功
	ErrorMsg     *string        `json:"error_msg" gorm:"type:text"`               // 错误信息
	Duration     int            `json:"duration" gorm:"default:0"`                // 操作耗时(毫秒)
	BytesTransferred int64      `json:"bytes_transferred" gorm:"default:0"`       // 传输字节数
	CreatedAt    time.Time      `json:"created_at" gorm:"not null"`

	// 关联关系
	Repository *Repository `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
}

// User 用户模型 (简化版)
type User struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Email    string    `json:"email" gorm:"size:255;uniqueIndex;not null"`
	FullName *string   `json:"full_name" gorm:"size:255"`
}

// BeforeCreate GORM钩子：创建前
func (r *Repository) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}

func (b *Branch) BeforeCreate(tx *gorm.DB) (err error) {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return
}

func (t *Tag) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return
}

func (w *Webhook) BeforeCreate(tx *gorm.DB) (err error) {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return
}

func (p *PushEvent) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}

func (a *AccessKey) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}

func (g *GitOperation) BeforeCreate(tx *gorm.DB) (err error) {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return
}

// TableName 指定表名
func (Repository) TableName() string {
	return "repositories"
}

func (Branch) TableName() string {
	return "branches"
}

func (Tag) TableName() string {
	return "tags"
}

func (Webhook) TableName() string {
	return "webhooks"
}

func (WebhookDelivery) TableName() string {
	return "webhook_deliveries"
}

func (PushEvent) TableName() string {
	return "push_events"
}

func (AccessKey) TableName() string {
	return "access_keys"
}

func (GitOperation) TableName() string {
	return "git_operations"
}

func (Project) TableName() string {
	return "projects"
}

func (User) TableName() string {
	return "users"
}