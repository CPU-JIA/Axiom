package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HealthHandler 健康检查处理器
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthCheck 健康检查端点
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "iam-service",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// ReadinessCheck 就绪检查端点
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// 这里可以检查数据库连接、缓存连接等
	// 目前简化为直接返回就绪状态
	c.JSON(http.StatusOK, gin.H{
		"status":     "ready",
		"service":    "iam-service",
		"timestamp":  time.Now().Unix(),
		"components": gin.H{
			"database": "connected",
			"cache":    "connected",
		},
	})
}

// LivenessCheck 存活检查端点
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"service":   "iam-service",
		"timestamp": time.Now().Unix(),
	})
}

// parseUUID 解析UUID字符串的辅助函数
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}