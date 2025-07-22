package database

import (
	"fmt"
	"time"

	"iam-service/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect 连接数据库
func Connect(databaseURL string) (*gorm.DB, error) {
	// 配置GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// 连接PostgreSQL
	db, err := gorm.Open(postgres.Open(databaseURL), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 获取底层sql.DB对象进行连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数  
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生存时间

	return db, nil
}

// Migrate 执行数据库迁移
func Migrate(db *gorm.DB) error {
	// 启用UUID扩展
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create uuid extension: %w", err)
	}
	
	// 启用pgcrypto扩展（用于密码hash）
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\"").Error; err != nil {
		return fmt.Errorf("failed to create pgcrypto extension: %w", err)
	}

	// 自动迁移模型
	models := []interface{}{
		&models.User{},
		&models.UserAuthentication{},  
		&models.MFASetting{},
		&models.LoginAttempt{},
		&models.RefreshToken{},
		&models.EmailVerification{},
		&models.PasswordReset{},
	}

	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	// 创建索引
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// createIndexes 创建额外的索引
func createIndexes(db *gorm.DB) error {
	indexes := []string{
		// 用户表索引
		`CREATE INDEX IF NOT EXISTS idx_users_email_status ON users(email, status) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_users_last_login_at ON users(last_login_at) WHERE deleted_at IS NULL`,
		
		// 认证表索引
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auth_provider_unique 
		 ON user_authentications(user_id, provider) 
		 WHERE deleted_at IS NULL AND is_active = true`,
		`CREATE INDEX IF NOT EXISTS idx_user_auth_provider_user_id ON user_authentications(provider, provider_user_id) WHERE deleted_at IS NULL`,
		
		// 登录尝试表索引
		`CREATE INDEX IF NOT EXISTS idx_login_attempts_email_created ON login_attempts(email, created_at) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_login_attempts_ip_created ON login_attempts(ip_address, created_at) WHERE deleted_at IS NULL`,
		
		// 刷新Token表索引
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id_active ON refresh_tokens(user_id) WHERE deleted_at IS NULL AND is_revoked = false`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at) WHERE deleted_at IS NULL`,
		
		// 邮箱验证表索引
		`CREATE INDEX IF NOT EXISTS idx_email_verifications_email_code ON email_verifications(email, code) WHERE deleted_at IS NULL AND is_used = false`,
		`CREATE INDEX IF NOT EXISTS idx_email_verifications_expires_at ON email_verifications(expires_at) WHERE deleted_at IS NULL`,
		
		// 密码重置表索引
		`CREATE INDEX IF NOT EXISTS idx_password_resets_expires_at ON password_resets(expires_at) WHERE deleted_at IS NULL AND is_used = false`,
	}

	for _, indexSQL := range indexes {
		if err := db.Exec(indexSQL).Error; err != nil {
			return fmt.Errorf("failed to create index: %s, error: %w", indexSQL, err)
		}
	}

	return nil
}

// Health 检查数据库健康状态
func Health(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}