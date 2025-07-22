package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cicd-service/internal/config"
	"cicd-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PipelineRunService 流水线运行服务接口
type PipelineRunService interface {
	Create(req *CreatePipelineRunRequest) (*models.PipelineRun, error)
	GetByID(id uuid.UUID) (*models.PipelineRun, error)
	GetByPipeline(pipelineID uuid.UUID, limit int) ([]models.PipelineRun, error)
	Cancel(id uuid.UUID, reason string) error
	Retry(id uuid.UUID) (*models.PipelineRun, error)
	UpdateStatus(id uuid.UUID, status string, message *string) error
	List(req *ListPipelineRunsRequest) ([]models.PipelineRun, int64, error)
	GetStatistics(req *PipelineRunStatsRequest) (*PipelineRunStats, error)
	CleanupExpiredRuns() error
}

type pipelineRunService struct {
	db            *gorm.DB
	config        *config.Config
	tektonService TektonService
}

// NewPipelineRunService 创建流水线运行服务实例
func NewPipelineRunService(db *gorm.DB, cfg *config.Config, tektonSvc TektonService) PipelineRunService {
	return &pipelineRunService{
		db:            db,
		config:        cfg,
		tektonService: tektonSvc,
	}
}

// CreatePipelineRunRequest 创建流水线运行请求
type CreatePipelineRunRequest struct {
	PipelineID    uuid.UUID              `json:"pipeline_id" validate:"required"`
	TriggerType   string                 `json:"trigger_type" validate:"required,oneof=manual webhook schedule api"`
	TriggerBy     *uuid.UUID             `json:"trigger_by"`
	TriggerData   map[string]interface{} `json:"trigger_data"`
	Parameters    map[string]interface{} `json:"parameters"`
	Environment   *string                `json:"environment"`
	ScheduledAt   *time.Time            `json:"scheduled_at"`
}

