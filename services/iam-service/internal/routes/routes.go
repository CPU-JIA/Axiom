package routes

import (
	"iam-service/internal/config"
	"iam-service/internal/handlers"
	"iam-service/internal/middleware"
	"iam-service/internal/services"
	"iam-service/pkg/logger"

	"github.com/gin-gonic/gin"
)

type Router struct {
	authHandler   *handlers.AuthHandler
	userHandler   *handlers.UserHandler
	healthHandler *handlers.HealthHandler
	config        *config.Config
	logger        logger.Logger
}

func NewRouter(
	authService *services.AuthService,
	userService *services.UserService,
	cfg *config.Config,
	log logger.Logger,
) *Router {
	return &Router{
		authHandler:   handlers.NewAuthHandler(authService, log),
		userHandler:   handlers.NewUserHandler(userService, log),
		healthHandler: handlers.NewHealthHandler(),
		config:        cfg,
		logger:        log,
	}
}

func (r *Router) Setup() *gin.Engine {
	// 根据环境设置Gin模式
	if r.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 全局中间件
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(r.logger))
	router.Use(middleware.Recovery(r.logger))
	router.Use(middleware.CORS())
	router.Use(middleware.Security())

	// 健康检查端点（无需认证）
	health := router.Group("/health")
	{
		health.GET("/", r.healthHandler.HealthCheck)
		health.GET("/ready", r.healthHandler.ReadinessCheck)
		health.GET("/live", r.healthHandler.LivenessCheck)
	}

	// API版本组
	v1 := router.Group("/api/v1")

	// 认证相关路由（无需认证）
	auth := v1.Group("/auth")
	{
		auth.POST("/register", r.authHandler.Register)
		auth.POST("/login", r.authHandler.Login)
		auth.POST("/refresh", r.authHandler.RefreshToken)
		auth.POST("/forgot-password", r.authHandler.ForgotPassword)
		auth.POST("/reset-password", r.authHandler.ResetPassword)
		auth.POST("/verify-email", r.authHandler.VerifyEmail)
		auth.POST("/resend-verification", r.authHandler.ResendVerification)
	}

	// 内部服务认证路由（需要内部认证）
	internal := v1.Group("/internal")
	internal.Use(middleware.InternalAuth(r.config.InternalSecret))
	{
		internal.POST("/introspect", r.authHandler.IntrospectToken)
		internal.POST("/switch-tenant", r.authHandler.SwitchTenant)
	}

	// 需要用户认证的路由
	authenticated := v1.Group("")
	authenticated.Use(middleware.JWTAuth(r.config.JWTSecret))
	{
		// 用户管理
		users := authenticated.Group("/users")
		{
			users.GET("/profile", r.userHandler.GetProfile)
			users.PUT("/profile", r.userHandler.UpdateProfile)
			users.POST("/change-password", r.userHandler.ChangePassword)
			users.POST("/upload-avatar", r.userHandler.UploadAvatar)

			// 管理员功能
			users.GET("", r.userHandler.ListUsers)           // GET /api/v1/users
			users.GET("/:id", r.userHandler.GetUser)         // GET /api/v1/users/:id
			users.PUT("/:id/status", r.userHandler.UpdateUserStatus) // PUT /api/v1/users/:id/status
			users.DELETE("/:id", r.userHandler.DeleteUser)   // DELETE /api/v1/users/:id
		}

		// MFA管理
		mfa := authenticated.Group("/mfa")
		{
			mfa.POST("/setup", r.authHandler.SetupMFA)
			mfa.POST("/verify", r.authHandler.VerifyMFA)
			mfa.DELETE("/disable", r.authHandler.DisableMFA)
			mfa.GET("/backup-codes", r.authHandler.GetBackupCodes)
			mfa.POST("/backup-codes/regenerate", r.authHandler.RegenerateBackupCodes)
		}
	}

	return router
}