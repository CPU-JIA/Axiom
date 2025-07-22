package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel 包含所有模型的通用字段
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate 在创建前生成UUID
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// Tenant 租户表
type Tenant struct {
	BaseModel
	
	// 基本信息
	Name        string      `gorm:"uniqueIndex;not null;size:50" json:"name"`
	Slug        string      `gorm:"uniqueIndex;not null;size:50" json:"slug"` // URL友好的标识符
	DisplayName string      `gorm:"size:100" json:"display_name"`
	Description string      `gorm:"size:500" json:"description"`
	LogoURL     string      `gorm:"size:1024" json:"logo_url,omitempty"`
	
	// 状态和计费
	Status      TenantStatus `gorm:"not null;default:'active'" json:"status"`
	PlanType    PlanType     `gorm:"not null;default:'free'" json:"plan_type"`
	
	// 资源限制
	MaxMembers       int   `gorm:"not null;default:10" json:"max_members"`
	MaxProjects      int   `gorm:"not null;default:5" json:"max_projects"`
	StorageQuota     int64 `gorm:"not null;default:1073741824" json:"storage_quota"` // 字节数，默认1GB
	StorageUsed      int64 `gorm:"default:0" json:"storage_used"`
	
	// 时间戳
	ActivatedAt   *time.Time `json:"activated_at,omitempty"`
	SuspendedAt   *time.Time `json:"suspended_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	
	// 设置
	Settings      string `gorm:"type:jsonb" json:"settings,omitempty"`      // JSON存储的租户设置
	Features      string `gorm:"type:jsonb" json:"features,omitempty"`      // JSON存储的功能开关
	CustomDomain  string `gorm:"size:255" json:"custom_domain,omitempty"`   // 自定义域名
	
	// 关联
	Members     []TenantMember     `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
	Invitations []TenantInvitation `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
	AuditLogs   []TenantAuditLog   `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
}

// TenantStatus 租户状态枚举
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusInactive  TenantStatus = "inactive"
	TenantStatusPending   TenantStatus = "pending"
	TenantStatusExpired   TenantStatus = "expired"
)

// PlanType 计划类型枚举
type PlanType string

const (
	PlanTypeFree       PlanType = "free"
	PlanTypeStarter    PlanType = "starter"
	PlanTypePro        PlanType = "pro"
	PlanTypeEnterprise PlanType = "enterprise"
)

// TenantMember 租户成员表
type TenantMember struct {
	BaseModel
	
	TenantID uuid.UUID    `gorm:"type:uuid;not null;index" json:"tenant_id"`
	UserID   uuid.UUID    `gorm:"type:uuid;not null;index" json:"user_id"`
	Role     MemberRole   `gorm:"not null" json:"role"`
	Status   MemberStatus `gorm:"not null;default:'active'" json:"status"`
	
	// 权限
	Permissions string `gorm:"type:jsonb" json:"permissions,omitempty"` // JSON存储的权限列表
	
	// 时间戳
	JoinedAt    time.Time  `gorm:"not null" json:"joined_at"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	SuspendedAt *time.Time `json:"suspended_at,omitempty"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
	
	// 邀请信息
	InvitedBy   *uuid.UUID `gorm:"type:uuid" json:"invited_by,omitempty"`
	InviteToken string     `gorm:"size:255" json:"-"` // 邀请令牌，不在API中返回
	
	// 关联
	Tenant Tenant `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
}

// MemberRole 成员角色枚举
type MemberRole string

const (
	MemberRoleOwner      MemberRole = "owner"
	MemberRoleAdmin      MemberRole = "admin"
	MemberRoleMaintainer MemberRole = "maintainer"
	MemberRoleDeveloper  MemberRole = "developer"
	MemberRoleGuest      MemberRole = "guest"
)

// MemberStatus 成员状态枚举
type MemberStatus string

const (
	MemberStatusActive    MemberStatus = "active"
	MemberStatusInactive  MemberStatus = "inactive"
	MemberStatusSuspended MemberStatus = "suspended"
	MemberStatusPending   MemberStatus = "pending"
)

// TenantInvitation 租户邀请表
type TenantInvitation struct {
	BaseModel
	
	TenantID    uuid.UUID    `gorm:"type:uuid;not null;index" json:"tenant_id"`
	InviterID   uuid.UUID    `gorm:"type:uuid;not null" json:"inviter_id"`
	Email       string       `gorm:"size:255;not null;index" json:"email"`
	Role        MemberRole   `gorm:"not null" json:"role"`
	Status      InviteStatus `gorm:"not null;default:'pending'" json:"status"`
	Token       string       `gorm:"size:255;uniqueIndex;not null" json:"-"` // 邀请令牌
	ExpiresAt   time.Time    `gorm:"not null;index" json:"expires_at"`
	AcceptedAt  *time.Time   `json:"accepted_at,omitempty"`
	AcceptedBy  *uuid.UUID   `gorm:"type:uuid" json:"accepted_by,omitempty"`
	
	// 权限
	Permissions string `gorm:"type:jsonb" json:"permissions,omitempty"`
	Message     string `gorm:"size:500" json:"message,omitempty"` // 邀请消息
	
	// 关联
	Tenant Tenant `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
}

