package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"iam-service/internal/middleware"
	"iam-service/internal/models"
	"iam-service/internal/services"
	"iam-service/pkg/logger"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *services.UserService
	logger      logger.Logger
}

func NewUserHandler(userService *services.UserService, logger logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// GetProfile 获取用户资料
func (h *UserHandler) GetProfile(c *gin.Context) {
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

	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user profile", "user_id", userID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

// UpdateProfile 更新用户资料
func (h *UserHandler) UpdateProfile(c *gin.Context) {
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

	var req services.UpdateUserRequest
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

	user, err := h.userService.UpdateUser(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("Failed to update user profile", "user_id", userID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "UPDATE_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

// ChangePassword 修改密码
func (h *UserHandler) ChangePassword(c *gin.Context) {
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
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
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

	changeReq := &services.ChangePasswordRequest{
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}

	err = h.userService.ChangePassword(c.Request.Context(), userID, changeReq)
	if err != nil {
		h.logger.Error("Failed to change password", "user_id", userID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "PASSWORD_CHANGE_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message": "Password changed successfully",
		},
	})
}

// UploadAvatar 上传头像
func (h *UserHandler) UploadAvatar(c *gin.Context) {
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

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "FILE_UPLOAD_FAILED",
				"message": "Failed to get uploaded file",
			},
		})
		return
	}
	defer file.Close()

	// 检查文件大小（限制为5MB）
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "FILE_TOO_LARGE",
				"message": "File size must be less than 5MB",
			},
		})
		return
	}

	avatarURL, err := h.userService.UploadAvatar(c.Request.Context(), userID, header)
	if err != nil {
		h.logger.Error("Failed to upload avatar", "user_id", userID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "AVATAR_UPLOAD_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"avatar_url": avatarURL,
		},
	})
}

// ListUsers 获取用户列表（管理员功能）
func (h *UserHandler) ListUsers(c *gin.Context) {
	// 检查管理员权限
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Admin access required",
			},
		})
		return
	}

	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	_ = c.Query("search") // 暂时不使用搜索功能
	
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	// 暂时返回空列表，需要实现管理员功能
	users := []*models.UserResponse{}
	total := int64(0)
	var err error
	if err != nil {
		h.logger.Error("Failed to list users", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "LIST_USERS_FAILED",
				"message": "Failed to retrieve user list",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"users":      users,
			"total":      total,
			"page":       page,
			"size":       size,
			"totalPages": (total + int64(size) - 1) / int64(size),
		},
	})
}

// GetUser 获取指定用户信息（管理员功能）
func (h *UserHandler) GetUser(c *gin.Context) {
	// 检查管理员权限
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Admin access required",
			},
		})
		return
	}

	userIDStr := c.Param("id")
	userID, err := parseUUID(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_USER_ID",
				"message": "Invalid user ID format",
			},
		})
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user", "user_id", userID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

// UpdateUserStatus 更新用户状态（管理员功能）
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	// 检查管理员权限
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Admin access required",
			},
		})
		return
	}

	userIDStr := c.Param("id")
	userID, err := parseUUID(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_USER_ID",
				"message": "Invalid user ID format",
			},
		})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=active inactive suspended"`
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

	// 暂时不实现管理员状态更新功能
	err = errors.New("admin functionality not implemented")
	if err != nil {
		h.logger.Error("Failed to update user status", "user_id", userID, "status", req.Status, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "STATUS_UPDATE_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message": "User status updated successfully",
		},
	})
}

// DeleteUser 删除用户（管理员功能）
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// 检查管理员权限
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Admin access required",
			},
		})
		return
	}

	userIDStr := c.Param("id")
	userID, err := parseUUID(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_USER_ID",
				"message": "Invalid user ID format",
			},
		})
		return
	}

	// 不允许删除自己
	currentUserID, _ := middleware.GetCurrentUserID(c)
	if currentUserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "CANNOT_DELETE_SELF",
				"message": "Cannot delete your own account",
			},
		})
		return
	}

	// 暂时不实现管理员删除功能
	err = errors.New("admin functionality not implemented")
	if err != nil {
		h.logger.Error("Failed to delete user", "user_id", userID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "DELETE_USER_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message": "User deleted successfully",
		},
	})
}