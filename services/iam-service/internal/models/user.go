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

// User 用户表
type User struct {
	BaseModel
	
	// 基本信息
	Email     string `gorm:"uniqueIndex;not null;size:255" json:"email"`
	FullName  string `gorm:"size:255" json:"full_name"`
	AvatarURL string `gorm:"size:1024" json:"avatar_url,omitempty"`
	
	// 状态
	Status       UserStatus `gorm:"not null;default:'active'" json:"status"`
	EmailVerified bool      `gorm:"default:false" json:"email_verified"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	
	// 关联
	Authentications []UserAuthentication `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	MFASettings     *MFASetting          `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	LoginAttempts   []LoginAttempt       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// UserResponse 用户响应结构
type UserResponse struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	FullName      string     `json:"full_name"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	Status        UserStatus `json:"status"`
	EmailVerified bool       `json:"email_verified"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	MFAEnabled    bool       `json:"mfa_enabled"`
}

// ToResponse 转换为响应格式
func (u *User) ToResponse() *UserResponse {
	response := &UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		FullName:      u.FullName,
		AvatarURL:     u.AvatarURL,
		Status:        u.Status,
		EmailVerified: u.EmailVerified,
		LastLoginAt:   u.LastLoginAt,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		MFAEnabled:    false,
	}

	// 设置MFA状态
	if u.MFASettings != nil {
		response.MFAEnabled = u.MFASettings.IsEnabled
	}

	return response
}

// UserStatus 用户状态枚举
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive" 
	UserStatusSuspended UserStatus = "suspended"
	UserStatusPending   UserStatus = "pending"
)

// UserAuthentication 用户认证信息
type UserAuthentication struct {
	BaseModel
	
	UserID           uuid.UUID    `gorm:"type:uuid;not null;index" json:"user_id"`
	Provider         AuthProvider `gorm:"not null" json:"provider"`
	ProviderUserID   string       `gorm:"size:255" json:"provider_user_id,omitempty"`
	Credentials      string       `gorm:"type:text;not null" json:"-"` // JSON存储, 不在API中返回
	IsActive         bool         `gorm:"default:true" json:"is_active"`
	LastUsedAt       *time.Time   `json:"last_used_at,omitempty"`
	
	// 关联
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// AuthProvider 认证提供方枚举
type AuthProvider string

const (
	AuthProviderLocal  AuthProvider = "local"
	AuthProviderGoogle AuthProvider = "google"  
	AuthProviderGithub AuthProvider = "github"
	AuthProviderSAML   AuthProvider = "saml"
)

// MFASetting MFA设置
type MFASetting struct {
	BaseModel
	
	UserID        uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	IsEnabled     bool      `gorm:"default:false" json:"is_enabled"`
	Secret        string    `gorm:"size:255;not null" json:"-"` // TOTP密钥，不在API中返回
	BackupCodes   string    `gorm:"type:text" json:"-"`         // 备用码，JSON存储，不在API中返回
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	
	// 关联
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// LoginAttempt 登录尝试记录
type LoginAttempt struct {
	BaseModel
	
	UserID    *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"` // 可为空，因为可能是无效用户
	Email     string     `gorm:"size:255;not null;index" json:"email"`
	IPAddress string     `gorm:"size:45;not null" json:"ip_address"`
	UserAgent string     `gorm:"size:1024" json:"user_agent"`
	Success   bool       `gorm:"not null" json:"success"`
	FailReason string    `gorm:"size:255" json:"fail_reason,omitempty"`
	
	// 关联
	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"-"`
}

// RefreshToken 刷新token
type RefreshToken struct {
	BaseModel
	
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string    `gorm:"size:255;uniqueIndex;not null" json:"-"` // token的hash值
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	IsRevoked bool      `gorm:"default:false;index" json:"is_revoked"`
	IPAddress string    `gorm:"size:45" json:"ip_address"`
	UserAgent string    `gorm:"size:1024" json:"user_agent"`
	
	// 关联
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// EmailVerification 邮箱验证
type EmailVerification struct {
	BaseModel
	
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Email     string    `gorm:"size:255;not null" json:"email"`
	Code      string    `gorm:"size:255;not null" json:"-"` // 验证码hash
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	IsUsed    bool      `gorm:"default:false" json:"is_used"`
	
	// 关联  
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// PasswordReset 密码重置
type PasswordReset struct {
	BaseModel
	
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string    `gorm:"size:255;uniqueIndex;not null" json:"-"` // token的hash值
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	IsUsed    bool      `gorm:"default:false" json:"is_used"`
	IPAddress string    `gorm:"size:45" json:"ip_address"`
	
	// 关联
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 方法定义表名
func (User) TableName() string              { return "users" }
func (UserAuthentication) TableName() string { return "user_authentications" }
func (MFASetting) TableName() string        { return "mfa_settings" }
func (LoginAttempt) TableName() string      { return "login_attempts" }
func (RefreshToken) TableName() string      { return "refresh_tokens" }
func (EmailVerification) TableName() string { return "email_verifications" }
func (PasswordReset) TableName() string     { return "password_resets" }