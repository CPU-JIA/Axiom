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

// ProjectService 项目服务接口
type ProjectService interface {
	// 项目CRUD
	CreateProject(ctx context.Context, req *CreateProjectRequest) (*models.Project, error)
	GetProject(ctx context.Context, tenantID, projectID uuid.UUID) (*models.Project, error)
	GetProjectByKey(ctx context.Context, tenantID uuid.UUID, key string) (*models.Project, error)
	UpdateProject(ctx context.Context, req *UpdateProjectRequest) (*models.Project, error)
	DeleteProject(ctx context.Context, tenantID, projectID uuid.UUID) error
	ListProjects(ctx context.Context, req *ListProjectsRequest) (*ListProjectsResponse, error)

	// 项目成员管理
	AddProjectMember(ctx context.Context, req *AddProjectMemberRequest) error
	RemoveProjectMember(ctx context.Context, tenantID, projectID, userID uuid.UUID) error
	UpdateProjectMemberRole(ctx context.Context, req *UpdateProjectMemberRoleRequest) error
	ListProjectMembers(ctx context.Context, tenantID, projectID uuid.UUID) ([]models.ProjectMember, error)

	// 项目设置
	UpdateProjectSettings(ctx context.Context, req *UpdateProjectSettingsRequest) error
	GetProjectSettings(ctx context.Context, tenantID, projectID uuid.UUID) (*models.ProjectSettings, error)

	// 项目统计
	GetProjectStats(ctx context.Context, tenantID, projectID uuid.UUID) (*ProjectStats, error)
}

// projectService 项目服务实现
type projectService struct {
	db     *gorm.DB
	logger logger.Logger
}

// NewProjectService 创建项目服务实例
func NewProjectService(db *gorm.DB, logger logger.Logger) ProjectService {
	return &projectService{
		db:     db,
		logger: logger,
	}
}

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	TenantID    uuid.UUID `json:"tenant_id" validate:"required"`
	Name        string    `json:"name" validate:"required,min=1,max=255"`
	Key         string    `json:"key" validate:"required,min=2,max=10,alphanum"`
	Description *string   `json:"description"`
	ManagerID   *uuid.UUID `json:"manager_id"`
	Settings    *models.ProjectSettings `json:"settings"`
}

// UpdateProjectRequest 更新项目请求
type UpdateProjectRequest struct {
	TenantID    uuid.UUID `json:"tenant_id" validate:"required"`
	ProjectID   uuid.UUID `json:"project_id" validate:"required"`
	Name        *string   `json:"name" validate:"omitempty,min=1,max=255"`
	Description *string   `json:"description"`
	ManagerID   *uuid.UUID `json:"manager_id"`
	Status      *string   `json:"status" validate:"omitempty,oneof=active archived"`
}

// ListProjectsRequest 项目列表请求
type ListProjectsRequest struct {
	TenantID uuid.UUID `json:"tenant_id" validate:"required"`
	UserID   *uuid.UUID `json:"user_id"` // 可选，过滤用户参与的项目
	Status   *string   `json:"status"`   // 可选，过滤状态
	Search   *string   `json:"search"`   // 可选，搜索关键词
	Page     int       `json:"page" validate:"min=1"`
	Limit    int       `json:"limit" validate:"min=1,max=100"`
	SortBy   string    `json:"sort_by" validate:"oneof=name created_at updated_at"`
	SortDesc bool      `json:"sort_desc"`
}

