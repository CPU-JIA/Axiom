package routes

import (
	"git-gateway-service/internal/config"
	"git-gateway-service/internal/handlers"
	"git-gateway-service/internal/middleware"
	"git-gateway-service/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRoutes 配置路由
func SetupRoutes(db *gorm.DB, cfg *config.Config) *gin.Engine {
	// 创建服务实例
	repoService := services.NewRepositoryService(db, cfg)
	branchService := services.NewBranchService(db)
	webhookService := services.NewWebhookService(db, cfg)
	accessKeyService := services.NewAccessKeyService(db)
	gitOpService := services.NewGitOperationService(db)

	// 创建处理器实例
	repoHandler := handlers.NewRepositoryHandler(repoService)
	branchHandler := handlers.NewBranchHandler(branchService)
	webhookHandler := handlers.NewWebhookHandler(webhookService)
	accessKeyHandler := handlers.NewAccessKeyHandler(accessKeyService)
	gitOpHandler := handlers.NewGitOperationHandler(gitOpService)

	// 设置Gin模式
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 全局中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	
	// CORS中间件
	router.Use(middleware.CORSMiddleware(cfg.CORS))

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"service": "git-gateway-service",
			"version": "1.0.0",
		})
	})

	// API路由组
	api := router.Group("/api/v1")
	
	// 应用JWT认证中间件
	api.Use(middleware.JWTAuthMiddleware(cfg.JWT.Secret))

	// 仓库管理路由
	repositories := api.Group("/repositories")
	{
		repositories.POST("", repoHandler.CreateRepository)
		repositories.GET("", repoHandler.ListRepositories)
		repositories.GET("/:id", repoHandler.GetRepository)
		repositories.PUT("/:id", repoHandler.UpdateRepository)
		repositories.DELETE("/:id", repoHandler.DeleteRepository)
		repositories.GET("/:id/stats", repoHandler.GetRepositoryStatistics)
		repositories.POST("/:id/stats", repoHandler.UpdateRepositoryStatistics)
		
		// 通过项目ID和名称获取仓库
		repositories.GET("/project/:project_id/name/:name", repoHandler.GetRepositoryByName)
	}

	// 分支管理路由
	branches := api.Group("/branches")
	{
		branches.POST("", branchHandler.CreateBranch)
		branches.GET("", branchHandler.ListBranches)
		branches.GET("/:id", branchHandler.GetBranch)
		branches.PUT("/:id", branchHandler.UpdateBranch)
		branches.DELETE("/:id", branchHandler.DeleteBranch)
		
		// 分支保护
		branches.POST("/:id/protection", branchHandler.SetBranchProtection)
		branches.DELETE("/:id/protection", branchHandler.RemoveBranchProtection)
		
		// 通过仓库ID和名称获取分支
		branches.GET("/repository/:repository_id/name/:name", branchHandler.GetBranchByName)
	}

	// 默认分支设置路由
	api.POST("/repositories/:repository_id/branches/:branch_id/default", branchHandler.SetDefaultBranch)

	// Webhook管理路由
	webhooks := api.Group("/webhooks")
	{
		webhooks.POST("", webhookHandler.CreateWebhook)
		webhooks.GET("", webhookHandler.ListWebhooks)
		webhooks.GET("/:id", webhookHandler.GetWebhook)
		webhooks.PUT("/:id", webhookHandler.UpdateWebhook)
		webhooks.DELETE("/:id", webhookHandler.DeleteWebhook)
		
		// 测试触发Webhook
		webhooks.POST("/repositories/:repository_id/trigger", webhookHandler.TriggerWebhook)
	}

	// 访问密钥管理路由
	accessKeys := api.Group("/access-keys")
	{
		accessKeys.POST("", accessKeyHandler.CreateAccessKey)
		accessKeys.GET("", accessKeyHandler.ListAccessKeys)
		accessKeys.GET("/:id", accessKeyHandler.GetAccessKey)
		accessKeys.PUT("/:id", accessKeyHandler.UpdateAccessKey)
		accessKeys.DELETE("/:id", accessKeyHandler.DeleteAccessKey)
		
		// 验证公钥
		accessKeys.POST("/validate", accessKeyHandler.ValidatePublicKey)
	}

	// Git操作审计路由
	operations := api.Group("/operations")
	{
		operations.GET("", gitOpHandler.ListOperations)
		operations.GET("/:id", gitOpHandler.GetOperation)
		operations.GET("/stats", gitOpHandler.GetOperationStats)
		
		// 清理旧记录（管理员功能）
		operations.DELETE("/cleanup", gitOpHandler.CleanupOldRecords)
	}

	// Git协议处理路由 (不需要JWT认证)
	gitProtocol := router.Group("/git")
	{
		// HTTP Git协议支持
		gitProtocol.Any("/*path", handleGitHTTPProtocol)
	}

	return router
}

// handleGitHTTPProtocol 处理Git HTTP协议请求
func handleGitHTTPProtocol(c *gin.Context) {
	// TODO: 实现Git HTTP协议处理逻辑
	// 这里需要实现git-upload-pack和git-receive-pack的HTTP包装器
	
	c.JSON(501, gin.H{
		"error": "Git HTTP协议处理器尚未实现",
		"path":  c.Param("path"),
	})
}