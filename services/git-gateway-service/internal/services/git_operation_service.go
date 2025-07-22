package services

import (
	"fmt"
	"time"

	"git-gateway-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GitOperationService Git操作审计服务接口
type GitOperationService interface {
	RecordOperation(req *RecordOperationRequest) (*models.GitOperation, error)
	GetByID(id uuid.UUID) (*models.GitOperation, error)
	GetByRepository(repositoryID uuid.UUID) ([]models.GitOperation, error)
	GetByUser(userID uuid.UUID) ([]models.GitOperation, error)
	GetOperationStats(req *OperationStatsRequest) (*OperationStats, error)
	List(req *ListOperationsRequest) ([]models.GitOperation, int64, error)
	CleanupOldRecords(retentionDays int) error
}

type gitOperationService struct {
	db *gorm.DB
}

// NewGitOperationService 创建Git操作审计服务实例
func NewGitOperationService(db *gorm.DB) GitOperationService {
	return &gitOperationService{db: db}
}

// RecordOperationRequest 记录操作请求
type RecordOperationRequest struct {
	RepositoryID     uuid.UUID `json:"repository_id" validate:"required"`
	UserID           uuid.UUID `json:"user_id" validate:"required"`
	Operation        string    `json:"operation" validate:"required,max=50"`
	Protocol         string    `json:"protocol" validate:"required,oneof=http ssh"`
	RefName          *string   `json:"ref_name" validate:"omitempty,max=255"`
	CommitSHA        *string   `json:"commit_sha" validate:"omitempty,len=40"`
	ClientIP         string    `json:"client_ip" validate:"required,max=45"`
	UserAgent        *string   `json:"user_agent" validate:"omitempty,max=512"`
	Success          bool      `json:"success"`
	ErrorMsg         *string   `json:"error_msg"`
	Duration         int       `json:"duration"` // 毫秒
	BytesTransferred int64     `json:"bytes_transferred"`
}

// ListOperationsRequest 列表查询请求
type ListOperationsRequest struct {
	RepositoryID *uuid.UUID `json:"repository_id"`
	UserID       *uuid.UUID `json:"user_id"`
	Operation    *string    `json:"operation"`
	Protocol     *string    `json:"protocol"`
	Success      *bool      `json:"success"`
	StartTime    *time.Time `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	ClientIP     *string    `json:"client_ip"`
	Page         int        `json:"page"`
	Limit        int        `json:"limit"`
	SortBy       string     `json:"sort_by"`
	SortDesc     bool       `json:"sort_desc"`
}

// OperationStatsRequest 操作统计请求
type OperationStatsRequest struct {
	RepositoryID *uuid.UUID `json:"repository_id"`
	UserID       *uuid.UUID `json:"user_id"`
	StartTime    *time.Time `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	GroupBy      string     `json:"group_by"` // day, hour, operation, protocol
}

// OperationStats 操作统计结果
type OperationStats struct {
	TotalOperations   int64                 `json:"total_operations"`
	SuccessOperations int64                 `json:"success_operations"`
	FailedOperations  int64                 `json:"failed_operations"`
	SuccessRate       float64               `json:"success_rate"`
	AverageDuration   float64               `json:"average_duration"`
	TotalBytesTransferred int64             `json:"total_bytes_transferred"`
	GroupedStats      []GroupedStat         `json:"grouped_stats"`
	OperationBreakdown map[string]int64     `json:"operation_breakdown"`
	ProtocolBreakdown map[string]int64      `json:"protocol_breakdown"`
}

// GroupedStat 分组统计
type GroupedStat struct {
	Group           string    `json:"group"`
	Count           int64     `json:"count"`
	SuccessCount    int64     `json:"success_count"`
	FailedCount     int64     `json:"failed_count"`
	SuccessRate     float64   `json:"success_rate"`
	AverageDuration float64   `json:"average_duration"`
	BytesTransferred int64    `json:"bytes_transferred"`
	Timestamp       *time.Time `json:"timestamp,omitempty"`
}

// Git操作常量
const (
	OperationPush   = "push"
	OperationPull   = "pull"
	OperationClone  = "clone"
	OperationFetch  = "fetch"
	OperationLsRefs = "ls-refs"
	OperationUploadPack = "upload-pack"
	OperationReceivePack = "receive-pack"
)

// RecordOperation 记录Git操作
func (s *gitOperationService) RecordOperation(req *RecordOperationRequest) (*models.GitOperation, error) {
	// 验证仓库和用户是否存在
	var repo models.Repository
	if err := s.db.Where("id = ? AND deleted_at IS NULL", req.RepositoryID).First(&repo).Error; err != nil {
		return nil, fmt.Errorf("仓库不存在")
	}

	// 创建操作记录
	operation := &models.GitOperation{
		RepositoryID:     req.RepositoryID,
		UserID:           req.UserID,
		Operation:        req.Operation,
		Protocol:         req.Protocol,
		RefName:          req.RefName,
		CommitSHA:        req.CommitSHA,
		ClientIP:         req.ClientIP,
		UserAgent:        req.UserAgent,
		Success:          req.Success,
		ErrorMsg:         req.ErrorMsg,
		Duration:         req.Duration,
		BytesTransferred: req.BytesTransferred,
		CreatedAt:        time.Now(),
	}

	if err := s.db.Create(operation).Error; err != nil {
		return nil, fmt.Errorf("记录Git操作失败: %w", err)
	}

	return operation, nil
}

// GetByID 根据ID获取操作记录
func (s *gitOperationService) GetByID(id uuid.UUID) (*models.GitOperation, error) {
	var operation models.GitOperation
	if err := s.db.Where("id = ?", id).
		Preload("Repository").
		First(&operation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("操作记录不存在")
		}
		return nil, fmt.Errorf("获取操作记录失败: %w", err)
	}
	return &operation, nil
}

