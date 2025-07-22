package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"project-service/internal/config"
	"project-service/internal/handlers"
	"project-service/internal/services"
	"project-service/pkg/logger"
	"project-service/pkg/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化日志
	logger := logger.New(cfg.GetLogLevel())

	// 根据环境设置Gin模式
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化数据库
	db, err := initDatabase(cfg, logger)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 初始化服务
	projectService := services.NewProjectService(db, logger)
	taskService := services.NewTaskService(db, logger)
	sprintService := services.NewSprintService(db, logger)

	// 初始化处理器
	projectHandler := handlers.NewProjectHandler(projectService, logger)
	taskHandler := handlers.NewTaskHandler(taskService, logger)
	sprintHandler := handlers.NewSprintHandler(sprintService, logger)

	// 创建路由器
	router := gin.New()

	// 全局中间件
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.CORS(cfg.CORS.AllowedOrigins))

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "project-service",
			"version":   "1.0.0",
			"timestamp": time.Now().Unix(),
		})
	})

	// API路由
	api := router.Group("/api/v1")
	{
		// JWT认证中间件
		api.Use(middleware.JWTAuth(cfg.JWT.Secret))

		// 注册路由
		projectHandler.RegisterRoutes(api)
		taskHandler.RegisterRoutes(api)
		sprintHandler.RegisterRoutes(api)
	}

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 启动服务器
	go func() {
		logger.Info("项目服务启动", 
			"port", cfg.Port,
			"environment", cfg.Environment,
		)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("启动服务器失败", "error", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("正在关闭服务器...")

	// 给服务器30秒时间完成正在处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("服务器强制关闭", "error", err)
	}

	// 关闭数据库连接
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	logger.Info("服务器已退出")
}

// initDatabase 初始化数据库连接
func initDatabase(cfg *config.Config, logger logger.Logger) (*gorm.DB, error) {
	logger.Info("正在连接数据库...", "host", cfg.Database.Host, "dbname", cfg.Database.DBName)

	db, err := gorm.Open(postgres.Open(cfg.Database.GetDSN()), &gorm.Config{
		Logger: logger.GetGormLogger(),
	})
	
	if err != nil {
		return nil, err
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	logger.Info("数据库连接成功")
	return db, nil
}