package handlers

import (
	"net/http"
	"strconv"

	"tenant-service/internal/middleware"
	"tenant-service/internal/models"
	"tenant-service/internal/services"
	"tenant-service/pkg/logger"

	"github.com/gin-gonic/gin"
)

type TenantHandler struct {
	tenantService *services.TenantService
	logger        logger.Logger
}

func NewTenantHandler(tenantService *services.TenantService, logger logger.Logger) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
		logger:        logger,
	}
}

// CreateTenant 创建租户
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not found in context",
			},
		})
		return
	}

	var req services.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "Invalid request data",
				"details": err.Error(),
			},
		})
		return
	}

	tenant, err := h.tenantService.CreateTenant(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("Failed to create tenant", "user_id", userID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "TENANT_CREATION_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": tenant,
	})
}

// GetTenant 获取租户信息
func (h *TenantHandler) GetTenant(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_TENANT_ID",
				"message": "Invalid tenant ID",
			},
		})
		return
	}

	tenant, err := h.tenantService.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get tenant", "tenant_id", tenantID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "TENANT_NOT_FOUND",
				"message": err.Error(),
			},
		})
		return
	}

	// 设置当前用户角色
	if role, err := middleware.GetMemberRole(c); err == nil {
		memberRole := models.MemberRole(role)
		tenant.CurrentUserRole = &memberRole
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tenant,
	})
}

// UpdateTenant 更新租户信息
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_TENANT_ID",
				"message": "Invalid tenant ID",
			},
		})
		return
	}

	var req services.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "Invalid request data",
				"details": err.Error(),
			},
		})
		return
	}

	tenant, err := h.tenantService.UpdateTenant(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Error("Failed to update tenant", "tenant_id", tenantID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "TENANT_UPDATE_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tenant,
	})
}

// ListMembers 获取租户成员列表
func (h *TenantHandler) ListMembers(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_TENANT_ID",
				"message": "Invalid tenant ID",
			},
		})
		return
	}

	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	members, total, err := h.tenantService.ListMembers(c.Request.Context(), tenantID, page, size)
	if err != nil {
		h.logger.Error("Failed to list members", "tenant_id", tenantID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "LIST_MEMBERS_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"members":    members,
			"total":      total,
			"page":       page,
			"size":       size,
			"totalPages": (total + int64(size) - 1) / int64(size),
		},
	})
}

// InviteMember 邀请成员
func (h *TenantHandler) InviteMember(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_TENANT_ID",
				"message": "Invalid tenant ID",
			},
		})
		return
	}

	inviterID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not found in context",
			},
		})
		return
	}

	var req struct {
		Email   string             `json:"email" binding:"required,email"`
		Role    models.MemberRole  `json:"role" binding:"required"`
		Message string             `json:"message,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "Invalid request data",
				"details": err.Error(),
			},
		})
		return
	}

	invitation, err := h.tenantService.InviteMember(c.Request.Context(), tenantID, inviterID, req.Email, req.Role)
	if err != nil {
		h.logger.Error("Failed to invite member", "tenant_id", tenantID, "email", req.Email, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVITE_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"invitation_id": invitation.ID,
			"email":         invitation.Email,
			"role":          invitation.Role,
			"expires_at":    invitation.ExpiresAt,
		},
	})
}

// GetMyTenants 获取当前用户的租户列表
func (h *TenantHandler) GetMyTenants(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not found in context",
			},
		})
		return
	}

	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	// 暂时返回空列表，需要实现用户租户查询
	tenants := []*models.TenantResponse{}
	total := int64(0)

	h.logger.Info("Retrieved user tenants", "user_id", userID, "count", len(tenants))

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"tenants":    tenants,
			"total":      total,
			"page":       page,
			"size":       size,
			"totalPages": (total + int64(size) - 1) / int64(size),
		},
	})
}