// GetByRepository 获取仓库的操作记录
func (s *gitOperationService) GetByRepository(repositoryID uuid.UUID) ([]models.GitOperation, error) {
	var operations []models.GitOperation
	if err := s.db.Where("repository_id = ?", repositoryID).
		Order("created_at DESC").
		Limit(100).
		Find(&operations).Error; err != nil {
		return nil, fmt.Errorf("获取仓库操作记录失败: %w", err)
	}
	return operations, nil
}

// GetByUser 获取用户的操作记录
func (s *gitOperationService) GetByUser(userID uuid.UUID) ([]models.GitOperation, error) {
	var operations []models.GitOperation
	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(100).
		Find(&operations).Error; err != nil {
		return nil, fmt.Errorf("获取用户操作记录失败: %w", err)
	}
	return operations, nil
}

// GetOperationStats 获取操作统计信息
func (s *gitOperationService) GetOperationStats(req *OperationStatsRequest) (*OperationStats, error) {
	query := s.db.Model(&models.GitOperation{})

	// 应用筛选条件
	if req.RepositoryID != nil {
		query = query.Where("repository_id = ?", *req.RepositoryID)
	}
	
	if req.UserID != nil {
		query = query.Where("user_id = ?", *req.UserID)
	}
	
	if req.StartTime != nil {
		query = query.Where("created_at >= ?", *req.StartTime)
	}
	
	if req.EndTime != nil {
		query = query.Where("created_at <= ?", *req.EndTime)
	}

	// 基础统计
	var totalOps, successOps, failedOps int64
	var totalDuration, totalBytes int64

	// 总操作数
	if err := query.Count(&totalOps).Error; err != nil {
		return nil, fmt.Errorf("统计总操作数失败: %w", err)
	}

	// 成功操作数
	if err := query.Where("success = ?", true).Count(&successOps).Error; err != nil {
		return nil, fmt.Errorf("统计成功操作数失败: %w", err)
	}

	failedOps = totalOps - successOps

	// 计算成功率
	var successRate float64
	if totalOps > 0 {
		successRate = float64(successOps) / float64(totalOps) * 100
	}

	// 平均耗时和总传输字节数
	var avgDuration float64
	if totalOps > 0 {
		type Result struct {
			TotalDuration int64 `json:"total_duration"`
			TotalBytes    int64 `json:"total_bytes"`
		}
		var result Result
		if err := query.Select("SUM(duration) as total_duration, SUM(bytes_transferred) as total_bytes").
			Scan(&result).Error; err != nil {
			return nil, fmt.Errorf("计算统计信息失败: %w", err)
		}
		totalDuration = result.TotalDuration
		totalBytes = result.TotalBytes
		avgDuration = float64(totalDuration) / float64(totalOps)
	}

	// 按操作类型分组统计
	operationBreakdown := make(map[string]int64)
	var operationStats []struct {
		Operation string `json:"operation"`
		Count     int64  `json:"count"`
	}
	if err := query.Select("operation, COUNT(*) as count").
		Group("operation").
		Scan(&operationStats).Error; err != nil {
		return nil, fmt.Errorf("统计操作类型失败: %w", err)
	}
	for _, stat := range operationStats {
		operationBreakdown[stat.Operation] = stat.Count
	}

	// 按协议分组统计
	protocolBreakdown := make(map[string]int64)
	var protocolStats []struct {
		Protocol string `json:"protocol"`
		Count    int64  `json:"count"`
	}
	if err := query.Select("protocol, COUNT(*) as count").
		Group("protocol").
		Scan(&protocolStats).Error; err != nil {
		return nil, fmt.Errorf("统计协议类型失败: %w", err)
	}
	for _, stat := range protocolStats {
		protocolBreakdown[stat.Protocol] = stat.Count
	}

	// 分组统计
	var groupedStats []GroupedStat
	if req.GroupBy != "" {
		groupedStats, _ = s.getGroupedStats(query, req.GroupBy)
	}

	stats := &OperationStats{
		TotalOperations:       totalOps,
		SuccessOperations:     successOps,
		FailedOperations:      failedOps,
		SuccessRate:           successRate,
		AverageDuration:       avgDuration,
		TotalBytesTransferred: totalBytes,
		GroupedStats:          groupedStats,
		OperationBreakdown:    operationBreakdown,
		ProtocolBreakdown:     protocolBreakdown,
	}

	return stats, nil
}

