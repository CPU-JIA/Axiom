package middleware

import (
	"net/http"
	"strconv"
	"time"

	"api-gateway/internal/auth"
	"api-gateway/internal/metrics"
	"api-gateway/internal/ratelimit"
	"api-gateway/pkg/logger"
	"api-gateway/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Logger 日志中间件
func Logger(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(start)

		// 构建完整路径
		if raw != "" {
			path = path + "?" + raw
		}

		// 获取客户端IP
		clientIP := utils.GetClientIP(c.Request.RemoteAddr, utils.NormalizeHeaders(c.Request.Header))

		// 记录日志
		log.Info("HTTP Request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency", latency,
			"ip", clientIP,
			"user_agent", c.Request.UserAgent(),
			"request_id", c.GetString("request_id"),
		)
	}
}

// Recovery 恢复中间件
func Recovery(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("Panic recovered",
					"error", err,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"ip", c.ClientIP(),
					"request_id", c.GetString("request_id"),
				)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "INTERNAL_SERVER_ERROR",
						"message": "Internal server error",
					},
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}

// RequestID 请求ID中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取请求ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 如果没有，生成一个新的UUID
			requestID = uuid.New().String()
		}

		// 设置到上下文和响应头
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// CORS 跨域中间件
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		
		// 检查是否允许该域名
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Accept,Authorization,X-Requested-With,X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length,X-Request-ID,X-RateLimit-Limit,X-RateLimit-Remaining")
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Security 安全中间件
func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置安全响应头
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// Metrics 监控指标中间件
func Metrics(m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		
		// 增加进行中的请求数
		m.IncrementInFlightRequests()
		defer m.DecrementInFlightRequests()

		// 处理请求
		c.Next()

		// 记录指标
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		service := utils.ExtractServiceName(path)

		m.RecordHTTPRequest(method, path, service, statusCode, duration)
	}
}

// RateLimit 限流中间件
func RateLimit(limiter ratelimit.RateLimiter, m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端IP
		clientIP := utils.GetClientIP(c.Request.RemoteAddr, utils.NormalizeHeaders(c.Request.Header))
		
		// 创建IP限流键
		ipKey := ratelimit.NewIPKey(clientIP)
		
		// 检查IP限流
		if !limiter.Allow(c.Request.Context(), ipKey.String()) {
			service := utils.ExtractServiceName(c.Request.URL.Path)
			m.RecordRateLimitBlock("ip", service)
			
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.Burst("")))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", "60")
			
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Rate limit exceeded for IP address",
				},
			})
			c.Abort()
			return
		}

		// 如果有用户信息，检查用户限流
		if userID := c.GetString("user_id"); userID != "" {
			userKey := ratelimit.NewUserKey(userID)
			if !limiter.Allow(c.Request.Context(), userKey.String()) {
				service := utils.ExtractServiceName(c.Request.URL.Path)
				m.RecordRateLimitBlock("user", service)
				
				c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.Burst("")))
				c.Header("X-RateLimit-Remaining", "0")
				
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"code":    "USER_RATE_LIMIT_EXCEEDED",
						"message": "Rate limit exceeded for user",
					},
				})
				c.Abort()
				return
			}
		}

		// 记录限流命中
		service := utils.ExtractServiceName(c.Request.URL.Path)
		m.RecordRateLimitHit("ip", service)

		c.Next()
	}
}

// JWTAuth JWT认证中间件（可选）
func JWTAuth(validator *auth.JWTValidator, m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 无认证头，继续处理（某些端点可能不需要认证）
			c.Next()
			return
		}

		// 验证JWT令牌
		claims, err := validator.ValidateToken(authHeader)
		if err != nil {
			m.RecordAuthFailure("invalid_token")
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
		c.Set("user_id", claims.UserID.String())
		c.Set("user_email", claims.Email)
		c.Set("role", claims.Role)
		if claims.TenantID != nil {
			c.Set("tenant_id", claims.TenantID.String())
		}

		// 记录认证成功
		m.RecordAuthRequest("success")

		c.Next()
	}
}

// RequireAuth 强制认证中间件
func RequireAuth(validator *auth.JWTValidator, m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.RecordAuthFailure("missing_token")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header is required",
				},
			})
			c.Abort()
			return
		}

		// 验证JWT令牌
		claims, err := validator.ValidateToken(authHeader)
		if err != nil {
			m.RecordAuthFailure("invalid_token")
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
		c.Set("user_id", claims.UserID.String())
		c.Set("user_email", claims.Email)
		c.Set("role", claims.Role)
		if claims.TenantID != nil {
			c.Set("tenant_id", claims.TenantID.String())
		}

		// 记录认证成功
		m.RecordAuthRequest("success")

		c.Next()
	}
}

// Timeout 超时中间件
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置超时上下文
		ctx, cancel := c.Request.Context(), func() {} // 默认不取消
		if timeout > 0 {
			ctx, cancel = c.Request.Context(), cancel
		}
		defer cancel()

		// 更新请求上下文
		c.Request = c.Request.WithContext(ctx)

		// 处理请求
		c.Next()
	}
}