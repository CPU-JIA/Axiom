package routes

import (
	"tenant-service/internal/config"
	"tenant-service/internal/handlers"
	"tenant-service/internal/middleware"
	"tenant-service/internal/services"
	"tenant-service/pkg/logger"

	"github.com/gin-gonic/gin"
)

type Router struct {
	tenantHandler *handlers.TenantHandler
	healthHandler *handlers.HealthHandler
	tenantService *services.TenantService
	config        *config.Config
	logger        logger.Logger
}

func NewRouter(
	tenantService *services.TenantService,
	tenantHandler *handlers.TenantHandler,
	healthHandler *handlers.HealthHandler,
	cfg *config.Config,
	log logger.Logger,
) *Router {
	return &Router{
		tenantHandler: tenantHandler,
		healthHandler: healthHandler,
		tenantService: tenantService,
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

	// 租户管理路由（需要JWT认证）
	tenants := v1.Group("/tenants")
	tenants.Use(middleware.JWTAuth(r.config.JWTSecret))
	{
		// 创建租户和获取用户租户列表
		tenants.POST("", r.tenantHandler.CreateTenant)
		tenants.GET("/my", r.tenantHandler.GetMyTenants)

		// 特定租户操作（需要租户权限验证）
		tenant := tenants.Group("/:tenant_id")
		tenant.Use(middleware.TenantAuth(r.tenantService))
		{
			// 租户信息管理
			tenant.GET("", r.tenantHandler.GetTenant)
			tenant.PUT("", middleware.RequireRole("admin"), r.tenantHandler.UpdateTenant)

			// 成员管理
			members := tenant.Group("/members")
			{
				members.GET("", r.tenantHandler.ListMembers)
				members.POST("/invite", middleware.RequireRole("admin"), r.tenantHandler.InviteMember)
				// TODO: 更多成员管理端点
				// members.DELETE("/:user_id", middleware.RequireRole("admin"), r.tenantHandler.RemoveMember)
				// members.PUT("/:user_id/role", middleware.RequireRole("owner"), r.tenantHandler.UpdateMemberRole)
			}

			// 邀请管理
			invitations := tenant.Group("/invitations")
			invitations.Use(middleware.RequireRole("admin"))
			{
				// TODO: 邀请管理端点
				// invitations.GET("", r.tenantHandler.ListInvitations)
				// invitations.DELETE("/:invitation_id", r.tenantHandler.CancelInvitation)
			}

			// 审计日志
			audit := tenant.Group("/audit")
			audit.Use(middleware.RequireRole("admin"))
			{
				// TODO: 审计日志端点
				// audit.GET("", r.tenantHandler.GetAuditLogs)
			}

			// 设置管理
			settings := tenant.Group("/settings")
			settings.Use(middleware.RequireRole("admin"))
			{
				// TODO: 设置管理端点
				// settings.GET("", r.tenantHandler.GetSettings)
				// settings.PUT("", r.tenantHandler.UpdateSettings)
			}
		}
	}

	// 邀请处理路由（无需认证，使用邀请令牌）
	_ = v1.Group("/invites") // 预留给未来实现
	{
		// TODO: 邀请处理端点
		// invites.GET("/:token", r.tenantHandler.GetInvitation)
		// invites.POST("/:token/accept", r.tenantHandler.AcceptInvitation)
		// invites.POST("/:token/decline", r.tenantHandler.DeclineInvitation)
	}

	// 内部API（服务间调用）
	_ = v1.Group("/internal").Use(middleware.InternalAuth(r.config.InternalSecret)) // 预留给未来实现
	{
		// TODO: 内部API端点
		// internal.GET("/tenants/:id", r.tenantHandler.GetTenantInternal)
		// internal.POST("/tenants/:id/members/sync", r.tenantHandler.SyncMembers)
		// internal.GET("/users/:id/tenants", r.tenantHandler.GetUserTenantsInternal)
	}

	return router
}