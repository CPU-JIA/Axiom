package routes

import (
	"cicd-service/internal/config"
	"cicd-service/internal/handlers"
	"cicd-service/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRoutes 设置路由
func SetupRoutes(
	db *gorm.DB,
	cfg *config.Config,
	pipelineHandler *handlers.PipelineHandler,
	pipelineRunHandler *handlers.PipelineRunHandler,
	cacheHandler *handlers.CacheHandler,
	healthHandler *handlers.HealthHandler,
) *gin.Engine {
	// 根据环境设置Gin模式
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// 基础中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS配置
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.CORS.AllowedOrigins
	corsConfig.AllowMethods = cfg.CORS.AllowedMethods
	corsConfig.AllowHeaders = append(cfg.CORS.AllowedHeaders, "Authorization")
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig))

	// 健康检查端点（不需要认证）
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/health/live", healthHandler.LivenessProbe)
	router.GET("/health/ready", healthHandler.ReadinessProbe)

	// API版本组
	v1 := router.Group("/api/v1")
	{
		// 应用认证中间件
		v1.Use(middleware.JWTAuthMiddleware(cfg.JWT.Secret))
		v1.Use(middleware.RequestIDMiddleware())
		v1.Use(middleware.LoggingMiddleware())

		// 流水线相关路由
		pipelines := v1.Group("/pipelines")
		{
			pipelines.POST("", pipelineHandler.CreatePipeline)
			pipelines.GET("", pipelineHandler.ListPipelines)
			pipelines.GET("/statistics", pipelineHandler.GetPipelineStatistics)
			pipelines.GET("/:id", pipelineHandler.GetPipeline)
			pipelines.PUT("/:id", pipelineHandler.UpdatePipeline)
			pipelines.DELETE("/:id", pipelineHandler.DeletePipeline)
			pipelines.POST("/:id/enable", pipelineHandler.EnablePipeline)
			pipelines.POST("/:id/disable", pipelineHandler.DisablePipeline)
			pipelines.POST("/:id/clone", pipelineHandler.ClonePipeline)
			pipelines.POST("/:id/trigger", pipelineRunHandler.TriggerPipelineByPipeline)
			pipelines.GET("/:pipeline_id/runs", pipelineRunHandler.GetPipelineRunsByPipeline)
		}

		// 流水线运行相关路由
		pipelineRuns := v1.Group("/pipeline-runs")
		{
			pipelineRuns.POST("", pipelineRunHandler.CreatePipelineRun)
			pipelineRuns.GET("", pipelineRunHandler.ListPipelineRuns)
			pipelineRuns.GET("/statistics", pipelineRunHandler.GetPipelineRunStatistics)
			pipelineRuns.GET("/:id", pipelineRunHandler.GetPipelineRun)
			pipelineRuns.POST("/:id/cancel", pipelineRunHandler.CancelPipelineRun)
			pipelineRuns.POST("/:id/retry", pipelineRunHandler.RetryPipelineRun)
		}

		// 构建缓存相关路由
		cache := v1.Group("/cache")
		{
			cache.POST("", cacheHandler.StoreCache)
			cache.GET("", cacheHandler.ListCaches)
			cache.GET("/statistics", cacheHandler.GetCacheStatistics)
			cache.POST("/cleanup", cacheHandler.CleanupCaches)
			cache.GET("/:key", cacheHandler.RetrieveCache)
			cache.DELETE("/:id", cacheHandler.DeleteCache)
			cache.DELETE("/by-key/:key", cacheHandler.DeleteCacheByKey)
			cache.GET("/:id/validate", cacheHandler.ValidateCache)
			cache.GET("/:id/path", cacheHandler.GetCachePath)
			cache.GET("/:id/checksum", cacheHandler.CalculateCacheChecksum)
		}

		// 项目相关的流水线路由
		projects := v1.Group("/projects")
		{
			projects.GET("/:project_id/pipelines", pipelineHandler.GetPipelinesByProject)
		}
	}

	// Webhook端点（可能需要不同的认证方式）
	webhooks := router.Group("/webhooks")
	{
		// 这里可以添加Git仓库webhook处理
		// 例如：webhooks.POST("/git/:repository_id", gitWebhookHandler.HandleWebhook)
		// 暂时保留，后续可以扩展
	}

	return router
}