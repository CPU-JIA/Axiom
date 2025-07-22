package handlers

import (
	"net/http"
	"sync"
	"time"

	"api-gateway/internal/config"
	"api-gateway/internal/proxy"
	"api-gateway/pkg/logger"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	proxy     *proxy.ServiceProxy
	services  map[string]config.ServiceConfig
	logger    logger.Logger
	lastCheck time.Time
	status    map[string]bool
	mu        sync.RWMutex
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(proxy *proxy.ServiceProxy, cfg *config.Config, logger logger.Logger) *HealthHandler {
	handler := &HealthHandler{
		proxy:    proxy,
		services: cfg.Services,
		logger:   logger,
		status:   make(map[string]bool),
	}

	// 启动定期健康检查
	if cfg.HealthCheck.Enabled {
		go handler.periodicHealthCheck(time.Duration(cfg.HealthCheck.Interval) * time.Second)
	}

	return handler
}

// HealthCheck API Gateway健康检查端点
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "api-gateway",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// ReadinessCheck 就绪检查端点
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	checks := make(map[string]interface{})
	allReady := true

	// 检查各个后端服务
	for serviceName := range h.services {
		status, exists := h.status[serviceName]
		if !exists || !status {
			checks[serviceName] = "unhealthy"
			allReady = false
		} else {
			checks[serviceName] = "healthy"
		}
	}

	// 检查检查时间是否过期
	if time.Since(h.lastCheck) > 5*time.Minute {
		checks["health_check"] = "stale"
		allReady = false
	} else {
		checks["health_check"] = "current"
	}

	httpStatus := http.StatusOK
	if !allReady {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":     map[bool]string{true: "ready", false: "not_ready"}[allReady],
		"service":    "api-gateway",
		"timestamp":  time.Now().Unix(),
		"components": checks,
		"last_check": h.lastCheck.Unix(),
	})
}

// LivenessCheck 存活检查端点
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"service":   "api-gateway",
		"timestamp": time.Now().Unix(),
	})
}

// ServiceHealth 单个服务健康检查端点
func (h *HealthHandler) ServiceHealth(c *gin.Context) {
	serviceName := c.Param("service")
	
	// 检查服务是否配置
	if _, exists := h.services[serviceName]; !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "SERVICE_NOT_FOUND",
				"message": "Service not found",
			},
		})
		return
	}

	// 执行健康检查
	err := h.proxy.HealthCheck(serviceName)
	healthy := err == nil

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}

	result := gin.H{
		"service":   serviceName,
		"status":    map[bool]string{true: "healthy", false: "unhealthy"}[healthy],
		"timestamp": time.Now().Unix(),
	}

	if !healthy {
		result["error"] = err.Error()
	}

	c.JSON(status, result)
}

// ServicesStatus 所有服务状态端点
func (h *HealthHandler) ServicesStatus(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	services := make(map[string]interface{})
	healthyCount := 0
	totalCount := len(h.services)

	for serviceName := range h.services {
		status, exists := h.status[serviceName]
		healthy := exists && status
		
		services[serviceName] = gin.H{
			"status":    map[bool]string{true: "healthy", false: "unhealthy"}[healthy],
			"timestamp": h.lastCheck.Unix(),
		}

		if healthy {
			healthyCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"services":      services,
		"summary": gin.H{
			"total":   totalCount,
			"healthy": healthyCount,
			"unhealthy": totalCount - healthyCount,
		},
		"last_check": h.lastCheck.Unix(),
		"timestamp":  time.Now().Unix(),
	})
}

// periodicHealthCheck 定期健康检查
func (h *HealthHandler) periodicHealthCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 立即执行一次检查
	h.checkAllServices()

	for range ticker.C {
		h.checkAllServices()
	}
}

// checkAllServices 检查所有服务
func (h *HealthHandler) checkAllServices() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.logger.Debug("Starting health check for all services")
	
	for serviceName := range h.services {
		healthy := h.checkService(serviceName)
		h.status[serviceName] = healthy
		
		h.logger.Debug("Service health check completed", 
			"service", serviceName, 
			"healthy", healthy,
		)
	}

	h.lastCheck = time.Now()
	h.logger.Info("Health check completed for all services", "timestamp", h.lastCheck)
}

// checkService 检查单个服务
func (h *HealthHandler) checkService(serviceName string) bool {
	err := h.proxy.HealthCheck(serviceName)
	if err != nil {
		h.logger.Warn("Service health check failed", 
			"service", serviceName, 
			"error", err,
		)
		return false
	}
	return true
}

// IsServiceHealthy 检查服务是否健康
func (h *HealthHandler) IsServiceHealthy(serviceName string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	status, exists := h.status[serviceName]
	return exists && status
}