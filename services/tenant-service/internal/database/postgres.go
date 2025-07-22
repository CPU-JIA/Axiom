package database

import (
	"fmt"
	"time"

	"tenant-service/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect 连接到数据库
func Connect(databaseURL string) (*gorm.DB, error) {
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(databaseURL), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return db, nil
}

// Migrate 执行数据库迁移
func Migrate(db *gorm.DB) error {
	// 启用UUID扩展
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create uuid extension: %w", err)
	}

	// 执行自动迁移
	err := db.AutoMigrate(
		&models.Tenant{},
		&models.TenantMember{},
		&models.TenantInvitation{},
		&models.TenantAuditLog{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
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
		// 租户表索引
		"CREATE INDEX IF NOT EXISTS idx_tenants_status_plan ON tenants(status, plan_type)",
		"CREATE INDEX IF NOT EXISTS idx_tenants_created_at ON tenants(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_tenants_expires_at ON tenants(expires_at) WHERE expires_at IS NOT NULL",
		
		// 租户成员表索引
		"CREATE INDEX IF NOT EXISTS idx_tenant_members_tenant_user ON tenant_members(tenant_id, user_id)",
		"CREATE INDEX IF NOT EXISTS idx_tenant_members_role_status ON tenant_members(role, status)",
		"CREATE INDEX IF NOT EXISTS idx_tenant_members_last_seen ON tenant_members(last_seen_at) WHERE last_seen_at IS NOT NULL",
		
		// 租户邀请表索引
		"CREATE INDEX IF NOT EXISTS idx_tenant_invitations_email_status ON tenant_invitations(email, status)",
		"CREATE INDEX IF NOT EXISTS idx_tenant_invitations_expires_at ON tenant_invitations(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_tenant_invitations_tenant_status ON tenant_invitations(tenant_id, status)",
		
		// 审计日志表索引
		"CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_action_created ON tenant_audit_logs(action, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_resource ON tenant_audit_logs(resource_type, resource_id) WHERE resource_id IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_user_created ON tenant_audit_logs(user_id, created_at) WHERE user_id IS NOT NULL",
	}

	for _, indexSQL := range indexes {
		if err := db.Exec(indexSQL).Error; err != nil {
			return fmt.Errorf("failed to create index: %s, error: %w", indexSQL, err)
		}
	}

	return nil
}

// HealthCheck 检查数据库健康状态
func HealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	
	return sqlDB.Ping()
}