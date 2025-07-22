package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git-gateway-service/internal/config"
	"git-gateway-service/internal/models"
	"git-gateway-service/internal/routes"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 自动迁移数据库表
	if err := autoMigrate(db); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 设置路由
	router := routes.SetupRoutes(db, cfg)

	// 启动服务器
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// 优雅关闭处理
	go func() {
		log.Printf("Git Gateway服务启动在端口: %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭Git Gateway服务...")

	// 给予5秒时间优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("服务器强制关闭: %v", err)
	}

	log.Println("Git Gateway服务已关闭")
}

// initDatabase 初始化数据库连接
func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.Database.GetDSN()
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// 禁用外键约束检查（在多租户环境中）
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Println("数据库连接成功")
	return db, nil
}

// autoMigrate 自动迁移数据库表
func autoMigrate(db *gorm.DB) error {
	log.Println("开始数据库迁移...")

	// 创建UUID扩展（PostgreSQL）
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		log.Printf("创建UUID扩展失败: %v", err)
	}

	// 自动迁移所有模型
	err := db.AutoMigrate(
		&models.Repository{},
		&models.Branch{},
		&models.Tag{},
		&models.Webhook{},
		&models.WebhookDelivery{},
		&models.PushEvent{},
		&models.AccessKey{},
		&models.GitOperation{},
		&models.Project{},
		&models.User{},
	)

	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Println("数据库迁移完成")
	return nil
}