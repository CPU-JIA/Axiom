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

// SprintService 迭代服务接口
type SprintService interface {
	// 迭代CRUD
	CreateSprint(ctx context.Context, req *CreateSprintRequest) (*models.Sprint, error)
	GetSprint(ctx context.Context, tenantID, sprintID uuid.UUID) (*models.Sprint, error)
	UpdateSprint(ctx context.Context, req *UpdateSprintRequest) (*models.Sprint, error)
	DeleteSprint(ctx context.Context, tenantID, sprintID uuid.UUID) error
	ListSprints(ctx context.Context, req *ListSprintsRequest) (*ListSprintsResponse, error)

	// 迭代状态管理
	StartSprint(ctx context.Context, req *StartSprintRequest) (*models.Sprint, error)
	CompleteSprint(ctx context.Context, req *CompleteSprintRequest) (*SprintCompletionResult, error)
	GetActiveSprint(ctx context.Context, tenantID, projectID uuid.UUID) (*models.Sprint, error)

	// 迭代报告
	GetSprintReport(ctx context.Context, tenantID, sprintID uuid.UUID) (*SprintReport, error)
	GetBurndownChart(ctx context.Context, tenantID, sprintID uuid.UUID) (*BurndownChart, error)
	GetVelocityChart(ctx context.Context, tenantID, projectID uuid.UUID, sprintCount int) (*VelocityChart, error)
}

// sprintService 迭代服务实现
type sprintService struct {
	db     *gorm.DB
	logger logger.Logger
}

// NewSprintService 创建迭代服务实例
func NewSprintService(db *gorm.DB, logger logger.Logger) SprintService {
	return &sprintService{
		db:     db,
		logger: logger,
	}
}

// CreateSprintRequest 创建迭代请求
type CreateSprintRequest struct {
	ProjectID uuid.UUID `json:"project_id" validate:"required"`
	Name      string    `json:"name" validate:"required,min=1,max=255"`
	Goal      *string   `json:"goal"`
	StartDate time.Time `json:"start_date" validate:"required"`
	EndDate   time.Time `json:"end_date" validate:"required"`
}

