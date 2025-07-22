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

// TaskHandler 任务处理器
type TaskHandler struct {
	taskService services.TaskService
	logger      logger.Logger
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(taskService services.TaskService, logger logger.Logger) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
		logger:      logger,
	}
}

// RegisterRoutes 注册路由
func (h *TaskHandler) RegisterRoutes(r *gin.RouterGroup) {
	tasks := r.Group("/tasks")
	{
		tasks.POST("", h.CreateTask)
		tasks.GET("", h.ListTasks)
		tasks.GET("/:task_id", h.GetTask)
		tasks.GET("/number/:project_id/:task_number", h.GetTaskByNumber)
		tasks.PUT("/:task_id", h.UpdateTask)
		tasks.DELETE("/:task_id", h.DeleteTask)
		tasks.PUT("/:task_id/status", h.UpdateTaskStatus)
		tasks.PUT("/:task_id/assign", h.AssignTask)
		tasks.PUT("/:task_id/sprint", h.MoveTaskToSprint)

		// 子任务管理
		tasks.POST("/:parent_task_id/subtasks", h.CreateSubTask)
		tasks.GET("/:parent_task_id/subtasks", h.GetSubTasks)

		// 看板相关
		tasks.PUT("/:task_id/order", h.UpdateTaskOrder)
	}

	// 项目级任务路由
	projects := r.Group("/projects/:project_id")
	{
		projects.GET("/kanban", h.GetKanbanBoard)
		projects.GET("/tasks", h.GetProjectTasks)
	}
}

// CreateTask 创建任务
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req services.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// 从JWT中获取创建者ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id required"})
		return
	}
	req.CreatorID = userID.(uuid.UUID)

	task, err := h.taskService.CreateTask(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("创建任务失败", "error", err, "project_id", req.ProjectID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("任务创建成功", "task_id", task.ID, "title", task.Title)
	c.JSON(http.StatusCreated, gin.H{"data": task})
}

// GetTask 获取任务详情
func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	task, err := h.taskService.GetTask(c.Request.Context(), tenantID.(uuid.UUID), taskID)
	if err != nil {
		if err.Error() == "任务不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("获取任务失败", "error", err, "task_id", taskID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": task})
}

// GetTaskByNumber 根据任务编号获取任务
func (h *TaskHandler) GetTaskByNumber(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	taskNumber, err := strconv.ParseInt(c.Param("task_number"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_number"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	task, err := h.taskService.GetTaskByNumber(c.Request.Context(), tenantID.(uuid.UUID), projectID, taskNumber)
	if err != nil {
		if err.Error() == "任务不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("获取任务失败", "error", err, "project_id", projectID, "task_number", taskNumber)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": task})
}

// UpdateTask 更新任务
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_id"})
		return
	}

	var req services.UpdateTaskRequest
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
	req.TaskID = taskID

	task, err := h.taskService.UpdateTask(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "任务不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("更新任务失败", "error", err, "task_id", taskID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": task})
}

// DeleteTask 删除任务
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	err = h.taskService.DeleteTask(c.Request.Context(), tenantID.(uuid.UUID), taskID)
	if err != nil {
		if err.Error() == "任务不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "存在子任务，无法删除" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("删除任务失败", "error", err, "task_id", taskID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "任务删除成功"})
}

