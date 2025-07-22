package services

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cicd-service/internal/config"
	"cicd-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CacheService 构建缓存服务接口
type CacheService interface {
	// 缓存操作
	Store(req *StoreCacheRequest) (*models.BuildCache, error)
	Retrieve(key string, projectID uuid.UUID) (*models.BuildCache, error)
	Delete(id uuid.UUID) error
	DeleteByKey(key string, projectID uuid.UUID) error
	
	// 缓存管理
	List(req *ListCacheRequest) ([]models.BuildCache, int64, error)
	GetStatistics(projectID *uuid.UUID) (*CacheStats, error)
	Cleanup() error
	ValidateCache(cache *models.BuildCache) bool
	
	// 文件操作
	GetCachePath(cache *models.BuildCache) string
	CalculateChecksum(filePath string) (string, error)
}

type cacheService struct {
	db     *gorm.DB
	config *config.Config
}

// NewCacheService 创建构建缓存服务实例
func NewCacheService(db *gorm.DB, cfg *config.Config) CacheService {
	// 确保缓存目录存在
	if err := os.MkdirAll(cfg.Cache.LocalPath, 0755); err != nil {
		fmt.Printf("创建缓存目录失败: %v\n", err)
	}
	
	return &cacheService{
		db:     db,
		config: cfg,
	}
}

// StoreCacheRequest 存储缓存请求
type StoreCacheRequest struct {
	ProjectID  uuid.UUID              `json:"project_id" validate:"required"`
	Key        string                 `json:"key" validate:"required,max=255"`
	SourcePath string                 `json:"source_path" validate:"required"`
	Metadata   map[string]interface{} `json:"metadata"`
	TTLHours   *int                   `json:"ttl_hours"`
}

