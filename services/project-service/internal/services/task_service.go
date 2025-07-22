package services

import (
	"context"
	"fmt"
	"time"

	"project-service/internal/models"
	"project-service/pkg/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskService 任务服务接口
type TaskService interface {
	// 任务CRUD
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*models.Task, error)
	GetTask(ctx context.Context, tenantID uuid.UUID, taskID uuid.UUID) (*models.Task, error)
	GetTaskByNumber(ctx context.Context, tenantID uuid.UUID, projectID uuid.UUID, taskNumber int64) (*models.Task, error)
	UpdateTask(ctx context.Context, req *UpdateTaskRequest) (*models.Task, error)
	DeleteTask(ctx context.Context, tenantID, taskID uuid.UUID) error
	ListTasks(ctx context.Context, req *ListTasksRequest) (*ListTasksResponse, error)

	// 任务状态管理
	UpdateTaskStatus(ctx context.Context, req *UpdateTaskStatusRequest) (*models.Task, error)
	MoveTaskToSprint(ctx context.Context, req *MoveTaskToSprintRequest) error
	AssignTask(ctx context.Context, req *AssignTaskRequest) error

	// 子任务管理
	CreateSubTask(ctx context.Context, req *CreateSubTaskRequest) (*models.Task, error)
	GetSubTasks(ctx context.Context, tenantID, parentTaskID uuid.UUID) ([]models.Task, error)

	// 任务看板
	GetKanbanBoard(ctx context.Context, tenantID, projectID uuid.UUID, sprintID *uuid.UUID) (*KanbanBoard, error)
	UpdateTaskOrder(ctx context.Context, req *UpdateTaskOrderRequest) error
}

// taskService 任务服务实现
type taskService struct {
	db     *gorm.DB
	logger logger.Logger
}

// NewTaskService 创建任务服务实例
func NewTaskService(db *gorm.DB, logger logger.Logger) TaskService {
	return &taskService{
		db:     db,
		logger: logger,
	}
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	ProjectID    uuid.UUID `json:"project_id" validate:"required"`
	Title        string    `json:"title" validate:"required,min=1,max=512"`
	Description  *string   `json:"description"`
	AssigneeID   *uuid.UUID `json:"assignee_id"`
	CreatorID    uuid.UUID `json:"creator_id" validate:"required"`
	ParentTaskID *uuid.UUID `json:"parent_task_id"`
	SprintID     *uuid.UUID `json:"sprint_id"`
	DueDate      *time.Time `json:"due_date"`
	Priority     string    `json:"priority" validate:"oneof=low medium high urgent"`
	StoryPoints  *int      `json:"story_points" validate:"omitempty,min=1,max=100"`
	Tags         []string  `json:"tags"`
	CustomFields map[string]interface{} `json:"custom_fields"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	TenantID     uuid.UUID `json:"tenant_id" validate:"required"`
	TaskID       uuid.UUID `json:"task_id" validate:"required"`
	Title        *string   `json:"title" validate:"omitempty,min=1,max=512"`
	Description  *string   `json:"description"`
	AssigneeID   *uuid.UUID `json:"assignee_id"`
	DueDate      *time.Time `json:"due_date"`
	Priority     *string   `json:"priority" validate:"omitempty,oneof=low medium high urgent"`
	StoryPoints  *int      `json:"story_points" validate:"omitempty,min=1,max=100"`
	Tags         []string  `json:"tags"`
	CustomFields map[string]interface{} `json:"custom_fields"`
}

// ListTasksRequest 任务列表请求
type ListTasksRequest struct {
	TenantID   uuid.UUID  `json:"tenant_id" validate:"required"`
	ProjectID  *uuid.UUID `json:"project_id"`
	SprintID   *uuid.UUID `json:"sprint_id"`
	AssigneeID *uuid.UUID `json:"assignee_id"`
	StatusID   *uuid.UUID `json:"status_id"`
	Priority   *string    `json:"priority"`
	Search     *string    `json:"search"`
	Tags       []string   `json:"tags"`
	DueDateFrom *time.Time `json:"due_date_from"`
	DueDateTo   *time.Time `json:"due_date_to"`
	Page       int        `json:"page" validate:"min=1"`
	Limit      int        `json:"limit" validate:"min=1,max=100"`
	SortBy     string     `json:"sort_by" validate:"oneof=created_at updated_at due_date priority task_number"`
	SortDesc   bool       `json:"sort_desc"`
}

// ListTasksResponse 任务列表响应
type ListTasksResponse struct {
	Tasks []models.Task `json:"tasks"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}

