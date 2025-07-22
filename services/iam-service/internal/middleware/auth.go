package middleware

import (
	"errors"
	"net/http"
	"strings"

	"iam-service/internal/services"

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

// GetCurrentUserEmail 从上下文获取当前用户邮箱
func GetCurrentUserEmail(c *gin.Context) (string, error) {
	email, exists := c.Get("user_email")
	if !exists {
		return "", errors.New("user_email not found in context")
	}

	if e, ok := email.(string); ok {
		return e, nil
	}

	return "", errors.New("invalid user_email type")
}

// GetCurrentTenantID 从上下文获取当前租户ID
func GetCurrentTenantID(c *gin.Context) (*uuid.UUID, error) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return nil, nil // 可能没有设置租户ID
	}

	if tid, ok := tenantID.(*uuid.UUID); ok {
		return tid, nil
	}

	return nil, errors.New("invalid tenant_id type")
}