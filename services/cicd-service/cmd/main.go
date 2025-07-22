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

	"cicd-service/internal/config"
	"cicd-service/internal/handlers"
	"cicd-service/internal/models"
	"cicd-service/internal/routes"
	"cicd-service/internal/services"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// 加载配置
	cfg := config.Load()
	
	log.Printf("🚀 启动CI/CD服务...")
	log.Printf("📊 环境: %s", cfg.Environment)
	log.Printf("🌍 端口: %s", cfg.Port)
	log.Printf("🗄️ 数据库: %s@%s:%d/%s", 
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// 初始化数据库
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v", err)
	}
	log.Printf("✅ 数据库连接成功")

	// 运行数据库迁移
	if err := runMigrations(db); err != nil {
		log.Fatalf("❌ 数据库迁移失败: %v", err)
	}
	log.Printf("✅ 数据库迁移完成")

	// 初始化服务
	cacheService := services.NewCacheService(db, cfg)
	pipelineService := services.NewPipelineService(db, cfg)
	
	// 初始化Tekton服务（可能失败，不阻塞启动）
	var tektonService services.TektonService
	var pipelineRunService services.PipelineRunService
	
	tektonService, err = services.NewTektonService(cfg, nil) // 暂时传nil，后面会设置
	if err != nil {
		log.Printf("⚠️ Tekton服务初始化失败: %v", err)
		log.Printf("⚠️ CI/CD功能将受限")
		// 创建一个空的tekton服务实现，避免nil指针
		tektonService = &noOpTektonService{}
	} else {
		log.Printf("✅ Tekton服务连接成功")
	}
	
	pipelineRunService = services.NewPipelineRunService(db, cfg, tektonService)
	
	// 如果tekton服务是真实的，更新其pipelineRunService引用
	if realTekton, ok := tektonService.(*services.tektonService); ok {
		// 这里需要设置runService字段，但由于是私有字段，需要修改服务设计
		_ = realTekton // 避免未使用变量警告
	}

	// 初始化处理器
	pipelineHandler := handlers.NewPipelineHandler(pipelineService)
	pipelineRunHandler := handlers.NewPipelineRunHandler(pipelineRunService)
	cacheHandler := handlers.NewCacheHandler(cacheService)
	healthHandler := handlers.NewHealthHandler(db, tektonService)

	// 设置路由
	router := routes.SetupRoutes(db, cfg, pipelineHandler, pipelineRunHandler, cacheHandler, healthHandler)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 启动事件监听（如果Tekton可用）
	if tektonService != nil {
		go func() {
			ctx := context.Background()
			if err := tektonService.WatchPipelineRuns(ctx); err != nil {
				log.Printf("⚠️ 启动PipelineRun事件监听失败: %v", err)
			}
		}()
		
		go func() {
			ctx := context.Background()
			if err := tektonService.WatchTaskRuns(ctx); err != nil {
				log.Printf("⚠️ 启动TaskRun事件监听失败: %v", err)
			}
		}()
	}

	// 启动缓存清理定时任务
	go startCacheCleanupRoutine(cacheService, cfg)

	// 启动流水线运行清理定时任务  
	go startPipelineRunCleanupRoutine(pipelineRunService, cfg)

	// 启动服务器
	go func() {
		log.Printf("🌟 CI/CD服务启动在端口 %s", cfg.Port)
		log.Printf("📚 健康检查: http://localhost:%s/health", cfg.Port)
		log.Printf("📖 API文档: http://localhost:%s/api/v1", cfg.Port)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ 服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("🛑 正在关闭CI/CD服务...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("❌ 服务器强制关闭: %v", err)
	} else {
		log.Printf("✅ CI/CD服务已优雅关闭")
	}
}

// initDatabase 初始化数据库连接
func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	var gormLogger logger.Interface
	if cfg.IsDevelopment() {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.GetDSN()), &gorm.Config{
		Logger:                                   gormLogger,
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

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// runMigrations 运行数据库迁移
func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Pipeline{},
		&models.Task{},
		&models.PipelineRun{},
		&models.TaskRun{},
		&models.BuildCache{},
		&models.Secret{},
		&models.Environment{},
		&models.Project{}, // 引用的项目模型
	)
}

// startCacheCleanupRoutine 启动缓存清理定时任务
func startCacheCleanupRoutine(cacheService services.CacheService, cfg *config.Config) {
	ticker := time.NewTicker(time.Duration(cfg.Cache.CleanupInterval) * time.Minute)
	defer ticker.Stop()

	log.Printf("⚡ 启动缓存清理定时任务，间隔: %d分钟", cfg.Cache.CleanupInterval)

	for range ticker.C {
		if err := cacheService.Cleanup(); err != nil {
			log.Printf("⚠️ 缓存清理失败: %v", err)
		} else {
			log.Printf("🧹 缓存清理完成")
		}
	}
}

// startPipelineRunCleanupRoutine 启动流水线运行清理定时任务
func startPipelineRunCleanupRoutine(pipelineRunService services.PipelineRunService, cfg *config.Config) {
	// 每6小时清理一次过期的运行记录
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	log.Printf("⚡ 启动流水线运行清理定时任务，间隔: 6小时")

	for range ticker.C {
		if err := pipelineRunService.CleanupExpiredRuns(); err != nil {
			log.Printf("⚠️ 流水线运行清理失败: %v", err)
		} else {
			log.Printf("🧹 流水线运行清理完成")
		}
	}
}

// noOpTektonService 空操作Tekton服务实现（当Tekton不可用时使用）
type noOpTektonService struct{}

func (s *noOpTektonService) CreatePipeline(ctx context.Context, pipeline *models.Pipeline) error {
	return fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) UpdatePipeline(ctx context.Context, pipeline *models.Pipeline) error {
	return fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) DeletePipeline(ctx context.Context, pipelineID uuid.UUID) error {
	return fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) CreatePipelineRun(ctx context.Context, req *services.TektonPipelineRunRequest) error {
	return fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) GetPipelineRunStatus(ctx context.Context, runID uuid.UUID) (*services.TektonPipelineRunStatus, error) {
	return nil, fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) CancelPipelineRun(ctx context.Context, runID uuid.UUID) error {
	return fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) GetTaskRunStatus(ctx context.Context, taskRunID uuid.UUID) (*services.TektonTaskRunStatus, error) {
	return nil, fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) GetTaskRunLogs(ctx context.Context, taskRunID uuid.UUID) (string, error) {
	return "", fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) WatchPipelineRuns(ctx context.Context) error {
	return fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) WatchTaskRuns(ctx context.Context) error {
	return fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) CreateSecret(ctx context.Context, secret *models.Secret) error {
	return fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) UpdateSecret(ctx context.Context, secret *models.Secret) error {
	return fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) DeleteSecret(ctx context.Context, secretID uuid.UUID) error {
	return fmt.Errorf("Tekton服务不可用")
}

func (s *noOpTektonService) HealthCheck(ctx context.Context) error {
	return fmt.Errorf("Tekton服务不可用")
}