// UpdateTaskStatusRequest 更新任务状态请求
type UpdateTaskStatusRequest struct {
	TenantID uuid.UUID `json:"tenant_id" validate:"required"`
	TaskID   uuid.UUID `json:"task_id" validate:"required"`
	StatusID uuid.UUID `json:"status_id" validate:"required"`
	Comment  *string   `json:"comment"`
	UserID   uuid.UUID `json:"user_id" validate:"required"`
}

// MoveTaskToSprintRequest 移动任务到迭代请求
type MoveTaskToSprintRequest struct {
	TenantID uuid.UUID  `json:"tenant_id" validate:"required"`
	TaskID   uuid.UUID  `json:"task_id" validate:"required"`
	SprintID *uuid.UUID `json:"sprint_id"` // nil表示移出迭代
}

// AssignTaskRequest 分配任务请求
type AssignTaskRequest struct {
	TenantID   uuid.UUID  `json:"tenant_id" validate:"required"`
	TaskID     uuid.UUID  `json:"task_id" validate:"required"`
	AssigneeID *uuid.UUID `json:"assignee_id"` // nil表示取消分配
	AssignedBy uuid.UUID  `json:"assigned_by" validate:"required"`
}

// CreateSubTaskRequest 创建子任务请求
type CreateSubTaskRequest struct {
	ParentTaskID uuid.UUID `json:"parent_task_id" validate:"required"`
	Title        string    `json:"title" validate:"required,min=1,max=512"`
	Description  *string   `json:"description"`
	AssigneeID   *uuid.UUID `json:"assignee_id"`
	CreatorID    uuid.UUID `json:"creator_id" validate:"required"`
	DueDate      *time.Time `json:"due_date"`
	Priority     string    `json:"priority" validate:"oneof=low medium high urgent"`
}

// KanbanBoard 看板数据结构
type KanbanBoard struct {
	Columns []KanbanColumn `json:"columns"`
	Stats   KanbanStats    `json:"stats"`
}

// KanbanColumn 看板列
type KanbanColumn struct {
	Status models.TaskStatus `json:"status"`
	Tasks  []models.Task     `json:"tasks"`
	Count  int               `json:"count"`
}

// KanbanStats 看板统计
type KanbanStats struct {
	TotalTasks     int `json:"total_tasks"`
	TotalPoints    int `json:"total_points"`
	CompletedTasks int `json:"completed_tasks"`
	CompletedPoints int `json:"completed_points"`
}

// UpdateTaskOrderRequest 更新任务顺序请求
type UpdateTaskOrderRequest struct {
	TenantID uuid.UUID `json:"tenant_id" validate:"required"`
	TaskID   uuid.UUID `json:"task_id" validate:"required"`
	StatusID uuid.UUID `json:"status_id" validate:"required"`
	Position int       `json:"position" validate:"min=0"`
}