// InviteStatus 邀请状态枚举
type InviteStatus string

const (
	InviteStatusPending   InviteStatus = "pending"
	InviteStatusAccepted  InviteStatus = "accepted"
	InviteStatusDeclined  InviteStatus = "declined"
	InviteStatusExpired   InviteStatus = "expired"
	InviteStatusCancelled InviteStatus = "cancelled"
)

// TenantAuditLog 租户审计日志
type TenantAuditLog struct {
	BaseModel
	
	TenantID    uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	UserID      *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	Action      string    `gorm:"size:100;not null;index" json:"action"`
	ResourceType string   `gorm:"size:50;index" json:"resource_type"`
	ResourceID   *uuid.UUID `gorm:"type:uuid" json:"resource_id,omitempty"`
	
	// 详细信息
	Details   string `gorm:"type:jsonb" json:"details,omitempty"` // JSON存储的详细信息
	IPAddress string `gorm:"size:45" json:"ip_address"`
	UserAgent string `gorm:"size:1024" json:"user_agent"`
	
	// 状态
	Status string `gorm:"size:20;default:'success'" json:"status"` // success, failed, warning
	Error  string `gorm:"size:1024" json:"error,omitempty"`
	
	// 关联
	Tenant Tenant `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
}

// TenantResponse 租户响应结构
type TenantResponse struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Slug        string       `json:"slug"`
	DisplayName string       `json:"display_name"`
	Description string       `json:"description"`
	LogoURL     string       `json:"logo_url,omitempty"`
	Status      TenantStatus `json:"status"`
	PlanType    PlanType     `json:"plan_type"`
	
	// 统计信息
	MemberCount  int `json:"member_count"`
	ProjectCount int `json:"project_count"`
	
	// 资源使用情况
	MaxMembers   int   `json:"max_members"`
	MaxProjects  int   `json:"max_projects"`
	StorageQuota int64 `json:"storage_quota"`
	StorageUsed  int64 `json:"storage_used"`
	
	// 时间戳
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	
	// 当前用户在该租户的角色
	CurrentUserRole *MemberRole `json:"current_user_role,omitempty"`
}

// ToResponse 转换为响应格式
func (t *Tenant) ToResponse() *TenantResponse {
	return &TenantResponse{
		ID:          t.ID,
		Name:        t.Name,
		Slug:        t.Slug,
		DisplayName: t.DisplayName,
		Description: t.Description,
		LogoURL:     t.LogoURL,
		Status:      t.Status,
		PlanType:    t.PlanType,
		MaxMembers:  t.MaxMembers,
		MaxProjects: t.MaxProjects,
		StorageQuota: t.StorageQuota,
		StorageUsed: t.StorageUsed,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		ActivatedAt: t.ActivatedAt,
		ExpiresAt:   t.ExpiresAt,
	}
}

// TenantMemberResponse 租户成员响应结构
type TenantMemberResponse struct {
	ID       uuid.UUID    `json:"id"`
	TenantID uuid.UUID    `json:"tenant_id"`
	UserID   uuid.UUID    `json:"user_id"`
	Role     MemberRole   `json:"role"`
	Status   MemberStatus `json:"status"`
	
	// 用户信息（从IAM服务获取）
	UserEmail    string `json:"user_email,omitempty"`
	UserName     string `json:"user_name,omitempty"`
	UserAvatarURL string `json:"user_avatar_url,omitempty"`
	
	// 时间戳
	JoinedAt    time.Time  `json:"joined_at"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
}

// ToResponse 转换为响应格式
func (m *TenantMember) ToResponse() *TenantMemberResponse {
	return &TenantMemberResponse{
		ID:       m.ID,
		TenantID: m.TenantID,
		UserID:   m.UserID,
		Role:     m.Role,
		Status:   m.Status,
		JoinedAt: m.JoinedAt,
		ActivatedAt: m.ActivatedAt,
		LastSeenAt:  m.LastSeenAt,
	}
}

// TableName 方法定义表名
func (Tenant) TableName() string           { return "tenants" }
func (TenantMember) TableName() string     { return "tenant_members" }
func (TenantInvitation) TableName() string { return "tenant_invitations" }
func (TenantAuditLog) TableName() string   { return "tenant_audit_logs" }