// ListPipelineRunsRequest 列表查询请求
type ListPipelineRunsRequest struct {
	PipelineID   *uuid.UUID `json:"pipeline_id"`
	ProjectID    *uuid.UUID `json:"project_id"`
	Status       *string    `json:"status"`
	TriggerType  *string    `json:"trigger_type"`
	TriggerBy    *uuid.UUID `json:"trigger_by"`
	StartTime    *time.Time `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	Page         int        `json:"page"`
	Limit        int        `json:"limit"`
	SortBy       string     `json:"sort_by"`
	SortDesc     bool       `json:"sort_desc"`
}

// PipelineRunStatsRequest 流水线运行统计请求
type PipelineRunStatsRequest struct {
	PipelineID *uuid.UUID `json:"pipeline_id"`
	ProjectID  *uuid.UUID `json:"project_id"`
	StartTime  *time.Time `json:"start_time"`
	EndTime    *time.Time `json:"end_time"`
	GroupBy    string     `json:"group_by"` // day, hour, pipeline, status
}

// PipelineRunStats 流水线运行统计结果
type PipelineRunStats struct {
	TotalRuns         int64                    `json:"total_runs"`
	SuccessfulRuns    int64                    `json:"successful_runs"`
	FailedRuns        int64                    `json:"failed_runs"`
	CancelledRuns     int64                    `json:"cancelled_runs"`
	PendingRuns       int64                    `json:"pending_runs"`
	RunningRuns       int64                    `json:"running_runs"`
	SuccessRate       float64                  `json:"success_rate"`
	AverageDuration   float64                  `json:"average_duration"`
	MedianDuration    float64                  `json:"median_duration"`
	GroupedStats      []GroupedRunStat         `json:"grouped_stats"`
	DurationTrend     []DurationTrendPoint     `json:"duration_trend"`
	SuccessRateTrend  []SuccessRateTrendPoint  `json:"success_rate_trend"`
}

// GroupedRunStat 分组运行统计
type GroupedRunStat struct {
	Group         string    `json:"group"`
	Count         int64     `json:"count"`
	SuccessCount  int64     `json:"success_count"`
	FailedCount   int64     `json:"failed_count"`
	SuccessRate   float64   `json:"success_rate"`
	AvgDuration   float64   `json:"avg_duration"`
	Timestamp     *time.Time `json:"timestamp,omitempty"`
}

// DurationTrendPoint 持续时间趋势点
type DurationTrendPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	AvgDuration float64   `json:"avg_duration"`
	Count       int64     `json:"count"`
}

// SuccessRateTrendPoint 成功率趋势点
type SuccessRateTrendPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	SuccessRate float64   `json:"success_rate"`
	TotalCount  int64     `json:"total_count"`
}

// Create 创建流水线运行
func (s *pipelineRunService) Create(req *CreatePipelineRunRequest) (*models.PipelineRun, error) {
	// 获取流水线信息
	var pipeline models.Pipeline
	if err := s.db.Where("id = ? AND deleted_at IS NULL", req.PipelineID).
		Preload("Tasks").First(&pipeline).Error; err != nil {
		return nil, fmt.Errorf("流水线不存在或已删除")
	}

	// 检查流水线状态
	if pipeline.Status != "active" {
		return nil, fmt.Errorf("流水线已禁用，无法运行")
	}

	// 检查并发运行限制
	var runningCount int64
	if err := s.db.Model(&models.PipelineRun{}).
		Where("pipeline_id = ? AND status IN (?)", req.PipelineID, []string{"pending", "running"}).
		Count(&runningCount).Error; err != nil {
		return nil, fmt.Errorf("检查并发运行失败: %w", err)
	}

	if runningCount >= int64(s.config.Tekton.MaxConcurrentRuns) {
		return nil, fmt.Errorf("已达到最大并发运行限制(%d)", s.config.Tekton.MaxConcurrentRuns)
	}

	// 创建流水线运行记录
	pipelineRun := &models.PipelineRun{
		PipelineID:  req.PipelineID,
		Status:      "pending",
		TriggerType: req.TriggerType,
		TriggerBy:   req.TriggerBy,
	}

	// 处理触发数据
	if req.TriggerData != nil {
		triggerDataJSON, err := json.Marshal(req.TriggerData)
		if err != nil {
			return nil, fmt.Errorf("序列化触发数据失败: %w", err)
		}
		pipelineRun.TriggerData = triggerDataJSON
	}

	// 保存到数据库
	if err := s.db.Create(pipelineRun).Error; err != nil {
		return nil, fmt.Errorf("创建流水线运行记录失败: %w", err)
	}

	// 异步启动流水线
	go s.startPipelineRunAsync(pipelineRun, &pipeline, req.Parameters)

	return pipelineRun, nil
}

// startPipelineRunAsync 异步启动流水线
func (s *pipelineRunService) startPipelineRunAsync(run *models.PipelineRun, pipeline *models.Pipeline, params map[string]interface{}) {
	ctx := context.Background()

	// 更新状态为运行中
	s.UpdateStatus(run.ID, "running", nil)
	startTime := time.Now()
	s.db.Model(run).Update("started_at", startTime)

	// 创建Tekton PipelineRun
	tektonRun := &TektonPipelineRunRequest{
		Name:        fmt.Sprintf("run-%s-%d", run.ID.String()[:8], run.RunNumber),
		PipelineID:  pipeline.ID,
		RunID:       run.ID,
		Parameters:  params,
		Timeout:     pipeline.Config.Timeout,
		Workspace:   pipeline.Config.Workspace,
		ServiceAccount: pipeline.Config.ServiceAccount,
	}

	// 提交到Tekton
	if err := s.tektonService.CreatePipelineRun(ctx, tektonRun); err != nil {
		s.UpdateStatus(run.ID, "failed", &err.Error())
		return
	}

	// 创建任务运行记录
	s.createTaskRuns(run, pipeline.Tasks)
}

// createTaskRuns 创建任务运行记录
func (s *pipelineRunService) createTaskRuns(pipelineRun *models.PipelineRun, tasks []models.Task) {
	for _, task := range tasks {
		taskRun := &models.TaskRun{
			PipelineRunID: pipelineRun.ID,
			TaskID:        task.ID,
			Name:          task.Name,
			Status:        "pending",
		}

		s.db.Create(taskRun)
	}
}

// GetByID 根据ID获取流水线运行
func (s *pipelineRunService) GetByID(id uuid.UUID) (*models.PipelineRun, error) {
	var pipelineRun models.PipelineRun
	if err := s.db.Where("id = ?", id).
		Preload("Pipeline").
		Preload("TaskRuns", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("TaskRuns.Task").
		First(&pipelineRun).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("流水线运行不存在")
		}
		return nil, fmt.Errorf("获取流水线运行失败: %w", err)
	}
	return &pipelineRun, nil
}

// GetByPipeline 获取流水线的运行历史
func (s *pipelineRunService) GetByPipeline(pipelineID uuid.UUID, limit int) ([]models.PipelineRun, error) {
	var runs []models.PipelineRun
	query := s.db.Where("pipeline_id = ?", pipelineID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&runs).Error; err != nil {
		return nil, fmt.Errorf("获取流水线运行历史失败: %w", err)
	}
	return runs, nil
}

// Cancel 取消流水线运行
func (s *pipelineRunService) Cancel(id uuid.UUID, reason string) error {
	run, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// 检查状态
	if run.Status != "pending" && run.Status != "running" {
		return fmt.Errorf("流水线运行状态为 %s，无法取消", run.Status)
	}

	// 取消Tekton PipelineRun
	ctx := context.Background()
	if err := s.tektonService.CancelPipelineRun(ctx, run.ID); err != nil {
		return fmt.Errorf("取消Tekton流水线运行失败: %w", err)
	}

	// 更新状态
	return s.UpdateStatus(id, "cancelled", &reason)
}

// Retry 重试流水线运行
func (s *pipelineRunService) Retry(id uuid.UUID) (*models.PipelineRun, error) {
	originalRun, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 检查状态
	if originalRun.Status != "failed" && originalRun.Status != "cancelled" {
		return nil, fmt.Errorf("只能重试失败或已取消的流水线运行")
	}

	// 解析触发数据
	var triggerData map[string]interface{}
	if originalRun.TriggerData != nil {
		if err := json.Unmarshal(originalRun.TriggerData, &triggerData); err != nil {
			return nil, fmt.Errorf("解析触发数据失败: %w", err)
		}
	}

	// 创建重试请求
	retryReq := &CreatePipelineRunRequest{
		PipelineID:  originalRun.PipelineID,
		TriggerType: "manual", // 重试总是手动触发
		TriggerBy:   originalRun.TriggerBy,
		TriggerData: triggerData,
	}

	return s.Create(retryReq)
}

// UpdateStatus 更新流水线运行状态
func (s *pipelineRunService) UpdateStatus(id uuid.UUID, status string, message *string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	// 根据状态更新时间戳
	switch status {
	case "running":
		updates["started_at"] = time.Now()
	case "succeeded", "failed", "cancelled", "timeout":
		now := time.Now()
		updates["finished_at"] = now
		
		// 计算持续时间
		var run models.PipelineRun
		if err := s.db.Where("id = ?", id).First(&run).Error; err == nil && run.StartedAt != nil {
			duration := int(now.Sub(*run.StartedAt).Seconds())
			updates["duration"] = duration
		}
	}

	if message != nil {
		updates["error_message"] = *message
	}

	if err := s.db.Model(&models.PipelineRun{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新流水线运行状态失败: %w", err)
	}

	// 更新流水线的最后运行信息
	if status == "running" {
		s.db.Model(&models.Pipeline{}).
			Where("id = (SELECT pipeline_id FROM pipeline_runs WHERE id = ?)", id).
			Updates(map[string]interface{}{
				"last_run_id": id,
				"last_run_at": time.Now(),
			})
	}

	return nil
}

// List 列表查询流水线运行
func (s *pipelineRunService) List(req *ListPipelineRunsRequest) ([]models.PipelineRun, int64, error) {
	query := s.db.Model(&models.PipelineRun{})

	// 应用筛选条件
	if req.PipelineID != nil {
		query = query.Where("pipeline_id = ?", *req.PipelineID)
	}

	if req.ProjectID != nil {
		query = query.Joins("JOIN pipelines ON pipeline_runs.pipeline_id = pipelines.id").
			Where("pipelines.project_id = ? AND pipelines.deleted_at IS NULL", *req.ProjectID)
	}

	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	if req.TriggerType != nil {
		query = query.Where("trigger_type = ?", *req.TriggerType)
	}

	if req.TriggerBy != nil {
		query = query.Where("trigger_by = ?", *req.TriggerBy)
	}

	if req.StartTime != nil {
		query = query.Where("created_at >= ?", *req.StartTime)
	}

	if req.EndTime != nil {
		query = query.Where("created_at <= ?", *req.EndTime)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计流水线运行总数失败: %w", err)
	}

	// 应用排序
	sortBy := "created_at"
	if req.SortBy != "" {
		sortBy = req.SortBy
	}

	if req.SortDesc {
		sortBy += " DESC"
	} else {
		sortBy += " ASC"
	}
	query = query.Order(sortBy)

	// 应用分页
	if req.Page > 0 && req.Limit > 0 {
		offset := (req.Page - 1) * req.Limit
		query = query.Offset(offset).Limit(req.Limit)
	}

	var runs []models.PipelineRun
	if err := query.Preload("Pipeline").Find(&runs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询流水线运行列表失败: %w", err)
	}

	return runs, total, nil
}

// GetStatistics 获取流水线运行统计信息
func (s *pipelineRunService) GetStatistics(req *PipelineRunStatsRequest) (*PipelineRunStats, error) {
	stats := &PipelineRunStats{}
	query := s.db.Model(&models.PipelineRun{})

	// 应用筛选条件
	if req.PipelineID != nil {
		query = query.Where("pipeline_id = ?", *req.PipelineID)
	}

	if req.ProjectID != nil {
		query = query.Joins("JOIN pipelines ON pipeline_runs.pipeline_id = pipelines.id").
			Where("pipelines.project_id = ? AND pipelines.deleted_at IS NULL", *req.ProjectID)
	}

	if req.StartTime != nil {
		query = query.Where("pipeline_runs.created_at >= ?", *req.StartTime)
	}

	if req.EndTime != nil {
		query = query.Where("pipeline_runs.created_at <= ?", *req.EndTime)
	}

	// 基础统计
	if err := query.Count(&stats.TotalRuns).Error; err != nil {
		return nil, fmt.Errorf("统计总运行次数失败: %w", err)
	}

	if err := query.Where("status = ?", "succeeded").Count(&stats.SuccessfulRuns).Error; err != nil {
		return nil, fmt.Errorf("统计成功运行次数失败: %w", err)
	}

	if err := query.Where("status = ?", "failed").Count(&stats.FailedRuns).Error; err != nil {
		return nil, fmt.Errorf("统计失败运行次数失败: %w", err)
	}

	if err := query.Where("status = ?", "cancelled").Count(&stats.CancelledRuns).Error; err != nil {
		return nil, fmt.Errorf("统计取消运行次数失败: %w", err)
	}

	if err := query.Where("status = ?", "pending").Count(&stats.PendingRuns).Error; err != nil {
		return nil, fmt.Errorf("统计待运行次数失败: %w", err)
	}

	if err := query.Where("status = ?", "running").Count(&stats.RunningRuns).Error; err != nil {
		return nil, fmt.Errorf("统计运行中次数失败: %w", err)
	}

	// 计算成功率
	if stats.TotalRuns > 0 {
		stats.SuccessRate = float64(stats.SuccessfulRuns) / float64(stats.TotalRuns) * 100
	}

	// 计算平均持续时间和中位数
	var durations []int
	query.Where("status = ? AND duration IS NOT NULL", "succeeded").
		Pluck("duration", &durations)

	if len(durations) > 0 {
		sum := 0
		for _, d := range durations {
			sum += d
		}
		stats.AverageDuration = float64(sum) / float64(len(durations))

		// 计算中位数
		if len(durations)%2 == 0 {
			stats.MedianDuration = float64(durations[len(durations)/2-1]+durations[len(durations)/2]) / 2
		} else {
			stats.MedianDuration = float64(durations[len(durations)/2])
		}
	}

	// 分组统计
	if req.GroupBy != "" {
		groupedStats, err := s.getGroupedStats(query, req.GroupBy)
		if err != nil {
			return nil, fmt.Errorf("获取分组统计失败: %w", err)
		}
		stats.GroupedStats = groupedStats
	}

	return stats, nil
}

// getGroupedStats 获取分组统计数据
func (s *pipelineRunService) getGroupedStats(query *gorm.DB, groupBy string) ([]GroupedRunStat, error) {
	var groupedStats []GroupedRunStat

	var selectFields, groupFields string
	switch groupBy {
	case "day":
		selectFields = "DATE(created_at) as group_key, COUNT(*) as count, " +
			"COUNT(CASE WHEN status = 'succeeded' THEN 1 END) as success_count, " +
			"COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_count, " +
			"AVG(CASE WHEN duration IS NOT NULL THEN duration END) as avg_duration, " +
			"DATE(created_at) as timestamp"
		groupFields = "DATE(created_at)"
	case "hour":
		selectFields = "DATE_TRUNC('hour', created_at) as group_key, COUNT(*) as count, " +
			"COUNT(CASE WHEN status = 'succeeded' THEN 1 END) as success_count, " +
			"COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_count, " +
			"AVG(CASE WHEN duration IS NOT NULL THEN duration END) as avg_duration, " +
			"DATE_TRUNC('hour', created_at) as timestamp"
		groupFields = "DATE_TRUNC('hour', created_at)"
	case "status":
		selectFields = "status as group_key, COUNT(*) as count"
		groupFields = "status"
	default:
		return groupedStats, nil
	}

	type GroupResult struct {
		GroupKey    string     `json:"group_key"`
		Count       int64      `json:"count"`
		SuccessCount int64     `json:"success_count"`
		FailedCount  int64     `json:"failed_count"`
		AvgDuration  float64   `json:"avg_duration"`
		Timestamp    *time.Time `json:"timestamp,omitempty"`
	}

	var results []GroupResult
	if err := query.Select(selectFields).
		Group(groupFields).
		Order(groupFields).
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("分组统计失败: %w", err)
	}

	for _, result := range results {
		var successRate float64
		if result.Count > 0 {
			successRate = float64(result.SuccessCount) / float64(result.Count) * 100
		}

		stat := GroupedRunStat{
			Group:        result.GroupKey,
			Count:        result.Count,
			SuccessCount: result.SuccessCount,
			FailedCount:  result.FailedCount,
			SuccessRate:  successRate,
			AvgDuration:  result.AvgDuration,
			Timestamp:    result.Timestamp,
		}
		groupedStats = append(groupedStats, stat)
	}

	return groupedStats, nil
}

// CleanupExpiredRuns 清理过期的流水线运行
func (s *pipelineRunService) CleanupExpiredRuns() error {
	// 清理已完成的PipelineRun（根据配置的TTL）
	pipelineRunCutoff := time.Now().Add(-time.Duration(s.config.Tekton.PipelineRunTTL) * time.Hour)
	
	if err := s.db.Where("status IN (?) AND finished_at < ?", 
		[]string{"succeeded", "failed", "cancelled", "timeout"}, pipelineRunCutoff).
		Delete(&models.PipelineRun{}).Error; err != nil {
		return fmt.Errorf("清理过期PipelineRun失败: %w", err)
	}

	// 清理TaskRun
	taskRunCutoff := time.Now().Add(-time.Duration(s.config.Tekton.TaskRunTTL) * time.Hour)
	
	if err := s.db.Where("status IN (?) AND finished_at < ?", 
		[]string{"succeeded", "failed", "cancelled", "skipped"}, taskRunCutoff).
		Delete(&models.TaskRun{}).Error; err != nil {
		return fmt.Errorf("清理过期TaskRun失败: %w", err)
	}

	return nil
}