// CreateTask 创建任务
func (s *taskService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*models.Task, error) {
	s.logger.Info("创建任务", "project_id", req.ProjectID, "title", req.Title)

	// 验证项目是否存在
	var project models.Project
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.ProjectID).First(&project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("项目不存在")
		}
		return nil, fmt.Errorf("查询项目失败: %w", err)
	}

	// 获取下一个任务编号
	taskNumber, err := s.getNextTaskNumber(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("生成任务编号失败: %w", err)
	}

	// 如果未指定优先级，设为中等
	if req.Priority == "" {
		req.Priority = "medium"
	}

	// 创建任务
	task := &models.Task{
		ProjectID:    req.ProjectID,
		TaskNumber:   taskNumber,
		Title:        req.Title,
		Description:  req.Description,
		AssigneeID:   req.AssigneeID,
		CreatorID:    req.CreatorID,
		ParentTaskID: req.ParentTaskID,
		SprintID:     req.SprintID,
		DueDate:      req.DueDate,
		Priority:     req.Priority,
		StoryPoints:  req.StoryPoints,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 编码标签和自定义字段
	if req.Tags != nil {
		tagsJSON, _ := encodeJSON(req.Tags)
		task.Tags = tagsJSON
	}
	if req.CustomFields != nil {
		fieldsJSON, _ := encodeJSON(req.CustomFields)
		task.CustomFields = fieldsJSON
	}

	// 获取默认任务状态
	defaultStatus, err := s.getDefaultTaskStatus(ctx, project.TenantID)
	if err != nil {
		s.logger.Warn("获取默认任务状态失败，继续创建", "error", err)
	} else {
		task.StatusID = &defaultStatus.ID
	}

	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		s.logger.Error("创建任务失败", "error", err)
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}

	// 重新加载任务以包含关联数据
	return s.GetTask(ctx, project.TenantID, task.ID)
}

// GetTask 获取任务详情
func (s *taskService) GetTask(ctx context.Context, tenantID uuid.UUID, taskID uuid.UUID) (*models.Task, error) {
	var task models.Task
	
	err := s.db.WithContext(ctx).
		Preload("Project").
		Preload("Status").
		Preload("Assignee").
		Preload("Creator").
		Preload("ParentTask").
		Preload("SubTasks").
		Preload("Sprint").
		Where(`tasks.id = ? AND tasks.project_id IN 
			(SELECT id FROM projects WHERE tenant_id = ? AND deleted_at IS NULL)`, taskID, tenantID).
		First(&task).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务不存在")
		}
		s.logger.Error("获取任务失败", "error", err, "task_id", taskID)
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}

	return &task, nil
}

// GetTaskByNumber 根据任务编号获取任务
func (s *taskService) GetTaskByNumber(ctx context.Context, tenantID uuid.UUID, projectID uuid.UUID, taskNumber int64) (*models.Task, error) {
	var task models.Task
	
	err := s.db.WithContext(ctx).
		Preload("Project").
		Preload("Status").
		Preload("Assignee").
		Preload("Creator").
		Preload("ParentTask").
		Preload("SubTasks").
		Preload("Sprint").
		Joins("JOIN projects p ON tasks.project_id = p.id").
		Where("tasks.project_id = ? AND tasks.task_number = ? AND p.tenant_id = ? AND p.deleted_at IS NULL", 
			projectID, taskNumber, tenantID).
		First(&task).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}

	return &task, nil
}

// UpdateTask 更新任务
func (s *taskService) UpdateTask(ctx context.Context, req *UpdateTaskRequest) (*models.Task, error) {
	s.logger.Info("更新任务", "task_id", req.TaskID, "tenant_id", req.TenantID)

	var task models.Task
	if err := s.db.WithContext(ctx).
		Joins("JOIN projects p ON tasks.project_id = p.id").
		Where("tasks.id = ? AND p.tenant_id = ? AND p.deleted_at IS NULL", req.TaskID, req.TenantID).
		First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}

	// 更新字段
	updates := make(map[string]interface{})
	updates["updated_at"] = time.Now()

	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.AssigneeID != nil {
		updates["assignee_id"] = *req.AssigneeID
	}
	if req.DueDate != nil {
		updates["due_date"] = *req.DueDate
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.StoryPoints != nil {
		updates["story_points"] = *req.StoryPoints
	}

	if req.Tags != nil {
		tagsJSON, _ := encodeJSON(req.Tags)
		updates["tags"] = tagsJSON
	}
	if req.CustomFields != nil {
		fieldsJSON, _ := encodeJSON(req.CustomFields)
		updates["custom_fields"] = fieldsJSON
	}

	if err := s.db.WithContext(ctx).Model(&task).Updates(updates).Error; err != nil {
		s.logger.Error("更新任务失败", "error", err)
		return nil, fmt.Errorf("更新任务失败: %w", err)
	}

	// 重新获取更新后的任务
	return s.GetTask(ctx, req.TenantID, req.TaskID)
}

