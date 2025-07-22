package middleware

import (
	"net/http"
	"strings"

	"git-gateway-service/internal/config"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS中间件
func CORSMiddleware(corsConfig config.CORSConfig) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// 检查是否允许该源
		allowed := false
		for _, allowedOrigin := range corsConfig.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		// 设置允许的方法
		allowedMethods := strings.Join(corsConfig.AllowedMethods, ", ")
		c.Writer.Header().Set("Access-Control-Allow-Methods", allowedMethods)

		// 设置允许的头
		allowedHeaders := strings.Join(corsConfig.AllowedHeaders, ", ")
		c.Writer.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

		// 设置其他CORS头
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})
}