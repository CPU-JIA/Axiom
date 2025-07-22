package handlers

import (
	"net/http"
	"strconv"
	"time"

	"project-service/internal/services"
	"project-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SprintHandler 迭代处理器
type SprintHandler struct {
	sprintService services.SprintService
	logger        logger.Logger
}

// NewSprintHandler 创建迭代处理器
func NewSprintHandler(sprintService services.SprintService, logger logger.Logger) *SprintHandler {
	return &SprintHandler{
		sprintService: sprintService,
		logger:        logger,
	}
}

// RegisterRoutes 注册路由
func (h *SprintHandler) RegisterRoutes(r *gin.RouterGroup) {
	sprints := r.Group("/sprints")
	{
		sprints.POST("", h.CreateSprint)
		sprints.GET("", h.ListSprints)
		sprints.GET("/:sprint_id", h.GetSprint)
		sprints.PUT("/:sprint_id", h.UpdateSprint)
		sprints.DELETE("/:sprint_id", h.DeleteSprint)
		sprints.POST("/:sprint_id/start", h.StartSprint)
		sprints.POST("/:sprint_id/complete", h.CompleteSprint)

		// 迭代报告
		sprints.GET("/:sprint_id/report", h.GetSprintReport)
		sprints.GET("/:sprint_id/burndown", h.GetBurndownChart)
	}

	// 项目级迭代路由
	projects := r.Group("/projects/:project_id")
	{
		projects.GET("/sprints", h.GetProjectSprints)
		projects.GET("/sprints/active", h.GetActiveSprint)
		projects.GET("/velocity", h.GetVelocityChart)
	}
}

// CreateSprint 创建迭代
func (h *SprintHandler) CreateSprint(c *gin.Context) {
	var req services.CreateSprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	sprint, err := h.sprintService.CreateSprint(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("创建迭代失败", "error", err, "project_id", req.ProjectID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("迭代创建成功", "sprint_id", sprint.ID, "name", sprint.Name)
	c.JSON(http.StatusCreated, gin.H{"data": sprint})
}

// GetSprint 获取迭代详情
func (h *SprintHandler) GetSprint(c *gin.Context) {
	sprintID, err := uuid.Parse(c.Param("sprint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sprint_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	sprint, err := h.sprintService.GetSprint(c.Request.Context(), tenantID.(uuid.UUID), sprintID)
	if err != nil {
		if err.Error() == "迭代不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("获取迭代失败", "error", err, "sprint_id", sprintID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sprint})
}

// UpdateSprint 更新迭代
func (h *SprintHandler) UpdateSprint(c *gin.Context) {
	sprintID, err := uuid.Parse(c.Param("sprint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sprint_id"})
		return
	}

	var req services.UpdateSprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	req.TenantID = tenantID.(uuid.UUID)
	req.SprintID = sprintID

	sprint, err := h.sprintService.UpdateSprint(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "迭代不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "活跃迭代不能修改时间范围" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("更新迭代失败", "error", err, "sprint_id", sprintID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sprint})
}

// DeleteSprint 删除迭代
func (h *SprintHandler) DeleteSprint(c *gin.Context) {
	sprintID, err := uuid.Parse(c.Param("sprint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sprint_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	err = h.sprintService.DeleteSprint(c.Request.Context(), tenantID.(uuid.UUID), sprintID)
	if err != nil {
		if err.Error() == "迭代不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "不能删除活跃迭代" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("删除迭代失败", "error", err, "sprint_id", sprintID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "迭代删除成功"})
}

// ListSprints 获取迭代列表
func (h *SprintHandler) ListSprints(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id required"})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	req := &services.ListSprintsRequest{
		TenantID:  tenantID.(uuid.UUID),
		ProjectID: projectID,
		Page:      1,
		Limit:     20,
		SortBy:    "created_at",
		SortDesc:  true,
	}

	// 解析查询参数
	if status := c.Query("status"); status != "" {
		req.Status = &status
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			req.Limit = limit
		}
	}

	if sortBy := c.Query("sort_by"); sortBy != "" {
		req.SortBy = sortBy
	}

	if sortDesc := c.Query("sort_desc"); sortDesc == "true" {
		req.SortDesc = true
	}

	response, err := h.sprintService.ListSprints(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("获取迭代列表失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// StartSprint 开始迭代
func (h *SprintHandler) StartSprint(c *gin.Context) {
	sprintID, err := uuid.Parse(c.Param("sprint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sprint_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	req := &services.StartSprintRequest{
		TenantID: tenantID.(uuid.UUID),
		SprintID: sprintID,
	}

	sprint, err := h.sprintService.StartSprint(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "迭代不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "只有计划状态的迭代可以开始" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err.Error()[:8] == "项目已有活跃迭代" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("开始迭代失败", "error", err, "sprint_id", sprintID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("迭代开始成功", "sprint_id", sprintID)
	c.JSON(http.StatusOK, gin.H{"data": sprint})
}

// CompleteSprint 完成迭代
func (h *SprintHandler) CompleteSprint(c *gin.Context) {
	sprintID, err := uuid.Parse(c.Param("sprint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sprint_id"})
		return
	}

	var req services.CompleteSprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	req.TenantID = tenantID.(uuid.UUID)
	req.SprintID = sprintID

	result, err := h.sprintService.CompleteSprint(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "迭代不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "只有活跃迭代可以完成" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("完成迭代失败", "error", err, "sprint_id", sprintID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("迭代完成成功", "sprint_id", sprintID)
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetSprintReport 获取迭代报告
func (h *SprintHandler) GetSprintReport(c *gin.Context) {
	sprintID, err := uuid.Parse(c.Param("sprint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sprint_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	report, err := h.sprintService.GetSprintReport(c.Request.Context(), tenantID.(uuid.UUID), sprintID)
	if err != nil {
		if err.Error() == "迭代不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("获取迭代报告失败", "error", err, "sprint_id", sprintID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// GetBurndownChart 获取燃尽图
func (h *SprintHandler) GetBurndownChart(c *gin.Context) {
	sprintID, err := uuid.Parse(c.Param("sprint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sprint_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	chart, err := h.sprintService.GetBurndownChart(c.Request.Context(), tenantID.(uuid.UUID), sprintID)
	if err != nil {
		if err.Error() == "迭代不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("获取燃尽图失败", "error", err, "sprint_id", sprintID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": chart})
}

// GetProjectSprints 获取项目迭代列表（项目级路由）
func (h *SprintHandler) GetProjectSprints(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	req := &services.ListSprintsRequest{
		TenantID:  tenantID.(uuid.UUID),
		ProjectID: projectID,
		Page:      1,
		Limit:     50,
		SortBy:    "start_date",
		SortDesc:  true,
	}

	// 解析查询参数
	if status := c.Query("status"); status != "" {
		req.Status = &status
	}

	response, err := h.sprintService.ListSprints(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("获取项目迭代失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// GetActiveSprint 获取项目的活跃迭代
func (h *SprintHandler) GetActiveSprint(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	sprint, err := h.sprintService.GetActiveSprint(c.Request.Context(), tenantID.(uuid.UUID), projectID)
	if err != nil {
		h.logger.Error("获取活跃迭代失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	if sprint == nil {
		c.JSON(http.StatusOK, gin.H{"data": nil, "message": "没有活跃迭代"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sprint})
}

// GetVelocityChart 获取速率图
func (h *SprintHandler) GetVelocityChart(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	sprintCount := 10 // 默认显示最近10个迭代
	if sprintCountStr := c.Query("sprint_count"); sprintCountStr != "" {
		if count, err := strconv.Atoi(sprintCountStr); err == nil && count > 0 && count <= 50 {
			sprintCount = count
		}
	}

	chart, err := h.sprintService.GetVelocityChart(c.Request.Context(), tenantID.(uuid.UUID), projectID, sprintCount)
	if err != nil {
		h.logger.Error("获取速率图失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": chart})
}