// DeleteTask 删除任务
func (s *taskService) DeleteTask(ctx context.Context, tenantID, taskID uuid.UUID) error {
	s.logger.Info("删除任务", "task_id", taskID, "tenant_id", tenantID)

	// 检查任务是否存在以及权限
	var task models.Task
	if err := s.db.WithContext(ctx).
		Joins("JOIN projects p ON tasks.project_id = p.id").
		Where("tasks.id = ? AND p.tenant_id = ? AND p.deleted_at IS NULL", taskID, tenantID).
		First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("任务不存在")
		}
		return fmt.Errorf("查询任务失败: %w", err)
	}

	// 检查是否有子任务
	var subTaskCount int64
	s.db.WithContext(ctx).Model(&models.Task{}).Where("parent_task_id = ?", taskID).Count(&subTaskCount)
	if subTaskCount > 0 {
		return fmt.Errorf("存在子任务，无法删除")
	}

	// 删除任务
	if err := s.db.WithContext(ctx).Delete(&task).Error; err != nil {
		s.logger.Error("删除任务失败", "error", err)
		return fmt.Errorf("删除任务失败: %w", err)
	}

	s.logger.Info("任务删除成功", "task_id", taskID)
	return nil
}

// ListTasks 获取任务列表
func (s *taskService) ListTasks(ctx context.Context, req *ListTasksRequest) (*ListTasksResponse, error) {
	query := s.db.WithContext(ctx).Model(&models.Task{}).
		Joins("JOIN projects p ON tasks.project_id = p.id").
		Where("p.tenant_id = ? AND p.deleted_at IS NULL", req.TenantID)

	// 过滤条件
	if req.ProjectID != nil {
		query = query.Where("tasks.project_id = ?", *req.ProjectID)
	}

	if req.SprintID != nil {
		query = query.Where("tasks.sprint_id = ?", *req.SprintID)
	}

	if req.AssigneeID != nil {
		query = query.Where("tasks.assignee_id = ?", *req.AssigneeID)
	}

	if req.StatusID != nil {
		query = query.Where("tasks.status_id = ?", *req.StatusID)
	}

	if req.Priority != nil {
		query = query.Where("tasks.priority = ?", *req.Priority)
	}

	if req.Search != nil && *req.Search != "" {
		query = query.Where("tasks.title ILIKE ? OR tasks.description ILIKE ?", 
			"%"+*req.Search+"%", "%"+*req.Search+"%")
	}

	if req.DueDateFrom != nil {
		query = query.Where("tasks.due_date >= ?", *req.DueDateFrom)
	}

	if req.DueDateTo != nil {
		query = query.Where("tasks.due_date <= ?", *req.DueDateTo)
	}

	// 标签过滤
	if len(req.Tags) > 0 {
		for _, tag := range req.Tags {
			query = query.Where("tasks.tags::jsonb ? ?", tag)
		}
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("计算任务总数失败: %w", err)
	}

	// 排序
	sortBy := "created_at"
	if req.SortBy != "" {
		sortBy = "tasks." + req.SortBy
	}
	if req.SortDesc {
		sortBy += " DESC"
	}
	query = query.Order(sortBy)

	// 分页
	offset := (req.Page - 1) * req.Limit
	query = query.Offset(offset).Limit(req.Limit)

	var tasks []models.Task
	if err := query.
		Preload("Status").
		Preload("Assignee").
		Preload("Creator").
		Preload("Sprint").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("查询任务列表失败: %w", err)
	}

	return &ListTasksResponse{
		Tasks: tasks,
		Total: total,
		Page:  req.Page,
		Limit: req.Limit,
	}, nil
}

