package handlers

import (
	"net/http"
	"strconv"

	"project-service/internal/models"
	"project-service/internal/services"
	"project-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProjectHandler 项目处理器
type ProjectHandler struct {
	projectService services.ProjectService
	logger         logger.Logger
}

// NewProjectHandler 创建项目处理器
func NewProjectHandler(projectService services.ProjectService, logger logger.Logger) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
		logger:         logger,
	}
}

// RegisterRoutes 注册路由
func (h *ProjectHandler) RegisterRoutes(r *gin.RouterGroup) {
	projects := r.Group("/projects")
	{
		projects.POST("", h.CreateProject)
		projects.GET("", h.ListProjects)
		projects.GET("/:project_id", h.GetProject)
		projects.GET("/key/:project_key", h.GetProjectByKey)
		projects.PUT("/:project_id", h.UpdateProject)
		projects.DELETE("/:project_id", h.DeleteProject)
		projects.GET("/:project_id/stats", h.GetProjectStats)
		projects.GET("/:project_id/settings", h.GetProjectSettings)
		projects.PUT("/:project_id/settings", h.UpdateProjectSettings)

		// 项目成员管理
		members := projects.Group("/:project_id/members")
		{
			members.GET("", h.ListProjectMembers)
			members.POST("", h.AddProjectMember)
			members.PUT("/:user_id/role", h.UpdateProjectMemberRole)
			members.DELETE("/:user_id", h.RemoveProjectMember)
		}
	}
}

// CreateProject 创建项目
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req services.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// 从JWT中获取租户ID
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}
	req.TenantID = tenantID.(uuid.UUID)

	project, err := h.projectService.CreateProject(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("创建项目失败", "error", err, "tenant_id", req.TenantID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("项目创建成功", "project_id", project.ID, "name", project.Name)
	c.JSON(http.StatusCreated, gin.H{"data": project})
}

// GetProject 获取项目详情
func (h *ProjectHandler) GetProject(c *gin.Context) {
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

	project, err := h.projectService.GetProject(c.Request.Context(), tenantID.(uuid.UUID), projectID)
	if err != nil {
		if err.Error() == "项目不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("获取项目失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": project})
}

// GetProjectByKey 根据项目键获取项目
func (h *ProjectHandler) GetProjectByKey(c *gin.Context) {
	projectKey := c.Param("project_key")
	if projectKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_key required"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	project, err := h.projectService.GetProjectByKey(c.Request.Context(), tenantID.(uuid.UUID), projectKey)
	if err != nil {
		if err.Error() == "项目不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("获取项目失败", "error", err, "project_key", projectKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": project})
}

// UpdateProject 更新项目
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	var req services.UpdateProjectRequest
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
	req.ProjectID = projectID

	project, err := h.projectService.UpdateProject(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "项目不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("更新项目失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": project})
}

// DeleteProject 删除项目
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
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

	err = h.projectService.DeleteProject(c.Request.Context(), tenantID.(uuid.UUID), projectID)
	if err != nil {
		if err.Error() == "项目不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("删除项目失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "项目删除成功"})
}

// ListProjects 获取项目列表
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	req := &services.ListProjectsRequest{
		TenantID: tenantID.(uuid.UUID),
		Page:     1,
		Limit:    20,
		SortBy:   "created_at",
		SortDesc: true,
	}

	// 解析查询参数
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			req.UserID = &userID
		}
	}

	if status := c.Query("status"); status != "" {
		req.Status = &status
	}

	if search := c.Query("search"); search != "" {
		req.Search = &search
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

	response, err := h.projectService.ListProjects(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("获取项目列表失败", "error", err, "tenant_id", req.TenantID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// GetProjectStats 获取项目统计信息
func (h *ProjectHandler) GetProjectStats(c *gin.Context) {
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

	stats, err := h.projectService.GetProjectStats(c.Request.Context(), tenantID.(uuid.UUID), projectID)
	if err != nil {
		h.logger.Error("获取项目统计失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// GetProjectSettings 获取项目设置
func (h *ProjectHandler) GetProjectSettings(c *gin.Context) {
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

	settings, err := h.projectService.GetProjectSettings(c.Request.Context(), tenantID.(uuid.UUID), projectID)
	if err != nil {
		if err.Error() == "项目不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("获取项目设置失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": settings})
}

// UpdateProjectSettings 更新项目设置
func (h *ProjectHandler) UpdateProjectSettings(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	var settings models.ProjectSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	req := &services.UpdateProjectSettingsRequest{
		TenantID:  tenantID.(uuid.UUID),
		ProjectID: projectID,
		Settings:  &settings,
	}

	err = h.projectService.UpdateProjectSettings(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "项目不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("更新项目设置失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "项目设置更新成功"})
}

// ListProjectMembers 获取项目成员列表
func (h *ProjectHandler) ListProjectMembers(c *gin.Context) {
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

	members, err := h.projectService.ListProjectMembers(c.Request.Context(), tenantID.(uuid.UUID), projectID)
	if err != nil {
		h.logger.Error("获取项目成员失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": members})
}

// AddProjectMember 添加项目成员
func (h *ProjectHandler) AddProjectMember(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	var req services.AddProjectMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id required"})
		return
	}

	req.TenantID = tenantID.(uuid.UUID)
	req.ProjectID = projectID
	req.AddedBy = userID.(uuid.UUID)

	err = h.projectService.AddProjectMember(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("添加项目成员失败", "error", err, "project_id", projectID, "user_id", req.UserID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "项目成员添加成功"})
}

// UpdateProjectMemberRole 更新项目成员角色
func (h *ProjectHandler) UpdateProjectMemberRole(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	memberUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var reqBody struct {
		RoleID uuid.UUID `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	req := &services.UpdateProjectMemberRoleRequest{
		TenantID:  tenantID.(uuid.UUID),
		ProjectID: projectID,
		UserID:    memberUserID,
		RoleID:    reqBody.RoleID,
	}

	err = h.projectService.UpdateProjectMemberRole(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("更新项目成员角色失败", "error", err, "project_id", projectID, "user_id", memberUserID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "项目成员角色更新成功"})
}

// RemoveProjectMember 移除项目成员
func (h *ProjectHandler) RemoveProjectMember(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	memberUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	err = h.projectService.RemoveProjectMember(c.Request.Context(), tenantID.(uuid.UUID), projectID, memberUserID)
	if err != nil {
		h.logger.Error("移除项目成员失败", "error", err, "project_id", projectID, "user_id", memberUserID)
		if err.Error() == "项目成员不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "项目成员移除成功"})
}