// UpdateSprintRequest 更新迭代请求
type UpdateSprintRequest struct {
	TenantID  uuid.UUID  `json:"tenant_id" validate:"required"`
	SprintID  uuid.UUID  `json:"sprint_id" validate:"required"`
	Name      *string    `json:"name" validate:"omitempty,min=1,max=255"`
	Goal      *string    `json:"goal"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
}

// ListSprintsRequest 迭代列表请求
type ListSprintsRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	ProjectID uuid.UUID `json:"project_id" validate:"required"`
	Status    *string   `json:"status" validate:"omitempty,oneof=planned active completed"`
	Page      int       `json:"page" validate:"min=1"`
	Limit     int       `json:"limit" validate:"min=1,max=100"`
	SortBy    string    `json:"sort_by" validate:"oneof=created_at start_date end_date"`
	SortDesc  bool      `json:"sort_desc"`
}

// ListSprintsResponse 迭代列表响应
type ListSprintsResponse struct {
	Sprints []models.Sprint `json:"sprints"`
	Total   int64           `json:"total"`
	Page    int             `json:"page"`
	Limit   int             `json:"limit"`
}

// StartSprintRequest 开始迭代请求
type StartSprintRequest struct {
	TenantID uuid.UUID `json:"tenant_id" validate:"required"`
	SprintID uuid.UUID `json:"sprint_id" validate:"required"`
}

// CompleteSprintRequest 完成迭代请求
type CompleteSprintRequest struct {
	TenantID              uuid.UUID  `json:"tenant_id" validate:"required"`
	SprintID              uuid.UUID  `json:"sprint_id" validate:"required"`
	MoveUnfinishedTo      *uuid.UUID `json:"move_unfinished_to"` // 移动未完成任务到指定迭代
	CreateNextSprint      bool       `json:"create_next_sprint"`
	NextSprintName        *string    `json:"next_sprint_name"`
	NextSprintStartDate   *time.Time `json:"next_sprint_start_date"`
	NextSprintEndDate     *time.Time `json:"next_sprint_end_date"`
}

// SprintCompletionResult 迭代完成结果
type SprintCompletionResult struct {
	CompletedSprint     *models.Sprint `json:"completed_sprint"`
	NextSprint          *models.Sprint `json:"next_sprint,omitempty"`
	MovedTasksCount     int            `json:"moved_tasks_count"`
	CompletedTasksCount int            `json:"completed_tasks_count"`
	TotalTasksCount     int            `json:"total_tasks_count"`
}

// SprintReport 迭代报告
type SprintReport struct {
	Sprint              *models.Sprint    `json:"sprint"`
	TasksSummary        TasksSummary      `json:"tasks_summary"`
	StoryPointsSummary  StoryPointsSummary `json:"story_points_summary"`
	CompletionRate      float64           `json:"completion_rate"`
	VelocityPoints      int               `json:"velocity_points"`
	TaskBreakdown       []TaskBreakdown   `json:"task_breakdown"`
	DailyProgress       []DailyProgress   `json:"daily_progress"`
}

// TasksSummary 任务摘要
type TasksSummary struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Remaining int `json:"remaining"`
}

// StoryPointsSummary 故事点摘要
type StoryPointsSummary struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Remaining int `json:"remaining"`
}

// TaskBreakdown 任务分解
type TaskBreakdown struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
	Points int    `json:"points"`
}

// DailyProgress 每日进度
type DailyProgress struct {
	Date           time.Time `json:"date"`
	CompletedTasks int       `json:"completed_tasks"`
	CompletedPoints int       `json:"completed_points"`
	RemainingPoints int       `json:"remaining_points"`
}

// BurndownChart 燃尽图数据
type BurndownChart struct {
	IdealLine   []BurndownPoint `json:"ideal_line"`
	ActualLine  []BurndownPoint `json:"actual_line"`
	SprintDays  int             `json:"sprint_days"`
	TotalPoints int             `json:"total_points"`
}

// BurndownPoint 燃尽图数据点
type BurndownPoint struct {
	Date   time.Time `json:"date"`
	Points int       `json:"points"`
}

// VelocityChart 速率图数据
type VelocityChart struct {
	Sprints       []VelocitySprint `json:"sprints"`
	AverageVelocity float64        `json:"average_velocity"`
}

// VelocitySprint 速率迭代数据
type VelocitySprint struct {
	Name           string `json:"name"`
	PlannedPoints  int    `json:"planned_points"`
	CompletedPoints int    `json:"completed_points"`
}

// CreateSprint 创建迭代
func (s *sprintService) CreateSprint(ctx context.Context, req *CreateSprintRequest) (*models.Sprint, error) {
	s.logger.Info("创建迭代", "project_id", req.ProjectID, "name", req.Name)

	// 验证项目是否存在
	var project models.Project
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.ProjectID).First(&project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("项目不存在")
		}
		return nil, fmt.Errorf("查询项目失败: %w", err)
	}

	// 验证时间范围
	if req.EndDate.Before(req.StartDate) {
		return nil, fmt.Errorf("结束时间不能早于开始时间")
	}

	// 检查是否与现有迭代时间重叠
	var overlappingSprint models.Sprint
	result := s.db.WithContext(ctx).Where(`project_id = ? AND status != 'completed' AND 
		((start_date <= ? AND end_date >= ?) OR (start_date <= ? AND end_date >= ?) OR 
		(start_date >= ? AND end_date <= ?))`, 
		req.ProjectID, req.StartDate, req.StartDate, req.EndDate, req.EndDate, req.StartDate, req.EndDate).
		First(&overlappingSprint)

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("检查迭代时间冲突失败: %w", result.Error)
	}

	if result.Error == nil {
		return nil, fmt.Errorf("迭代时间与现有迭代 '%s' 冲突", overlappingSprint.Name)
	}

	// 创建迭代
	sprint := &models.Sprint{
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Goal:      req.Goal,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Status:    "planned",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(sprint).Error; err != nil {
		s.logger.Error("创建迭代失败", "error", err)
		return nil, fmt.Errorf("创建迭代失败: %w", err)
	}

	s.logger.Info("迭代创建成功", "sprint_id", sprint.ID, "name", sprint.Name)
	return sprint, nil
}

// GetSprint 获取迭代详情
func (s *sprintService) GetSprint(ctx context.Context, tenantID, sprintID uuid.UUID) (*models.Sprint, error) {
	var sprint models.Sprint
	
	err := s.db.WithContext(ctx).
		Preload("Project").
		Preload("Tasks").
		Preload("Tasks.Assignee").
		Preload("Tasks.Status").
		Joins("JOIN projects p ON sprints.project_id = p.id").
		Where("sprints.id = ? AND p.tenant_id = ? AND p.deleted_at IS NULL", sprintID, tenantID).
		First(&sprint).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("迭代不存在")
		}
		s.logger.Error("获取迭代失败", "error", err, "sprint_id", sprintID)
		return nil, fmt.Errorf("获取迭代失败: %w", err)
	}

	return &sprint, nil
}

// UpdateSprint 更新迭代
func (s *sprintService) UpdateSprint(ctx context.Context, req *UpdateSprintRequest) (*models.Sprint, error) {
	s.logger.Info("更新迭代", "sprint_id", req.SprintID, "tenant_id", req.TenantID)

	var sprint models.Sprint
	if err := s.db.WithContext(ctx).
		Joins("JOIN projects p ON sprints.project_id = p.id").
		Where("sprints.id = ? AND p.tenant_id = ? AND p.deleted_at IS NULL", req.SprintID, req.TenantID).
		First(&sprint).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("迭代不存在")
		}
		return nil, fmt.Errorf("查询迭代失败: %w", err)
	}

	// 如果迭代已激活，限制可修改的字段
	if sprint.Status == "active" {
		if req.StartDate != nil || req.EndDate != nil {
			return nil, fmt.Errorf("活跃迭代不能修改时间范围")
		}
	}

	// 更新字段
	updates := make(map[string]interface{})
	updates["updated_at"] = time.Now()

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Goal != nil {
		updates["goal"] = *req.Goal
	}
	if req.StartDate != nil {
		updates["start_date"] = *req.StartDate
	}
	if req.EndDate != nil {
		updates["end_date"] = *req.EndDate
	}

	// 验证更新后的时间范围
	if req.StartDate != nil && req.EndDate != nil {
		if req.EndDate.Before(*req.StartDate) {
			return nil, fmt.Errorf("结束时间不能早于开始时间")
		}
	}

	if err := s.db.WithContext(ctx).Model(&sprint).Updates(updates).Error; err != nil {
		s.logger.Error("更新迭代失败", "error", err)
		return nil, fmt.Errorf("更新迭代失败: %w", err)
	}

	// 重新获取更新后的迭代
	return s.GetSprint(ctx, req.TenantID, req.SprintID)
}

// DeleteSprint 删除迭代
func (s *sprintService) DeleteSprint(ctx context.Context, tenantID, sprintID uuid.UUID) error {
	s.logger.Info("删除迭代", "sprint_id", sprintID, "tenant_id", tenantID)

	// 检查迭代状态
	var sprint models.Sprint
	if err := s.db.WithContext(ctx).
		Joins("JOIN projects p ON sprints.project_id = p.id").
		Where("sprints.id = ? AND p.tenant_id = ? AND p.deleted_at IS NULL", sprintID, tenantID).
		First(&sprint).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("迭代不存在")
		}
		return fmt.Errorf("查询迭代失败: %w", err)
	}

	if sprint.Status == "active" {
		return fmt.Errorf("不能删除活跃迭代")
	}

	// 检查是否有关联任务
	var taskCount int64
	s.db.WithContext(ctx).Model(&models.Task{}).Where("sprint_id = ?", sprintID).Count(&taskCount)
	if taskCount > 0 {
		// 将任务移出迭代
		s.db.WithContext(ctx).Model(&models.Task{}).Where("sprint_id = ?", sprintID).Update("sprint_id", nil)
	}

	// 删除迭代
	if err := s.db.WithContext(ctx).Delete(&sprint).Error; err != nil {
		s.logger.Error("删除迭代失败", "error", err)
		return fmt.Errorf("删除迭代失败: %w", err)
	}

	s.logger.Info("迭代删除成功", "sprint_id", sprintID)
	return nil
}

// ListSprints 获取迭代列表
func (s *sprintService) ListSprints(ctx context.Context, req *ListSprintsRequest) (*ListSprintsResponse, error) {
	query := s.db.WithContext(ctx).Model(&models.Sprint{}).
		Joins("JOIN projects p ON sprints.project_id = p.id").
		Where("sprints.project_id = ? AND p.tenant_id = ? AND p.deleted_at IS NULL", req.ProjectID, req.TenantID)

	// 过滤条件
	if req.Status != nil {
		query = query.Where("sprints.status = ?", *req.Status)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("计算迭代总数失败: %w", err)
	}

	// 排序
	sortBy := "created_at"
	if req.SortBy != "" {
		sortBy = "sprints." + req.SortBy
	}
	if req.SortDesc {
		sortBy += " DESC"
	}
	query = query.Order(sortBy)

	// 分页
	offset := (req.Page - 1) * req.Limit
	query = query.Offset(offset).Limit(req.Limit)

	var sprints []models.Sprint
	if err := query.Find(&sprints).Error; err != nil {
		return nil, fmt.Errorf("查询迭代列表失败: %w", err)
	}

	// 加载任务计数
	for i := range sprints {
		var taskCount int64
		s.db.WithContext(ctx).Model(&models.Task{}).Where("sprint_id = ?", sprints[i].ID).Count(&taskCount)
		// 这里可以添加任务计数到Sprint结构体中，或者通过其他方式返回
	}

	return &ListSprintsResponse{
		Sprints: sprints,
		Total:   total,
		Page:    req.Page,
		Limit:   req.Limit,
	}, nil
}

// StartSprint 开始迭代
func (s *sprintService) StartSprint(ctx context.Context, req *StartSprintRequest) (*models.Sprint, error) {
	s.logger.Info("开始迭代", "sprint_id", req.SprintID, "tenant_id", req.TenantID)

	var sprint models.Sprint
	if err := s.db.WithContext(ctx).
		Joins("JOIN projects p ON sprints.project_id = p.id").
		Where("sprints.id = ? AND p.tenant_id = ? AND p.deleted_at IS NULL", req.SprintID, req.TenantID).
		First(&sprint).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("迭代不存在")
		}
		return nil, fmt.Errorf("查询迭代失败: %w", err)
	}

	if sprint.Status != "planned" {
		return nil, fmt.Errorf("只有计划状态的迭代可以开始")
	}

	// 检查同项目下是否有其他活跃迭代
	var activeSprint models.Sprint
	result := s.db.WithContext(ctx).Where("project_id = ? AND status = 'active' AND id != ?", 
		sprint.ProjectID, req.SprintID).First(&activeSprint)
	
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("检查活跃迭代失败: %w", result.Error)
	}

	if result.Error == nil {
		return nil, fmt.Errorf("项目已有活跃迭代 '%s'", activeSprint.Name)
	}

	// 更新迭代状态
	updates := map[string]interface{}{
		"status":     "active",
		"updated_at": time.Now(),
	}

	if err := s.db.WithContext(ctx).Model(&sprint).Updates(updates).Error; err != nil {
		s.logger.Error("开始迭代失败", "error", err)
		return nil, fmt.Errorf("开始迭代失败: %w", err)
	}

	s.logger.Info("迭代开始成功", "sprint_id", req.SprintID)
	return s.GetSprint(ctx, req.TenantID, req.SprintID)
}

// CompleteSprint 完成迭代
func (s *sprintService) CompleteSprint(ctx context.Context, req *CompleteSprintRequest) (*SprintCompletionResult, error) {
	s.logger.Info("完成迭代", "sprint_id", req.SprintID, "tenant_id", req.TenantID)

	// 在事务中执行
	var result SprintCompletionResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取迭代信息
		var sprint models.Sprint
		if err := tx.Joins("JOIN projects p ON sprints.project_id = p.id").
			Where("sprints.id = ? AND p.tenant_id = ? AND p.deleted_at IS NULL", req.SprintID, req.TenantID).
			First(&sprint).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("迭代不存在")
			}
			return fmt.Errorf("查询迭代失败: %w", err)
		}

		if sprint.Status != "active" {
			return fmt.Errorf("只有活跃迭代可以完成")
		}

		// 统计任务信息
		var totalTasks, completedTasks int64
		tx.Model(&models.Task{}).Where("sprint_id = ?", req.SprintID).Count(&totalTasks)
		tx.Model(&models.Task{}).
			Joins("LEFT JOIN task_statuses ts ON tasks.status_id = ts.id").
			Where("tasks.sprint_id = ? AND ts.category = 'done'", req.SprintID).
			Count(&completedTasks)

		result.TotalTasksCount = int(totalTasks)
		result.CompletedTasksCount = int(completedTasks)

		// 处理未完成的任务
		if req.MoveUnfinishedTo != nil {
			unfinishedResult := tx.Model(&models.Task{}).
				Joins("LEFT JOIN task_statuses ts ON tasks.status_id = ts.id").
				Where("tasks.sprint_id = ? AND (ts.category != 'done' OR ts.category IS NULL)", req.SprintID).
				Update("sprint_id", *req.MoveUnfinishedTo)
			
			if unfinishedResult.Error != nil {
				return fmt.Errorf("移动未完成任务失败: %w", unfinishedResult.Error)
			}
			result.MovedTasksCount = int(unfinishedResult.RowsAffected)
		}

		// 完成迭代
		if err := tx.Model(&sprint).Updates(map[string]interface{}{
			"status":     "completed",
			"updated_at": time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("完成迭代失败: %w", err)
		}

		result.CompletedSprint = &sprint

		// 创建下一个迭代
		if req.CreateNextSprint && req.NextSprintName != nil {
			nextSprint := &models.Sprint{
				ProjectID: sprint.ProjectID,
				Name:      *req.NextSprintName,
				Status:    "planned",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if req.NextSprintStartDate != nil {
				nextSprint.StartDate = *req.NextSprintStartDate
			} else {
				nextSprint.StartDate = sprint.EndDate.AddDate(0, 0, 1) // 下一天开始
			}

			if req.NextSprintEndDate != nil {
				nextSprint.EndDate = *req.NextSprintEndDate
			} else {
				duration := sprint.EndDate.Sub(sprint.StartDate)
				nextSprint.EndDate = nextSprint.StartDate.Add(duration)
			}

			if err := tx.Create(nextSprint).Error; err != nil {
				return fmt.Errorf("创建下一个迭代失败: %w", err)
			}

			result.NextSprint = nextSprint
		}

		return nil
	})

	if err != nil {
		s.logger.Error("完成迭代失败", "error", err)
		return nil, err
	}

	s.logger.Info("迭代完成成功", "sprint_id", req.SprintID)
	return &result, nil
}

// GetActiveSprint 获取项目的活跃迭代
func (s *sprintService) GetActiveSprint(ctx context.Context, tenantID, projectID uuid.UUID) (*models.Sprint, error) {
	var sprint models.Sprint
	
	err := s.db.WithContext(ctx).
		Preload("Tasks").
		Preload("Tasks.Assignee").
		Preload("Tasks.Status").
		Joins("JOIN projects p ON sprints.project_id = p.id").
		Where("sprints.project_id = ? AND sprints.status = 'active' AND p.tenant_id = ? AND p.deleted_at IS NULL", 
			projectID, tenantID).
		First(&sprint).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有活跃迭代是正常的
		}
		return nil, fmt.Errorf("获取活跃迭代失败: %w", err)
	}

	return &sprint, nil
}

// GetSprintReport 获取迭代报告
func (s *sprintService) GetSprintReport(ctx context.Context, tenantID, sprintID uuid.UUID) (*SprintReport, error) {
	sprint, err := s.GetSprint(ctx, tenantID, sprintID)
	if err != nil {
		return nil, err
	}

	report := &SprintReport{
		Sprint: sprint,
	}

	// 统计任务摘要
	var totalTasks, completedTasks int64
	s.db.WithContext(ctx).Model(&models.Task{}).Where("sprint_id = ?", sprintID).Count(&totalTasks)
	s.db.WithContext(ctx).Model(&models.Task{}).
		Joins("LEFT JOIN task_statuses ts ON tasks.status_id = ts.id").
		Where("tasks.sprint_id = ? AND ts.category = 'done'", sprintID).
		Count(&completedTasks)

	report.TasksSummary = TasksSummary{
		Total:     int(totalTasks),
		Completed: int(completedTasks),
		Remaining: int(totalTasks - completedTasks),
	}

	// 统计故事点
	var totalPoints, completedPoints int
	s.db.WithContext(ctx).Model(&models.Task{}).
		Where("sprint_id = ?", sprintID).
		Select("COALESCE(SUM(story_points), 0)").
		Scan(&totalPoints)

	s.db.WithContext(ctx).Model(&models.Task{}).
		Joins("LEFT JOIN task_statuses ts ON tasks.status_id = ts.id").
		Where("tasks.sprint_id = ? AND ts.category = 'done'", sprintID).
		Select("COALESCE(SUM(story_points), 0)").
		Scan(&completedPoints)

	report.StoryPointsSummary = StoryPointsSummary{
		Total:     totalPoints,
		Completed: completedPoints,
		Remaining: totalPoints - completedPoints,
	}

	// 计算完成率
	if totalPoints > 0 {
		report.CompletionRate = float64(completedPoints) / float64(totalPoints)
	}
	report.VelocityPoints = completedPoints

	// 任务状态分解
	var breakdowns []TaskBreakdown
	s.db.WithContext(ctx).Model(&models.Task{}).
		Select("ts.name as status, COUNT(*) as count, COALESCE(SUM(tasks.story_points), 0) as points").
		Joins("LEFT JOIN task_statuses ts ON tasks.status_id = ts.id").
		Where("tasks.sprint_id = ?", sprintID).
		Group("ts.name").
		Scan(&breakdowns)

	report.TaskBreakdown = breakdowns

	return report, nil
}

// GetBurndownChart 获取燃尽图数据
func (s *sprintService) GetBurndownChart(ctx context.Context, tenantID, sprintID uuid.UUID) (*BurndownChart, error) {
	sprint, err := s.GetSprint(ctx, tenantID, sprintID)
	if err != nil {
		return nil, err
	}

	// 获取总故事点数
	var totalPoints int
	s.db.WithContext(ctx).Model(&models.Task{}).
		Where("sprint_id = ?", sprintID).
		Select("COALESCE(SUM(story_points), 0)").
		Scan(&totalPoints)

	// 计算迭代天数
	sprintDays := int(sprint.EndDate.Sub(sprint.StartDate).Hours()/24) + 1

	chart := &BurndownChart{
		SprintDays:  sprintDays,
		TotalPoints: totalPoints,
		IdealLine:   make([]BurndownPoint, sprintDays),
		ActualLine:  make([]BurndownPoint, 0),
	}

	// 计算理想燃尽线
	pointsPerDay := float64(totalPoints) / float64(sprintDays-1)
	for i := 0; i < sprintDays; i++ {
		date := sprint.StartDate.AddDate(0, 0, i)
		remainingPoints := totalPoints - int(float64(i)*pointsPerDay)
		if remainingPoints < 0 {
			remainingPoints = 0
		}
		chart.IdealLine[i] = BurndownPoint{
			Date:   date,
			Points: remainingPoints,
		}
	}

	// 计算实际燃尽线（这里简化实现，实际应该基于任务完成的历史数据）
	// 在实际项目中，你需要记录每天的任务完成情况
	currentDate := time.Now()
	if currentDate.After(sprint.EndDate) {
		currentDate = sprint.EndDate
	}

	for d := sprint.StartDate; d.Before(currentDate.AddDate(0, 0, 1)); d = d.AddDate(0, 0, 1) {
		// 这里应该查询截至当天完成的任务点数
		var completedPoints int
		s.db.WithContext(ctx).Model(&models.Task{}).
			Joins("LEFT JOIN task_statuses ts ON tasks.status_id = ts.id").
			Where("tasks.sprint_id = ? AND ts.category = 'done' AND tasks.updated_at <= ?", 
				sprintID, d.Add(24*time.Hour-time.Second)).
			Select("COALESCE(SUM(story_points), 0)").
			Scan(&completedPoints)

		remainingPoints := totalPoints - completedPoints
		chart.ActualLine = append(chart.ActualLine, BurndownPoint{
			Date:   d,
			Points: remainingPoints,
		})
	}

	return chart, nil
}

// GetVelocityChart 获取速率图数据
func (s *sprintService) GetVelocityChart(ctx context.Context, tenantID, projectID uuid.UUID, sprintCount int) (*VelocityChart, error) {
	var sprints []models.Sprint
	
	err := s.db.WithContext(ctx).
		Joins("JOIN projects p ON sprints.project_id = p.id").
		Where("sprints.project_id = ? AND p.tenant_id = ? AND sprints.status = 'completed' AND p.deleted_at IS NULL", 
			projectID, tenantID).
		Order("sprints.end_date DESC").
		Limit(sprintCount).
		Find(&sprints).Error

	if err != nil {
		return nil, fmt.Errorf("获取已完成迭代失败: %w", err)
	}

	chart := &VelocityChart{
		Sprints: make([]VelocitySprint, len(sprints)),
	}

	var totalVelocity int
	for i, sprint := range sprints {
		var plannedPoints, completedPoints int
		
		// 计划点数（迭代开始时的所有任务点数）
		s.db.WithContext(ctx).Model(&models.Task{}).
			Where("sprint_id = ?", sprint.ID).
			Select("COALESCE(SUM(story_points), 0)").
			Scan(&plannedPoints)

		// 完成点数
		s.db.WithContext(ctx).Model(&models.Task{}).
			Joins("LEFT JOIN task_statuses ts ON tasks.status_id = ts.id").
			Where("tasks.sprint_id = ? AND ts.category = 'done'", sprint.ID).
			Select("COALESCE(SUM(story_points), 0)").
			Scan(&completedPoints)

		chart.Sprints[i] = VelocitySprint{
			Name:            sprint.Name,
			PlannedPoints:   plannedPoints,
			CompletedPoints: completedPoints,
		}

		totalVelocity += completedPoints
	}

	if len(sprints) > 0 {
		chart.AverageVelocity = float64(totalVelocity) / float64(len(sprints))
	}

	return chart, nil
}