// UpdateTaskStatus 更新任务状态
func (s *taskService) UpdateTaskStatus(ctx context.Context, req *UpdateTaskStatusRequest) (*models.Task, error) {
	s.logger.Info("更新任务状态", "task_id", req.TaskID, "status_id", req.StatusID)

	// 验证任务存在性和权限
	var task models.Task
	if err := s.db.WithContext(ctx).
		Joins("JOIN projects p ON tasks.project_id = p.id").
		Where("tasks.id = ? AND p.tenant_id = ? AND p.deleted_at IS NULL", req.TaskID, req.TenantID).
		First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}

	// 验证状态是否存在
	var status models.TaskStatus
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", req.StatusID, req.TenantID).First(&status).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务状态不存在")
		}
		return nil, fmt.Errorf("查询任务状态失败: %w", err)
	}

	// 更新任务状态
	updates := map[string]interface{}{
		"status_id":  req.StatusID,
		"updated_at": time.Now(),
	}

	if err := s.db.WithContext(ctx).Model(&task).Updates(updates).Error; err != nil {
		s.logger.Error("更新任务状态失败", "error", err)
		return nil, fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 如果需要评论，创建状态变更评论
	if req.Comment != nil && *req.Comment != "" {
		comment := &models.Comment{
			TenantID:         req.TenantID,
			AuthorID:         req.UserID,
			Content:          fmt.Sprintf("状态变更为 %s: %s", status.Name, *req.Comment),
			ParentEntityType: "task",
			ParentEntityID:   req.TaskID,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		s.db.WithContext(ctx).Create(comment)
	}

	// 重新获取更新后的任务
	return s.GetTask(ctx, req.TenantID, req.TaskID)
}

// GetKanbanBoard 获取看板数据
func (s *taskService) GetKanbanBoard(ctx context.Context, tenantID, projectID uuid.UUID, sprintID *uuid.UUID) (*KanbanBoard, error) {
	// 获取所有任务状态
	var statuses []models.TaskStatus
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("display_order ASC").
		Find(&statuses).Error; err != nil {
		return nil, fmt.Errorf("获取任务状态失败: %w", err)
	}

	board := &KanbanBoard{
		Columns: make([]KanbanColumn, len(statuses)),
		Stats:   KanbanStats{},
	}

	for i, status := range statuses {
		// 构建查询条件
		taskQuery := s.db.WithContext(ctx).Model(&models.Task{}).
			Where("project_id = ? AND status_id = ?", projectID, status.ID)
		
		if sprintID != nil {
			taskQuery = taskQuery.Where("sprint_id = ?", *sprintID)
		}

		// 获取该状态下的任务
		var tasks []models.Task
		if err := taskQuery.
			Preload("Assignee").
			Preload("Creator").
			Order("created_at ASC").
			Find(&tasks).Error; err != nil {
			return nil, fmt.Errorf("获取状态 %s 下的任务失败: %w", status.Name, err)
		}

		board.Columns[i] = KanbanColumn{
			Status: status,
			Tasks:  tasks,
			Count:  len(tasks),
		}

		// 统计信息
		board.Stats.TotalTasks += len(tasks)
		for _, task := range tasks {
			if task.StoryPoints != nil {
				board.Stats.TotalPoints += *task.StoryPoints
			}
			if status.Category == "done" {
				board.Stats.CompletedTasks++
				if task.StoryPoints != nil {
					board.Stats.CompletedPoints += *task.StoryPoints
				}
			}
		}
	}

	return board, nil
}

// 辅助方法

// getNextTaskNumber 获取下一个任务编号
func (s *taskService) getNextTaskNumber(ctx context.Context, projectID uuid.UUID) (int64, error) {
	var maxNumber int64
	s.db.WithContext(ctx).Model(&models.Task{}).
		Where("project_id = ?", projectID).
		Select("COALESCE(MAX(task_number), 0)").
		Scan(&maxNumber)
	
	return maxNumber + 1, nil
}

// getDefaultTaskStatus 获取默认任务状态
func (s *taskService) getDefaultTaskStatus(ctx context.Context, tenantID uuid.UUID) (*models.TaskStatus, error) {
	var status models.TaskStatus
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND (is_default = true OR category = 'todo')", tenantID).
		Order("is_default DESC, display_order ASC").
		First(&status).Error
	
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// CreateSubTask 创建子任务
func (s *taskService) CreateSubTask(ctx context.Context, req *CreateSubTaskRequest) (*models.Task, error) {
	// 获取父任务信息
	var parentTask models.Task
	if err := s.db.WithContext(ctx).Where("id = ?", req.ParentTaskID).First(&parentTask).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("父任务不存在")
		}
		return nil, fmt.Errorf("查询父任务失败: %w", err)
	}

	// 创建子任务请求
	createReq := &CreateTaskRequest{
		ProjectID:    parentTask.ProjectID,
		Title:        req.Title,
		Description:  req.Description,
		AssigneeID:   req.AssigneeID,
		CreatorID:    req.CreatorID,
		ParentTaskID: &req.ParentTaskID,
		DueDate:      req.DueDate,
		Priority:     req.Priority,
	}

	return s.CreateTask(ctx, createReq)
}