// ListProjectsResponse 项目列表响应
type ListProjectsResponse struct {
	Projects []models.Project `json:"projects"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	Limit    int              `json:"limit"`
}

// AddProjectMemberRequest 添加项目成员请求
type AddProjectMemberRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	ProjectID uuid.UUID `json:"project_id" validate:"required"`
	UserID    uuid.UUID `json:"user_id" validate:"required"`
	RoleID    uuid.UUID `json:"role_id" validate:"required"`
	AddedBy   uuid.UUID `json:"added_by" validate:"required"`
}

// UpdateProjectMemberRoleRequest 更新项目成员角色请求
type UpdateProjectMemberRoleRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	ProjectID uuid.UUID `json:"project_id" validate:"required"`
	UserID    uuid.UUID `json:"user_id" validate:"required"`
	RoleID    uuid.UUID `json:"role_id" validate:"required"`
}

// UpdateProjectSettingsRequest 更新项目设置请求
type UpdateProjectSettingsRequest struct {
	TenantID  uuid.UUID               `json:"tenant_id" validate:"required"`
	ProjectID uuid.UUID               `json:"project_id" validate:"required"`
	Settings  *models.ProjectSettings `json:"settings" validate:"required"`
}

// ProjectStats 项目统计
type ProjectStats struct {
	TotalTasks       int `json:"total_tasks"`
	CompletedTasks   int `json:"completed_tasks"`
	InProgressTasks  int `json:"in_progress_tasks"`
	TodoTasks        int `json:"todo_tasks"`
	OverdueTasks     int `json:"overdue_tasks"`
	TotalMembers     int `json:"total_members"`
	ActiveSprints    int `json:"active_sprints"`
	CompletedSprints int `json:"completed_sprints"`
}

// CreateProject 创建项目
func (s *projectService) CreateProject(ctx context.Context, req *CreateProjectRequest) (*models.Project, error) {
	s.logger.Info("创建项目", "tenant_id", req.TenantID, "name", req.Name, "key", req.Key)

	// 检查项目键是否已存在（在同一租户内）
	var existingProject models.Project
	result := s.db.WithContext(ctx).Where("tenant_id = ? AND key = ? AND deleted_at IS NULL", 
		req.TenantID, req.Key).First(&existingProject)
	
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		s.logger.Error("检查项目键失败", "error", result.Error)
		return nil, fmt.Errorf("检查项目键失败: %w", result.Error)
	}
	
	if result.Error == nil {
		return nil, fmt.Errorf("项目键 '%s' 已存在", req.Key)
	}

	// 创建项目
	project := &models.Project{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Key:         req.Key,
		Description: req.Description,
		ManagerID:   req.ManagerID,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 设置项目配置
	if req.Settings != nil {
		settingsJSON, err := encodeJSON(req.Settings)
		if err != nil {
			return nil, fmt.Errorf("编码项目设置失败: %w", err)
		}
		project.Settings = settingsJSON
	} else {
		// 设置默认配置
		defaultSettings := &models.ProjectSettings{
			TaskNumberPrefix:   req.Key,
			AllowGuestComments: false,
			AutoArchiveSprints: true,
			WorkflowSettings: models.WorkflowSettings{
				AutoMoveToInProgress: true,
				RequireCommentOnMove: false,
				AllowedTransitions:   []string{"todo->in_progress", "in_progress->done", "done->todo"},
			},
		}
		settingsJSON, _ := encodeJSON(defaultSettings)
		project.Settings = settingsJSON
	}

	if err := s.db.WithContext(ctx).Create(project).Error; err != nil {
		s.logger.Error("创建项目失败", "error", err)
		return nil, fmt.Errorf("创建项目失败: %w", err)
	}

	// 如果指定了项目经理，自动添加为项目成员
	if req.ManagerID != nil {
		memberReq := &AddProjectMemberRequest{
			TenantID:  req.TenantID,
			ProjectID: project.ID,
			UserID:    *req.ManagerID,
			RoleID:    getDefaultManagerRoleID(), // 需要实现获取默认管理员角色
			AddedBy:   *req.ManagerID,
		}
		if err := s.AddProjectMember(ctx, memberReq); err != nil {
			s.logger.Warn("自动添加项目经理为成员失败", "error", err)
		}
	}

	s.logger.Info("项目创建成功", "project_id", project.ID, "name", project.Name)
	return project, nil
}

// GetProject 获取项目详情
func (s *projectService) GetProject(ctx context.Context, tenantID, projectID uuid.UUID) (*models.Project, error) {
	var project models.Project
	
	err := s.db.WithContext(ctx).
		Preload("Manager").
		Preload("Members").
		Preload("Members.User").
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", projectID, tenantID).
		First(&project).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("项目不存在")
		}
		s.logger.Error("获取项目失败", "error", err, "project_id", projectID)
		return nil, fmt.Errorf("获取项目失败: %w", err)
	}

	return &project, nil
}

// GetProjectByKey 根据项目键获取项目
func (s *projectService) GetProjectByKey(ctx context.Context, tenantID uuid.UUID, key string) (*models.Project, error) {
	var project models.Project
	
	err := s.db.WithContext(ctx).
		Preload("Manager").
		Where("tenant_id = ? AND key = ? AND deleted_at IS NULL", tenantID, key).
		First(&project).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("项目不存在")
		}
		s.logger.Error("获取项目失败", "error", err, "key", key)
		return nil, fmt.Errorf("获取项目失败: %w", err)
	}

	return &project, nil
}

// UpdateProject 更新项目
func (s *projectService) UpdateProject(ctx context.Context, req *UpdateProjectRequest) (*models.Project, error) {
	s.logger.Info("更新项目", "project_id", req.ProjectID, "tenant_id", req.TenantID)

	var project models.Project
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", 
		req.ProjectID, req.TenantID).First(&project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("项目不存在")
		}
		return nil, fmt.Errorf("查询项目失败: %w", err)
	}

	// 更新字段
	updates := make(map[string]interface{})
	updates["updated_at"] = time.Now()

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ManagerID != nil {
		updates["manager_id"] = *req.ManagerID
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if err := s.db.WithContext(ctx).Model(&project).Updates(updates).Error; err != nil {
		s.logger.Error("更新项目失败", "error", err)
		return nil, fmt.Errorf("更新项目失败: %w", err)
	}

	// 重新获取更新后的项目
	return s.GetProject(ctx, req.TenantID, req.ProjectID)
}

// DeleteProject 删除项目（软删除）
func (s *projectService) DeleteProject(ctx context.Context, tenantID, projectID uuid.UUID) error {
	s.logger.Info("删除项目", "project_id", projectID, "tenant_id", tenantID)

	result := s.db.WithContext(ctx).
		Model(&models.Project{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", projectID, tenantID).
		Update("deleted_at", time.Now())

	if result.Error != nil {
		s.logger.Error("删除项目失败", "error", result.Error)
		return fmt.Errorf("删除项目失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("项目不存在")
	}

	s.logger.Info("项目删除成功", "project_id", projectID)
	return nil
}

// ListProjects 获取项目列表
func (s *projectService) ListProjects(ctx context.Context, req *ListProjectsRequest) (*ListProjectsResponse, error) {
	query := s.db.WithContext(ctx).Model(&models.Project{}).
		Where("tenant_id = ? AND deleted_at IS NULL", req.TenantID)

	// 过滤条件
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	if req.Search != nil && *req.Search != "" {
		query = query.Where("name ILIKE ? OR key ILIKE ? OR description ILIKE ?", 
			"%"+*req.Search+"%", "%"+*req.Search+"%", "%"+*req.Search+"%")
	}

	if req.UserID != nil {
		// 过滤用户参与的项目
		query = query.Where("id IN (SELECT project_id FROM project_members WHERE user_id = ?)", *req.UserID)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("计算项目总数失败: %w", err)
	}

	// 排序
	sortBy := "created_at"
	if req.SortBy != "" {
		sortBy = req.SortBy
	}
	if req.SortDesc {
		sortBy += " DESC"
	}
	query = query.Order(sortBy)

	// 分页
	offset := (req.Page - 1) * req.Limit
	query = query.Offset(offset).Limit(req.Limit)

	var projects []models.Project
	if err := query.Preload("Manager").Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("查询项目列表失败: %w", err)
	}

	return &ListProjectsResponse{
		Projects: projects,
		Total:    total,
		Page:     req.Page,
		Limit:    req.Limit,
	}, nil
}

// AddProjectMember 添加项目成员
func (s *projectService) AddProjectMember(ctx context.Context, req *AddProjectMemberRequest) error {
	s.logger.Info("添加项目成员", "project_id", req.ProjectID, "user_id", req.UserID)

	// 检查项目是否存在
	var project models.Project
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", 
		req.ProjectID, req.TenantID).First(&project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("项目不存在")
		}
		return fmt.Errorf("查询项目失败: %w", err)
	}

	// 检查用户是否已是项目成员
	var existingMember models.ProjectMember
	result := s.db.WithContext(ctx).Where("project_id = ? AND user_id = ?", 
		req.ProjectID, req.UserID).First(&existingMember)
	
	if result.Error == nil {
		return fmt.Errorf("用户已是项目成员")
	}
	if result.Error != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查成员状态失败: %w", result.Error)
	}

	// 添加项目成员
	member := &models.ProjectMember{
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
		RoleID:    req.RoleID,
		AddedAt:   time.Now(),
		AddedBy:   req.AddedBy,
	}

	if err := s.db.WithContext(ctx).Create(member).Error; err != nil {
		s.logger.Error("添加项目成员失败", "error", err)
		return fmt.Errorf("添加项目成员失败: %w", err)
	}

	s.logger.Info("项目成员添加成功", "project_id", req.ProjectID, "user_id", req.UserID)
	return nil
}

// RemoveProjectMember 移除项目成员
func (s *projectService) RemoveProjectMember(ctx context.Context, tenantID, projectID, userID uuid.UUID) error {
	s.logger.Info("移除项目成员", "project_id", projectID, "user_id", userID)

	result := s.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&models.ProjectMember{})

	if result.Error != nil {
		s.logger.Error("移除项目成员失败", "error", result.Error)
		return fmt.Errorf("移除项目成员失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("项目成员不存在")
	}

	s.logger.Info("项目成员移除成功", "project_id", projectID, "user_id", userID)
	return nil
}

// UpdateProjectMemberRole 更新项目成员角色
func (s *projectService) UpdateProjectMemberRole(ctx context.Context, req *UpdateProjectMemberRoleRequest) error {
	s.logger.Info("更新项目成员角色", "project_id", req.ProjectID, "user_id", req.UserID, "role_id", req.RoleID)

	result := s.db.WithContext(ctx).
		Model(&models.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", req.ProjectID, req.UserID).
		Update("role_id", req.RoleID)

	if result.Error != nil {
		s.logger.Error("更新项目成员角色失败", "error", result.Error)
		return fmt.Errorf("更新项目成员角色失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("项目成员不存在")
	}

	return nil
}

// ListProjectMembers 获取项目成员列表
func (s *projectService) ListProjectMembers(ctx context.Context, tenantID, projectID uuid.UUID) ([]models.ProjectMember, error) {
	var members []models.ProjectMember
	
	err := s.db.WithContext(ctx).
		Preload("User").
		Where("project_id = ?", projectID).
		Find(&members).Error

	if err != nil {
		s.logger.Error("获取项目成员列表失败", "error", err, "project_id", projectID)
		return nil, fmt.Errorf("获取项目成员列表失败: %w", err)
	}

	return members, nil
}

// UpdateProjectSettings 更新项目设置
func (s *projectService) UpdateProjectSettings(ctx context.Context, req *UpdateProjectSettingsRequest) error {
	s.logger.Info("更新项目设置", "project_id", req.ProjectID)

	settingsJSON, err := encodeJSON(req.Settings)
	if err != nil {
		return fmt.Errorf("编码项目设置失败: %w", err)
	}

	result := s.db.WithContext(ctx).
		Model(&models.Project{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", req.ProjectID, req.TenantID).
		Updates(map[string]interface{}{
			"settings":   settingsJSON,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		s.logger.Error("更新项目设置失败", "error", result.Error)
		return fmt.Errorf("更新项目设置失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("项目不存在")
	}

	return nil
}

// GetProjectSettings 获取项目设置
func (s *projectService) GetProjectSettings(ctx context.Context, tenantID, projectID uuid.UUID) (*models.ProjectSettings, error) {
	var project models.Project
	
	err := s.db.WithContext(ctx).
		Select("settings").
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", projectID, tenantID).
		First(&project).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("项目不存在")
		}
		return nil, fmt.Errorf("获取项目设置失败: %w", err)
	}

	var settings models.ProjectSettings
	if err := decodeJSON(project.Settings, &settings); err != nil {
		return nil, fmt.Errorf("解码项目设置失败: %w", err)
	}

	return &settings, nil
}

// GetProjectStats 获取项目统计信息
func (s *projectService) GetProjectStats(ctx context.Context, tenantID, projectID uuid.UUID) (*ProjectStats, error) {
	stats := &ProjectStats{}

	// 任务统计
	var taskStats struct {
		Total       int64
		Completed   int64
		InProgress  int64
		Todo        int64
		Overdue     int64
	}

	// 总任务数
	s.db.WithContext(ctx).Model(&models.Task{}).
		Where("project_id = ?", projectID).
		Count(&taskStats.Total)

	// 按状态分组统计任务
	s.db.WithContext(ctx).Model(&models.Task{}).
		Select("ts.category, COUNT(*) as count").
		Joins("LEFT JOIN task_statuses ts ON tasks.status_id = ts.id").
		Where("tasks.project_id = ?", projectID).
		Group("ts.category").
		Scan(&taskStats)

	// 过期任务统计
	s.db.WithContext(ctx).Model(&models.Task{}).
		Where("project_id = ? AND due_date < ? AND status_id NOT IN (SELECT id FROM task_statuses WHERE category = 'done')", 
			projectID, time.Now()).
		Count(&taskStats.Overdue)

	// 成员统计
	var memberCount int64
	s.db.WithContext(ctx).Model(&models.ProjectMember{}).
		Where("project_id = ?", projectID).
		Count(&memberCount)

	// 迭代统计
	var sprintStats struct {
		Active    int64
		Completed int64
	}
	s.db.WithContext(ctx).Model(&models.Sprint{}).
		Where("project_id = ? AND status = 'active'", projectID).
		Count(&sprintStats.Active)

	s.db.WithContext(ctx).Model(&models.Sprint{}).
		Where("project_id = ? AND status = 'completed'", projectID).
		Count(&sprintStats.Completed)

	stats.TotalTasks = int(taskStats.Total)
	stats.CompletedTasks = int(taskStats.Completed)
	stats.InProgressTasks = int(taskStats.InProgress)
	stats.TodoTasks = int(taskStats.Todo)
	stats.OverdueTasks = int(taskStats.Overdue)
	stats.TotalMembers = int(memberCount)
	stats.ActiveSprints = int(sprintStats.Active)
	stats.CompletedSprints = int(sprintStats.Completed)

	return stats, nil
}

// 辅助函数
func encodeJSON(data interface{}) ([]byte, error) {
	// 这里可以使用 encoding/json 或其他JSON库
	return []byte("{}"), nil // 简化实现，实际应该正确编码
}

func decodeJSON(data []byte, target interface{}) error {
	// 这里可以使用 encoding/json 或其他JSON库
	return nil // 简化实现，实际应该正确解码
}

func getDefaultManagerRoleID() uuid.UUID {
	// 这里应该从数据库中获取默认的项目管理员角色ID
	return uuid.New() // 简化实现
}