// getGroupedStats 获取分组统计数据
func (s *gitOperationService) getGroupedStats(query *gorm.DB, groupBy string) ([]GroupedStat, error) {
	var groupedStats []GroupedStat
	
	var selectFields, groupFields string
	switch groupBy {
	case "day":
		selectFields = "DATE(created_at) as group_key, COUNT(*) as count, " +
			"COUNT(CASE WHEN success = true THEN 1 END) as success_count, " +
			"COUNT(CASE WHEN success = false THEN 1 END) as failed_count, " +
			"AVG(duration) as avg_duration, SUM(bytes_transferred) as bytes_transferred, " +
			"DATE(created_at) as timestamp"
		groupFields = "DATE(created_at)"
	case "hour":
		selectFields = "DATE_TRUNC('hour', created_at) as group_key, COUNT(*) as count, " +
			"COUNT(CASE WHEN success = true THEN 1 END) as success_count, " +
			"COUNT(CASE WHEN success = false THEN 1 END) as failed_count, " +
			"AVG(duration) as avg_duration, SUM(bytes_transferred) as bytes_transferred, " +
			"DATE_TRUNC('hour', created_at) as timestamp"
		groupFields = "DATE_TRUNC('hour', created_at)"
	case "operation":
		selectFields = "operation as group_key, COUNT(*) as count, " +
			"COUNT(CASE WHEN success = true THEN 1 END) as success_count, " +
			"COUNT(CASE WHEN success = false THEN 1 END) as failed_count, " +
			"AVG(duration) as avg_duration, SUM(bytes_transferred) as bytes_transferred"
		groupFields = "operation"
	case "protocol":
		selectFields = "protocol as group_key, COUNT(*) as count, " +
			"COUNT(CASE WHEN success = true THEN 1 END) as success_count, " +
			"COUNT(CASE WHEN success = false THEN 1 END) as failed_count, " +
			"AVG(duration) as avg_duration, SUM(bytes_transferred) as bytes_transferred"
		groupFields = "protocol"
	default:
		return groupedStats, nil
	}

	type GroupResult struct {
		GroupKey         string     `json:"group_key"`
		Count            int64      `json:"count"`
		SuccessCount     int64      `json:"success_count"`
		FailedCount      int64      `json:"failed_count"`
		AvgDuration      float64    `json:"avg_duration"`
		BytesTransferred int64      `json:"bytes_transferred"`
		Timestamp        *time.Time `json:"timestamp,omitempty"`
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

		stat := GroupedStat{
			Group:            result.GroupKey,
			Count:            result.Count,
			SuccessCount:     result.SuccessCount,
			FailedCount:      result.FailedCount,
			SuccessRate:      successRate,
			AverageDuration:  result.AvgDuration,
			BytesTransferred: result.BytesTransferred,
			Timestamp:        result.Timestamp,
		}
		groupedStats = append(groupedStats, stat)
	}

	return groupedStats, nil
}

// List 列表查询操作记录
func (s *gitOperationService) List(req *ListOperationsRequest) ([]models.GitOperation, int64, error) {
	query := s.db.Model(&models.GitOperation{})

	// 应用筛选条件
	if req.RepositoryID != nil {
		query = query.Where("repository_id = ?", *req.RepositoryID)
	}
	
	if req.UserID != nil {
		query = query.Where("user_id = ?", *req.UserID)
	}
	
	if req.Operation != nil {
		query = query.Where("operation = ?", *req.Operation)
	}
	
	if req.Protocol != nil {
		query = query.Where("protocol = ?", *req.Protocol)
	}
	
	if req.Success != nil {
		query = query.Where("success = ?", *req.Success)
	}
	
	if req.StartTime != nil {
		query = query.Where("created_at >= ?", *req.StartTime)
	}
	
	if req.EndTime != nil {
		query = query.Where("created_at <= ?", *req.EndTime)
	}
	
	if req.ClientIP != nil {
		query = query.Where("client_ip = ?", *req.ClientIP)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计操作记录总数失败: %w", err)
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

	var operations []models.GitOperation
	if err := query.Preload("Repository").Find(&operations).Error; err != nil {
		return nil, 0, fmt.Errorf("查询操作记录列表失败: %w", err)
	}

	return operations, total, nil
}

// CleanupOldRecords 清理旧的操作记录
func (s *gitOperationService) CleanupOldRecords(retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	
	result := s.db.Where("created_at < ?", cutoffTime).Delete(&models.GitOperation{})
	if result.Error != nil {
		return fmt.Errorf("清理旧操作记录失败: %w", result.Error)
	}

	fmt.Printf("已清理 %d 条超过 %d 天的操作记录\n", result.RowsAffected, retentionDays)
	return nil
}