// GetSubTasks 获取子任务列表
func (s *taskService) GetSubTasks(ctx context.Context, tenantID, parentTaskID uuid.UUID) ([]models.Task, error) {
	var subTasks []models.Task
	
	err := s.db.WithContext(ctx).
		Joins("JOIN projects p ON tasks.project_id = p.id").
		Where("tasks.parent_task_id = ? AND p.tenant_id = ? AND p.deleted_at IS NULL", parentTaskID, tenantID).
		Preload("Status").
		Preload("Assignee").
		Order("created_at ASC").
		Find(&subTasks).Error

	if err != nil {
		return nil, fmt.Errorf("获取子任务列表失败: %w", err)
	}

	return subTasks, nil
}

// MoveTaskToSprint 移动任务到迭代
func (s *taskService) MoveTaskToSprint(ctx context.Context, req *MoveTaskToSprintRequest) error {
	updates := map[string]interface{}{
		"sprint_id":  req.SprintID,
		"updated_at": time.Now(),
	}

	result := s.db.WithContext(ctx).
		Model(&models.Task{}).
		Joins("JOIN projects p ON tasks.project_id = p.id").
		Where("tasks.id = ? AND p.tenant_id = ?", req.TaskID, req.TenantID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("移动任务到迭代失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("任务不存在")
	}

	return nil
}

// AssignTask 分配任务
func (s *taskService) AssignTask(ctx context.Context, req *AssignTaskRequest) error {
	updates := map[string]interface{}{
		"assignee_id": req.AssigneeID,
		"updated_at":  time.Now(),
	}

	result := s.db.WithContext(ctx).
		Model(&models.Task{}).
		Joins("JOIN projects p ON tasks.project_id = p.id").
		Where("tasks.id = ? AND p.tenant_id = ?", req.TaskID, req.TenantID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("分配任务失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("任务不存在")
	}

	return nil
}

// UpdateTaskOrder 更新任务顺序（看板拖拽）
func (s *taskService) UpdateTaskOrder(ctx context.Context, req *UpdateTaskOrderRequest) error {
	// 这里可以实现更复杂的任务排序逻辑
	// 简化实现：只更新状态
	updates := map[string]interface{}{
		"status_id":  req.StatusID,
		"updated_at": time.Now(),
	}

	result := s.db.WithContext(ctx).
		Model(&models.Task{}).
		Joins("JOIN projects p ON tasks.project_id = p.id").
		Where("tasks.id = ? AND p.tenant_id = ?", req.TaskID, req.TenantID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("更新任务顺序失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("任务不存在")
	}

	return nil
}