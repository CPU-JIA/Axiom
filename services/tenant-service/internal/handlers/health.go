package handlers

import (
	"net/http"
	"time"

	"tenant-service/internal/database"
	"tenant-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	db          *gorm.DB
	redisClient *redis.Client
	logger      logger.Logger
}

func NewHealthHandler(db *gorm.DB, redisClient *redis.Client, logger logger.Logger) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
	}
}

// HealthCheck 健康检查端点
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "tenant-service",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// ReadinessCheck 就绪检查端点
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	checks := gin.H{
		"database": "disconnected",
		"cache":    "disconnected",
	}

	allReady := true

	// 检查数据库连接
	if err := database.HealthCheck(h.db); err != nil {
		h.logger.Error("Database health check failed", "error", err)
		checks["database"] = "error: " + err.Error()
		allReady = false
	} else {
		checks["database"] = "connected"
	}

	// 检查Redis连接
	if err := database.HealthRedis(h.redisClient); err != nil {
		h.logger.Error("Redis health check failed", "error", err)
		checks["cache"] = "error: " + err.Error()
		allReady = false
	} else {
		checks["cache"] = "connected"
	}

	status := http.StatusOK
	if !allReady {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status":     map[bool]string{true: "ready", false: "not_ready"}[allReady],
		"service":    "tenant-service",
		"timestamp":  time.Now().Unix(),
		"components": checks,
	})
}

// LivenessCheck 存活检查端点
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"service":   "tenant-service",
		"timestamp": time.Now().Unix(),
	})
}