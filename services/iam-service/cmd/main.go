package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"iam-service/internal/config"
	"iam-service/internal/database"
	"iam-service/internal/routes"
	"iam-service/internal/services"
	"iam-service/pkg/logger"
)

func main() {
	// 初始化配置
	cfg := config.Load()

	// 初始化日志
	log := logger.New(cfg.LogLevel)

	// 连接数据库
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}

	// 自动迁移数据库表
	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to migrate database", "error", err)
	}

	// 初始化Redis
	redisClient := database.NewRedisClient(cfg.RedisURL)

	// 初始化服务层
	userService := services.NewUserService(db, redisClient, log)
	authService := services.NewAuthService(db, redisClient, cfg.JWTSecret, log)

	// 初始化路由
	router := routes.NewRouter(authService, userService, cfg, log)
	engine := router.Setup()

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      engine,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 启动服务器
	go func() {
		log.Info("Starting IAM service", 
			"port", cfg.Port,
			"environment", cfg.Environment,
		)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	// 给服务器5秒时间完成正在处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", "error", err)
	}

	log.Info("Server exited")
}