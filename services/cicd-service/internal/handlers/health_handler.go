package handlers

import (
	"context"
	"net/http"
	"time"

	"cicd-service/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db            *gorm.DB
	tektonService services.TektonService
}

func NewHealthHandler(db *gorm.DB, tektonService services.TektonService) *HealthHandler {
	return &HealthHandler{
		db:            db,
		tektonService: tektonService,
	}
}

// HealthCheck 应用健康检查
// @Summary 应用健康检查
// @Description 检查应用各组件健康状态
// @Tags health
// @Produce json
// @Success 200 {object} APIResponse{data=HealthResponse}
// @Failure 503 {object} APIResponse{data=HealthResponse}
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response := &HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Checks:    make(map[string]ComponentHealth),
	}

	allHealthy := true

	// 检查数据库连接
	dbHealth := h.checkDatabase()
	response.Checks["database"] = dbHealth
	if dbHealth.Status != "healthy" {
		allHealthy = false
	}

	// 检查Tekton连接
	tektonHealth := h.checkTekton(ctx)
	response.Checks["tekton"] = tektonHealth
	if tektonHealth.Status != "healthy" {
		allHealthy = false
	}

	// 检查磁盘空间
	diskHealth := h.checkDiskSpace()
	response.Checks["disk"] = diskHealth
	if diskHealth.Status != "healthy" {
		allHealthy = false
	}

	// 设置整体状态
	if !allHealthy {
		response.Status = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, APIResponse{
			Success: false,
			Message: "服务不健康",
			Data:    response,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "服务健康",
		Data:    response,
	})
}

// LivenessProbe Kubernetes存活探针
// @Summary Kubernetes存活探针
// @Description 检查应用是否存活
// @Tags health
// @Produce json
// @Success 200 {object} BasicResponse
// @Router /health/live [get]
func (h *HealthHandler) LivenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, BasicResponse{
		Status:  "ok",
		Message: "应用存活",
	})
}

// ReadinessProbe Kubernetes就绪探针
// @Summary Kubernetes就绪探针
// @Description 检查应用是否就绪
// @Tags health
// @Produce json
// @Success 200 {object} BasicResponse
// @Failure 503 {object} BasicResponse
// @Router /health/ready [get]
func (h *HealthHandler) ReadinessProbe(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 检查关键依赖
	if !h.isReady(ctx) {
		c.JSON(http.StatusServiceUnavailable, BasicResponse{
			Status:  "not_ready",
			Message: "应用未就绪",
		})
		return
	}

	c.JSON(http.StatusOK, BasicResponse{
		Status:  "ok",
		Message: "应用就绪",
	})
}

// checkDatabase 检查数据库健康状态
func (h *HealthHandler) checkDatabase() ComponentHealth {
	health := ComponentHealth{
		Status:      "healthy",
		CheckedAt:   time.Now(),
		ResponseTime: 0,
	}

	start := time.Now()

	// 执行简单的数据库查询
	sqlDB, err := h.db.DB()
	if err != nil {
		health.Status = "unhealthy"
		health.Error = err.Error()
		return health
	}

	if err := sqlDB.Ping(); err != nil {
		health.Status = "unhealthy"
		health.Error = err.Error()
		return health
	}

	health.ResponseTime = time.Since(start).Milliseconds()

	// 检查响应时间阈值
	if health.ResponseTime > 1000 { // 超过1秒认为不健康
		health.Status = "degraded"
		health.Error = "数据库响应时间过长"
	}

	return health
}

// checkTekton 检查Tekton健康状态
func (h *HealthHandler) checkTekton(ctx context.Context) ComponentHealth {
	health := ComponentHealth{
		Status:      "healthy",
		CheckedAt:   time.Now(),
		ResponseTime: 0,
	}

	start := time.Now()

	if err := h.tektonService.HealthCheck(ctx); err != nil {
		health.Status = "unhealthy"
		health.Error = err.Error()
	}

	health.ResponseTime = time.Since(start).Milliseconds()

	// 检查响应时间阈值
	if health.ResponseTime > 2000 { // 超过2秒认为不健康
		health.Status = "degraded"
		health.Error = "Tekton响应时间过长"
	}

	return health
}

// checkDiskSpace 检查磁盘空间
func (h *HealthHandler) checkDiskSpace() ComponentHealth {
	health := ComponentHealth{
		Status:    "healthy",
		CheckedAt: time.Now(),
	}

	// 这里应该实现实际的磁盘空间检查逻辑
	// 为简化，假设总是健康
	health.Details = map[string]interface{}{
		"message": "磁盘空间检查未完全实现",
	}

	return health
}

// isReady 检查应用是否就绪
func (h *HealthHandler) isReady(ctx context.Context) bool {
	// 检查数据库
	sqlDB, err := h.db.DB()
	if err != nil {
		return false
	}

	if err := sqlDB.Ping(); err != nil {
		return false
	}

	// 检查Tekton（可选，因为就绪检查应该更宽松）
	if err := h.tektonService.HealthCheck(ctx); err != nil {
		// Tekton不可用时，应用仍可以提供基本服务
		// 这里可以根据实际需求调整
		return true
	}

	return true
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string                        `json:"status"`
	Timestamp time.Time                     `json:"timestamp"`
	Checks    map[string]ComponentHealth    `json:"checks"`
}

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Status       string                 `json:"status"`
	CheckedAt    time.Time              `json:"checked_at"`
	ResponseTime int64                  `json:"response_time_ms"`
	Error        string                 `json:"error,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// BasicResponse 基础响应
type BasicResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}