// ListTasks 获取任务列表
func (h *TaskHandler) ListTasks(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	req := &services.ListTasksRequest{
		TenantID: tenantID.(uuid.UUID),
		Page:     1,
		Limit:    20,
		SortBy:   "created_at",
		SortDesc: true,
	}

	// 解析查询参数
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if projectID, err := uuid.Parse(projectIDStr); err == nil {
			req.ProjectID = &projectID
		}
	}

	if sprintIDStr := c.Query("sprint_id"); sprintIDStr != "" {
		if sprintID, err := uuid.Parse(sprintIDStr); err == nil {
			req.SprintID = &sprintID
		}
	}

	if assigneeIDStr := c.Query("assignee_id"); assigneeIDStr != "" {
		if assigneeID, err := uuid.Parse(assigneeIDStr); err == nil {
			req.AssigneeID = &assigneeID
		}
	}

	if statusIDStr := c.Query("status_id"); statusIDStr != "" {
		if statusID, err := uuid.Parse(statusIDStr); err == nil {
			req.StatusID = &statusID
		}
	}

	if priority := c.Query("priority"); priority != "" {
		req.Priority = &priority
	}

	if search := c.Query("search"); search != "" {
		req.Search = &search
	}

	if dueDateFromStr := c.Query("due_date_from"); dueDateFromStr != "" {
		if dueDateFrom, err := time.Parse("2006-01-02", dueDateFromStr); err == nil {
			req.DueDateFrom = &dueDateFrom
		}
	}

	if dueDateToStr := c.Query("due_date_to"); dueDateToStr != "" {
		if dueDateTo, err := time.Parse("2006-01-02", dueDateToStr); err == nil {
			req.DueDateTo = &dueDateTo
		}
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

	response, err := h.taskService.ListTasks(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("获取任务列表失败", "error", err, "tenant_id", req.TenantID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// UpdateTaskStatus 更新任务状态
func (h *TaskHandler) UpdateTaskStatus(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_id"})
		return
	}

	var reqBody struct {
		StatusID uuid.UUID `json:"status_id" binding:"required"`
		Comment  *string   `json:"comment"`
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

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id required"})
		return
	}

	req := &services.UpdateTaskStatusRequest{
		TenantID: tenantID.(uuid.UUID),
		TaskID:   taskID,
		StatusID: reqBody.StatusID,
		Comment:  reqBody.Comment,
		UserID:   userID.(uuid.UUID),
	}

	task, err := h.taskService.UpdateTaskStatus(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "任务不存在" || err.Error() == "任务状态不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("更新任务状态失败", "error", err, "task_id", taskID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": task})
}

// AssignTask 分配任务
func (h *TaskHandler) AssignTask(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_id"})
		return
	}

	var reqBody struct {
		AssigneeID *uuid.UUID `json:"assignee_id"`
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

	assignedByID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id required"})
		return
	}

	req := &services.AssignTaskRequest{
		TenantID:   tenantID.(uuid.UUID),
		TaskID:     taskID,
		AssigneeID: reqBody.AssigneeID,
		AssignedBy: assignedByID.(uuid.UUID),
	}

	err = h.taskService.AssignTask(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "任务不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("分配任务失败", "error", err, "task_id", taskID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := "任务分配成功"
	if reqBody.AssigneeID == nil {
		message = "任务取消分配成功"
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// MoveTaskToSprint 移动任务到迭代
func (h *TaskHandler) MoveTaskToSprint(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_id"})
		return
	}

	var reqBody struct {
		SprintID *uuid.UUID `json:"sprint_id"`
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

	req := &services.MoveTaskToSprintRequest{
		TenantID: tenantID.(uuid.UUID),
		TaskID:   taskID,
		SprintID: reqBody.SprintID,
	}

	err = h.taskService.MoveTaskToSprint(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "任务不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("移动任务到迭代失败", "error", err, "task_id", taskID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := "任务移动到迭代成功"
	if reqBody.SprintID == nil {
		message = "任务移出迭代成功"
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// CreateSubTask 创建子任务
func (h *TaskHandler) CreateSubTask(c *gin.Context) {
	parentTaskID, err := uuid.Parse(c.Param("parent_task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_task_id"})
		return
	}

	var req services.CreateSubTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id required"})
		return
	}

	req.ParentTaskID = parentTaskID
	req.CreatorID = userID.(uuid.UUID)

	if req.Priority == "" {
		req.Priority = "medium"
	}

	task, err := h.taskService.CreateSubTask(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("创建子任务失败", "error", err, "parent_task_id", parentTaskID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("子任务创建成功", "task_id", task.ID, "parent_task_id", parentTaskID)
	c.JSON(http.StatusCreated, gin.H{"data": task})
}

// GetSubTasks 获取子任务列表
func (h *TaskHandler) GetSubTasks(c *gin.Context) {
	parentTaskID, err := uuid.Parse(c.Param("parent_task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_task_id"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	subTasks, err := h.taskService.GetSubTasks(c.Request.Context(), tenantID.(uuid.UUID), parentTaskID)
	if err != nil {
		h.logger.Error("获取子任务失败", "error", err, "parent_task_id", parentTaskID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": subTasks})
}

// UpdateTaskOrder 更新任务顺序（看板拖拽）
func (h *TaskHandler) UpdateTaskOrder(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_id"})
		return
	}

	var reqBody struct {
		StatusID uuid.UUID `json:"status_id" binding:"required"`
		Position int       `json:"position" binding:"min=0"`
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

	req := &services.UpdateTaskOrderRequest{
		TenantID: tenantID.(uuid.UUID),
		TaskID:   taskID,
		StatusID: reqBody.StatusID,
		Position: reqBody.Position,
	}

	err = h.taskService.UpdateTaskOrder(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "任务不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("更新任务顺序失败", "error", err, "task_id", taskID)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "任务顺序更新成功"})
}

// GetKanbanBoard 获取看板数据
func (h *TaskHandler) GetKanbanBoard(c *gin.Context) {
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

	var sprintID *uuid.UUID
	if sprintIDStr := c.Query("sprint_id"); sprintIDStr != "" {
		if parsedSprintID, err := uuid.Parse(sprintIDStr); err == nil {
			sprintID = &parsedSprintID
		}
	}

	board, err := h.taskService.GetKanbanBoard(c.Request.Context(), tenantID.(uuid.UUID), projectID, sprintID)
	if err != nil {
		h.logger.Error("获取看板数据失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": board})
}

// GetProjectTasks 获取项目任务（项目级路由）
func (h *TaskHandler) GetProjectTasks(c *gin.Context) {
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

	req := &services.ListTasksRequest{
		TenantID:  tenantID.(uuid.UUID),
		ProjectID: &projectID,
		Page:      1,
		Limit:     50,
		SortBy:    "task_number",
		SortDesc:  false,
	}

	// 解析其他查询参数
	if sprintIDStr := c.Query("sprint_id"); sprintIDStr != "" {
		if sprintID, err := uuid.Parse(sprintIDStr); err == nil {
			req.SprintID = &sprintID
		}
	}

	if assigneeIDStr := c.Query("assignee_id"); assigneeIDStr != "" {
		if assigneeID, err := uuid.Parse(assigneeIDStr); err == nil {
			req.AssigneeID = &assigneeID
		}
	}

	if statusIDStr := c.Query("status_id"); statusIDStr != "" {
		if statusID, err := uuid.Parse(statusIDStr); err == nil {
			req.StatusID = &statusID
		}
	}

	response, err := h.taskService.ListTasks(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("获取项目任务失败", "error", err, "project_id", projectID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}