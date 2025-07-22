package middleware

import (
	"errors"
	"net/http"
	"strings"

	"tenant-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTAuth JWT认证中间件
func JWTAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header is required",
				},
			})
			c.Abort()
			return
		}

		// 检查Bearer前缀
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN_FORMAT",
					"message": "Authorization header must use Bearer scheme",
				},
			})
			c.Abort()
			return
		}

		// 解析JWT
		claims := &services.JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Invalid or expired access token",
				},
			})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("tenant_id", claims.TenantID)
		c.Set("role", claims.Role)
		c.Set("jti", claims.ID)

		c.Next()
	}
}

// InternalAuth 内部服务认证中间件
func InternalAuth(internalSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Internal-Token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED", 
					"message": "Internal token is required",
				},
			})
			c.Abort()
			return
		}

		if token != internalSecret {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_INTERNAL_TOKEN",
					"message": "Invalid internal token",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// TenantAuth 租户访问权限中间件
func TenantAuth(tenantService *services.TenantService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "User ID not found in context",
				},
			})
			c.Abort()
			return
		}

		tenantIDParam := c.Param("tenant_id")
		if tenantIDParam == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "MISSING_TENANT_ID",
					"message": "Tenant ID is required",
				},
			})
			c.Abort()
			return
		}

		tenantID, err := uuid.Parse(tenantIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "INVALID_TENANT_ID",
					"message": "Invalid tenant ID format",
				},
			})
			c.Abort()
			return
		}

		// 检查用户是否是租户成员
		member, err := tenantService.GetMember(c.Request.Context(), tenantID, userID.(uuid.UUID))
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "ACCESS_DENIED",
					"message": "You don't have access to this tenant",
				},
			})
			c.Abort()
			return
		}

		// 将租户信息存入上下文
		c.Set("current_tenant_id", tenantID)
		c.Set("member_role", member.Role)
		c.Set("member_status", member.Status)

		c.Next()
	}
}

// RequireRole 要求特定角色的中间件
func RequireRole(minRole string) gin.HandlerFunc {
	roleHierarchy := map[string]int{
		"guest":      1,
		"developer":  2,
		"maintainer": 3,
		"admin":      4,
		"owner":      5,
	}

	return func(c *gin.Context) {
		memberRole, exists := c.Get("member_role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "ACCESS_DENIED",
					"message": "Member role not found",
				},
			})
			c.Abort()
			return
		}

		currentRoleLevel := roleHierarchy[string(memberRole.(string))]
		requiredRoleLevel := roleHierarchy[minRole]

		if currentRoleLevel < requiredRoleLevel {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "INSUFFICIENT_PERMISSIONS",
					"message": "You don't have sufficient permissions for this action",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUserID 从上下文获取当前用户ID
func GetCurrentUserID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user_id not found in context")
	}

	if uid, ok := userID.(uuid.UUID); ok {
		return uid, nil
	}

	return uuid.Nil, errors.New("invalid user_id type")
}

// GetCurrentTenantID 从上下文获取当前租户ID
func GetCurrentTenantID(c *gin.Context) (uuid.UUID, error) {
	tenantID, exists := c.Get("current_tenant_id")
	if !exists {
		return uuid.Nil, errors.New("current_tenant_id not found in context")
	}

	if tid, ok := tenantID.(uuid.UUID); ok {
		return tid, nil
	}

	return uuid.Nil, errors.New("invalid current_tenant_id type")
}

// GetMemberRole 从上下文获取成员角色
func GetMemberRole(c *gin.Context) (string, error) {
	role, exists := c.Get("member_role")
	if !exists {
		return "", errors.New("member_role not found in context")
	}

	if r, ok := role.(string); ok {
		return r, nil
	}

	return "", errors.New("invalid member_role type")
}