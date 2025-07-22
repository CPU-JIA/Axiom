package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-gateway/internal/auth"
	"api-gateway/internal/config"
	"api-gateway/internal/handlers"
	"api-gateway/internal/metrics"
	"api-gateway/internal/middleware"
	"api-gateway/internal/proxy"
	"api-gateway/internal/ratelimit"
	"api-gateway/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 初始化配置
	cfg := config.Load()

	// 初始化日志
	log := logger.New(cfg.LogLevel)

	// 根据环境设置Gin模式
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化Redis客户端
	var redisClient *redis.Client
	if cfg.RedisURL != "" {
		opts, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			log.Fatal("Failed to parse Redis URL", "error", err)
		}
		redisClient = redis.NewClient(opts)
		
		// 测试Redis连接
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Warn("Failed to connect to Redis, using memory rate limiter", "error", err)
			redisClient = nil
		}
	}

	// 初始化监控指标
	metricsCollector := metrics.NewMetrics()

	// 初始化限流器
	var rateLimiter ratelimit.RateLimiter
	if cfg.RateLimit.Enabled {
		if redisClient != nil {
			rateLimiter = ratelimit.NewRedisRateLimiter(
				redisClient, 
				cfg.RateLimit.DefaultRPS, 
				cfg.RateLimit.DefaultBurst, 
				log,
			)
		} else {
			rateLimiter = ratelimit.NewMemoryRateLimiter(
				cfg.RateLimit.DefaultRPS, 
				cfg.RateLimit.DefaultBurst,
			)
		}
	}

	// 初始化JWT验证器
	jwtValidator := auth.NewJWTValidator(cfg.JWTSecret)

	// 初始化服务代理
	serviceProxy := proxy.NewServiceProxy(cfg, log)

	// 初始化健康检查处理器
	healthHandler := handlers.NewHealthHandler(serviceProxy, cfg, log)

	// 创建路由器
	router := gin.New()

	// 全局中间件
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(log))
	router.Use(middleware.Recovery(log))
	router.Use(middleware.Security())
	router.Use(middleware.CORS(cfg.CORS.AllowedOrigins))
	
	// 监控中间件
	if cfg.Metrics.Enabled {
		router.Use(middleware.Metrics(metricsCollector))
	}

	// 限流中间件
	if rateLimiter != nil {
		router.Use(middleware.RateLimit(rateLimiter, metricsCollector))
	}

	// 可选JWT认证中间件（不强制，允许部分端点无需认证）
	router.Use(middleware.JWTAuth(jwtValidator, metricsCollector))

	// 健康检查路由
	health := router.Group("/health")
	{
		health.GET("", healthHandler.HealthCheck)
		health.GET("/ready", healthHandler.ReadinessCheck)
		health.GET("/live", healthHandler.LivenessCheck)
		health.GET("/services", healthHandler.ServicesStatus)
		health.GET("/services/:service", healthHandler.ServiceHealth)
	}

	// 主要API路由 - 代理到后端服务
	api := router.Group("/api")
	{
		// API v1路由
		v1 := api.Group("/v1")
		{
			// 认证相关 - 路由到IAM服务
			v1.Any("/auth/*path", serviceProxy.ProxyRequest)
			
			// 租户管理 - 路由到Tenant服务（需要认证）
			tenants := v1.Group("/tenants")
			tenants.Use(middleware.RequireAuth(jwtValidator, metricsCollector))
			tenants.Any("/*path", serviceProxy.ProxyRequest)
			
			// 项目管理 - 路由到Project服务（需要认证）
			projects := v1.Group("/projects") 
			projects.Use(middleware.RequireAuth(jwtValidator, metricsCollector))
			projects.Any("/*path", serviceProxy.ProxyRequest)
			
			// Git操作 - 路由到Git Gateway服务（需要认证）
			git := v1.Group("/git")
			git.Use(middleware.RequireAuth(jwtValidator, metricsCollector))
			git.Any("/*path", serviceProxy.ProxyRequest)
			
			// CI/CD管道 - 路由到CI/CD服务（需要认证）
			cicd := v1.Group("/cicd")
			cicd.Use(middleware.RequireAuth(jwtValidator, metricsCollector))
			cicd.Any("/*path", serviceProxy.ProxyRequest)
			
			// 通知服务 - 路由到Notification服务（需要认证）
			notifications := v1.Group("/notifications")
			notifications.Use(middleware.RequireAuth(jwtValidator, metricsCollector))
			notifications.Any("/*path", serviceProxy.ProxyRequest)
			
			// 知识库 - 路由到KB服务
			kb := v1.Group("/kb")
			kb.Any("/*path", serviceProxy.ProxyRequest)
		}
	}

	// 静态文件服务（如果需要）
	router.Static("/static", "./static")

	// 默认路由 - 返回API信息
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":     "api-gateway",
			"version":     "1.0.0",
			"environment": cfg.Environment,
			"timestamp":   time.Now().Unix(),
			"endpoints": gin.H{
				"health":  "/health",
				"api":     "/api/v1",
				"metrics": "/metrics",
			},
		})
	})

	// 404处理
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "The requested resource was not found",
				"path":    c.Request.URL.Path,
			},
		})
	})

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Timeout.Read) * time.Second,
		WriteTimeout: time.Duration(cfg.Timeout.Write) * time.Second,
		IdleTimeout:  time.Duration(cfg.Timeout.Idle) * time.Second,
	}

	// 启动监控服务器（如果启用）
	var metricsSrv *http.Server
	if cfg.Metrics.Enabled {
		metricsRouter := gin.New()
		metricsRouter.GET(cfg.Metrics.Path, gin.WrapH(promhttp.Handler()))
		
		metricsSrv = &http.Server{
			Addr:    ":" + cfg.Metrics.Port,
			Handler: metricsRouter,
		}
		
		go func() {
			log.Info("Starting metrics server", "port", cfg.Metrics.Port, "path", cfg.Metrics.Path)
			if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("Metrics server failed", "error", err)
			}
		}()
	}

	// 启动主服务器
	go func() {
		log.Info("Starting API Gateway", 
			"port", cfg.Port,
			"environment", cfg.Environment,
			"rate_limit_enabled", cfg.RateLimit.Enabled,
			"metrics_enabled", cfg.Metrics.Enabled,
		)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down servers...")

	// 给服务器5秒时间完成正在处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭主服务器
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", "error", err)
	}

	// 关闭监控服务器
	if metricsSrv != nil {
		if err := metricsSrv.Shutdown(ctx); err != nil {
			log.Error("Metrics server forced to shutdown", "error", err)
		}
	}

	// 关闭Redis连接
	if redisClient != nil {
		redisClient.Close()
	}

	log.Info("Servers exited")
}