// ListCacheRequest 列表查询请求
type ListCacheRequest struct {
	ProjectID *uuid.UUID `json:"project_id"`
	Search    *string    `json:"search"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
	SortBy    string     `json:"sort_by"`
	SortDesc  bool       `json:"sort_desc"`
}

// CacheStats 缓存统计信息
type CacheStats struct {
	TotalCaches      int64   `json:"total_caches"`
	TotalSize        int64   `json:"total_size"`        // 总大小(bytes)
	TotalHits        int64   `json:"total_hits"`        // 总命中次数
	HitRate          float64 `json:"hit_rate"`          // 命中率
	AverageSize      float64 `json:"average_size"`      // 平均大小
	ExpiredCaches    int64   `json:"expired_caches"`    // 过期缓存数量
	RecentActivity   []CacheActivity `json:"recent_activity"` // 最近活动
	SizeDistribution []SizeBucket    `json:"size_distribution"` // 大小分布
}

// CacheActivity 缓存活动
type CacheActivity struct {
	CacheID   uuid.UUID `json:"cache_id"`
	Key       string    `json:"key"`
	Action    string    `json:"action"` // store, hit, miss
	Size      int64     `json:"size"`
	Timestamp time.Time `json:"timestamp"`
}

// SizeBucket 大小分组
type SizeBucket struct {
	Range string `json:"range"`
	Count int64  `json:"count"`
	Size  int64  `json:"size"`
}

// Store 存储缓存
func (s *cacheService) Store(req *StoreCacheRequest) (*models.BuildCache, error) {
	// 检查源文件是否存在
	if _, err := os.Stat(req.SourcePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("源文件不存在: %s", req.SourcePath)
	}

	// 检查缓存是否已存在
	var existingCache models.BuildCache
	if err := s.db.Where("project_id = ? AND key = ?", req.ProjectID, req.Key).
		First(&existingCache).Error; err == nil {
		// 缓存已存在，删除旧的
		s.Delete(existingCache.ID)
	}

	// 计算校验和
	checksum, err := s.CalculateChecksum(req.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("计算文件校验和失败: %w", err)
	}

	// 获取文件大小
	fileInfo, err := os.Stat(req.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 生成缓存路径
	cacheID := uuid.New()
	cachePath := s.generateCachePath(req.ProjectID, req.Key, cacheID)

	// 确保缓存目录存在
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("创建缓存目录失败: %w", err)
	}

	// 复制文件到缓存位置
	if err := s.copyFile(req.SourcePath, cachePath); err != nil {
		return nil, fmt.Errorf("复制文件到缓存失败: %w", err)
	}

	// 创建缓存记录
	cache := &models.BuildCache{
		ID:        cacheID,
		ProjectID: req.ProjectID,
		Key:       req.Key,
		Path:      cachePath,
		Size:      fileInfo.Size(),
		HitCount:  0,
		Checksum:  checksum,
	}

	// 处理元数据
	if req.Metadata != nil {
		metadataJSON, err := jsonMarshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("序列化元数据失败: %w", err)
		}
		cache.Metadata = metadataJSON
	}

	// 设置过期时间
	ttlHours := s.config.Cache.TTLHours
	if req.TTLHours != nil {
		ttlHours = *req.TTLHours
	}
	if ttlHours > 0 {
		expiresAt := time.Now().Add(time.Duration(ttlHours) * time.Hour)
		cache.ExpiresAt = &expiresAt
	}

	// 保存到数据库
	if err := s.db.Create(cache).Error; err != nil {
		// 删除已复制的文件
		os.Remove(cachePath)
		return nil, fmt.Errorf("保存缓存记录失败: %w", err)
	}

	return cache, nil
}

// Retrieve 检索缓存
func (s *cacheService) Retrieve(key string, projectID uuid.UUID) (*models.BuildCache, error) {
	var cache models.BuildCache
	if err := s.db.Where("project_id = ? AND key = ?", projectID, key).
		First(&cache).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("缓存不存在")
		}
		return nil, fmt.Errorf("检索缓存失败: %w", err)
	}

	// 检查缓存是否过期
	if cache.ExpiresAt != nil && time.Now().After(*cache.ExpiresAt) {
		// 缓存已过期，删除
		s.Delete(cache.ID)
		return nil, fmt.Errorf("缓存已过期")
	}

	// 验证缓存文件是否存在
	if !s.ValidateCache(&cache) {
		// 缓存文件不存在或损坏，删除记录
		s.Delete(cache.ID)
		return nil, fmt.Errorf("缓存文件损坏或不存在")
	}

	// 更新命中统计
	s.updateHitStats(cache.ID)

	return &cache, nil
}

// updateHitStats 更新命中统计
func (s *cacheService) updateHitStats(cacheID uuid.UUID) {
	s.db.Model(&models.BuildCache{}).
		Where("id = ?", cacheID).
		Updates(map[string]interface{}{
			"hit_count":    gorm.Expr("hit_count + 1"),
			"last_used_at": time.Now(),
		})
}

// Delete 删除缓存
func (s *cacheService) Delete(id uuid.UUID) error {
	var cache models.BuildCache
	if err := s.db.Where("id = ?", id).First(&cache).Error; err != nil {
		return fmt.Errorf("缓存不存在")
	}

	// 删除缓存文件
	if err := os.Remove(cache.Path); err != nil && !os.IsNotExist(err) {
		fmt.Printf("删除缓存文件失败: %v\n", err)
	}

	// 删除数据库记录
	if err := s.db.Delete(&cache).Error; err != nil {
		return fmt.Errorf("删除缓存记录失败: %w", err)
	}

	return nil
}

// DeleteByKey 根据键删除缓存
func (s *cacheService) DeleteByKey(key string, projectID uuid.UUID) error {
	var cache models.BuildCache
	if err := s.db.Where("project_id = ? AND key = ?", projectID, key).
		First(&cache).Error; err != nil {
		return fmt.Errorf("缓存不存在")
	}

	return s.Delete(cache.ID)
}

// List 列表查询缓存
func (s *cacheService) List(req *ListCacheRequest) ([]models.BuildCache, int64, error) {
	query := s.db.Model(&models.BuildCache{})

	// 应用筛选条件
	if req.ProjectID != nil {
		query = query.Where("project_id = ?", *req.ProjectID)
	}

	if req.Search != nil {
		searchTerm := fmt.Sprintf("%%%s%%", *req.Search)
		query = query.Where("key ILIKE ?", searchTerm)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计缓存总数失败: %w", err)
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

	var caches []models.BuildCache
	if err := query.Find(&caches).Error; err != nil {
		return nil, 0, fmt.Errorf("查询缓存列表失败: %w", err)
	}

	return caches, total, nil
}

// GetStatistics 获取缓存统计信息
func (s *cacheService) GetStatistics(projectID *uuid.UUID) (*CacheStats, error) {
	stats := &CacheStats{}
	query := s.db.Model(&models.BuildCache{})

	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	}

	// 基础统计
	if err := query.Count(&stats.TotalCaches).Error; err != nil {
		return nil, fmt.Errorf("统计缓存总数失败: %w", err)
	}

	// 总大小和总命中数
	var result struct {
		TotalSize int64 `json:"total_size"`
		TotalHits int64 `json:"total_hits"`
	}
	if err := query.Select("SUM(size) as total_size, SUM(hit_count) as total_hits").
		Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("计算缓存统计失败: %w", err)
	}
	stats.TotalSize = result.TotalSize
	stats.TotalHits = result.TotalHits

	// 计算平均大小
	if stats.TotalCaches > 0 {
		stats.AverageSize = float64(stats.TotalSize) / float64(stats.TotalCaches)
	}

	// 计算命中率
	var totalAccess int64
	s.db.Model(&models.BuildCache{}).Select("SUM(hit_count + 1)").Scan(&totalAccess)
	if totalAccess > 0 {
		stats.HitRate = float64(stats.TotalHits) / float64(totalAccess) * 100
	}

	// 过期缓存数量
	s.db.Model(&models.BuildCache{}).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Count(&stats.ExpiredCaches)

	// 大小分布
	stats.SizeDistribution = s.getSizeDistribution(query)

	return stats, nil
}

// getSizeDistribution 获取大小分布
func (s *cacheService) getSizeDistribution(query *gorm.DB) []SizeBucket {
	buckets := []SizeBucket{
		{Range: "< 1MB", Count: 0, Size: 0},
		{Range: "1-10MB", Count: 0, Size: 0},
		{Range: "10-100MB", Count: 0, Size: 0},
		{Range: "100MB-1GB", Count: 0, Size: 0},
		{Range: "> 1GB", Count: 0, Size: 0},
	}

	var caches []models.BuildCache
	query.Select("size").Find(&caches)

	for _, cache := range caches {
		size := cache.Size
		if size < 1024*1024 { // < 1MB
			buckets[0].Count++
			buckets[0].Size += size
		} else if size < 10*1024*1024 { // 1-10MB
			buckets[1].Count++
			buckets[1].Size += size
		} else if size < 100*1024*1024 { // 10-100MB
			buckets[2].Count++
			buckets[2].Size += size
		} else if size < 1024*1024*1024 { // 100MB-1GB
			buckets[3].Count++
			buckets[3].Size += size
		} else { // > 1GB
			buckets[4].Count++
			buckets[4].Size += size
		}
	}

	return buckets
}

// Cleanup 清理过期和超限缓存
func (s *cacheService) Cleanup() error {
	// 清理过期缓存
	var expiredCaches []models.BuildCache
	if err := s.db.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Find(&expiredCaches).Error; err != nil {
		return fmt.Errorf("查找过期缓存失败: %w", err)
	}

	for _, cache := range expiredCaches {
		s.Delete(cache.ID)
	}

	// 检查总缓存大小限制
	var totalSize int64
	s.db.Model(&models.BuildCache{}).Select("SUM(size)").Scan(&totalSize)

	maxSizeBytes := int64(s.config.Cache.MaxSizeGB) * 1024 * 1024 * 1024
	if totalSize > maxSizeBytes {
		// 删除最久未使用的缓存
		var oldCaches []models.BuildCache
		s.db.Order("COALESCE(last_used_at, created_at) ASC").
			Limit(int(totalSize-maxSizeBytes) / (1024 * 1024)). // 大概估算需要删除的数量
			Find(&oldCaches)

		for _, cache := range oldCaches {
			s.Delete(cache.ID)
			totalSize -= cache.Size
			if totalSize <= maxSizeBytes {
				break
			}
		}
	}

	return nil
}

// ValidateCache 验证缓存完整性
func (s *cacheService) ValidateCache(cache *models.BuildCache) bool {
	// 检查文件是否存在
	if _, err := os.Stat(cache.Path); os.IsNotExist(err) {
		return false
	}

	// 验证校验和
	if cache.Checksum != "" {
		checksum, err := s.CalculateChecksum(cache.Path)
		if err != nil || checksum != cache.Checksum {
			return false
		}
	}

	return true
}

// GetCachePath 获取缓存路径
func (s *cacheService) GetCachePath(cache *models.BuildCache) string {
	return cache.Path
}

// generateCachePath 生成缓存路径
func (s *cacheService) generateCachePath(projectID uuid.UUID, key string, cacheID uuid.UUID) string {
	return filepath.Join(
		s.config.Cache.LocalPath,
		projectID.String(),
		fmt.Sprintf("%s-%s", key, cacheID.String()[:8]),
	)
}

// copyFile 复制文件
func (s *cacheService) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// CalculateChecksum 计算文件校验和
func (s *cacheService) CalculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}