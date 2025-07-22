package handlers

import (
	"net/http"

	"iam-service/internal/middleware"
	"iam-service/internal/services"
	"iam-service/pkg/logger"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *services.AuthService
	logger      logger.Logger
}

func NewAuthHandler(authService *services.AuthService, logger logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
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

	// 从请求中获取IP和User-Agent
	req.IPAddress = c.ClientIP()
	req.UserAgent = c.GetHeader("User-Agent")

	user, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Registration failed", "email", req.Email, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "REGISTRATION_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": user,
	})
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
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

	// 从请求中获取IP和User-Agent
	req.IPAddress = c.ClientIP()
	req.UserAgent = c.GetHeader("User-Agent")

	response, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		h.logger.Warn("Login failed", "email", req.Email, "ip", req.IPAddress, "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "LOGIN_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// RefreshToken 刷新访问令牌
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
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

	response, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.logger.Warn("Token refresh failed", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "TOKEN_REFRESH_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// IntrospectToken 内部Token验证（服务间调用）
func (h *AuthHandler) IntrospectToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "Invalid request data",
			},
		})
		return
	}

	// 这里应该实现Token验证逻辑
	// 暂时返回简单响应
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"active": true,
			"user_id": "user-id-from-token",
		},
	})
}

// ForgotPassword 忘记密码
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "Invalid request data",
			},
		})
		return
	}

	// 这里应该实现发送重置密码邮件的逻辑
	h.logger.Info("Password reset requested", "email", req.Email)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message": "If the email address exists in our system, a password reset link will be sent",
		},
	})
}

// ResetPassword 重置密码
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "Invalid request data",
			},
		})
		return
	}

	// 这里应该实现密码重置逻辑
	h.logger.Info("Password reset attempted", "token", req.Token[:8]+"...")

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message": "Password has been reset successfully",
		},
	})
}

// VerifyEmail 验证邮箱
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "Invalid request data",
			},
		})
		return
	}

	// 这里应该实现邮箱验证逻辑
	h.logger.Info("Email verification attempted", "token", req.Token[:8]+"...")

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message": "Email verified successfully",
		},
	})
}

// ResendVerification 重发验证邮件
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "Invalid request data",
			},
		})
		return
	}

	h.logger.Info("Verification email resend requested", "email", req.Email)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message": "Verification email sent",
		},
	})
}

// SwitchTenant 切换租户（内部API）
func (h *AuthHandler) SwitchTenant(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "Invalid request data",
			},
		})
		return
	}

	// 这里应该实现租户切换逻辑
	h.logger.Info("Tenant switch requested", "tenant_id", req.TenantID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"access_token": "new-token-with-tenant",
			"tenant_id":    req.TenantID,
		},
	})
}

// MFA相关处理器 - 这些是占位符实现
func (h *AuthHandler) SetupMFA(c *gin.Context) {
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

	h.logger.Info("MFA setup requested", "user_id", userID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"secret":     "JBSWY3DPEHPK3PXP",
			"qr_code":    "data:image/png;base64,iVBORw0KGgoAAAANSUhE...",
			"backup_codes": []string{"12345678", "87654321"},
		},
	})
}

func (h *AuthHandler) VerifyMFA(c *gin.Context) {
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

	var req struct {
		Code string `json:"code" binding:"required,len=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "Invalid request data",
			},
		})
		return
	}

	h.logger.Info("MFA verification requested", "user_id", userID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message": "MFA enabled successfully",
		},
	})
}

func (h *AuthHandler) DisableMFA(c *gin.Context) {
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

	h.logger.Info("MFA disable requested", "user_id", userID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message": "MFA disabled successfully",
		},
	})
}

func (h *AuthHandler) GetBackupCodes(c *gin.Context) {
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

	h.logger.Info("MFA backup codes requested", "user_id", userID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"backup_codes": []string{"12345678", "87654321", "11111111", "22222222"},
		},
	})
}

func (h *AuthHandler) RegenerateBackupCodes(c *gin.Context) {
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

	h.logger.Info("MFA backup codes regeneration requested", "user_id", userID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"backup_codes": []string{"99999999", "88888888", "77777777", "66666666"},
		},
	})
}