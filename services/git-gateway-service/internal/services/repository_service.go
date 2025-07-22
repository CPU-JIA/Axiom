package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"git-gateway-service/internal/config"
	"git-gateway-service/internal/models"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RepositoryService 仓库服务接口
type RepositoryService interface {
	Create(req *CreateRepositoryRequest) (*models.Repository, error)
	GetByID(id uuid.UUID) (*models.Repository, error)
	GetByProjectID(projectID uuid.UUID) ([]models.Repository, error)
	GetByName(projectID uuid.UUID, name string) (*models.Repository, error)
	Update(id uuid.UUID, req *UpdateRepositoryRequest) (*models.Repository, error)
	Delete(id uuid.UUID) error
	GetStatistics(id uuid.UUID) (*RepositoryStats, error)
	UpdateStatistics(id uuid.UUID) error
	List(req *ListRepositoriesRequest) ([]models.Repository, int64, error)
	InitializeGitRepository(repo *models.Repository) error
	GetRepositorySize(repoPath string) (int64, error)
}

type repositoryService struct {
	db     *gorm.DB
	config *config.Config
}

// NewRepositoryService 创建仓库服务实例
func NewRepositoryService(db *gorm.DB, cfg *config.Config) RepositoryService {
	return &repositoryService{
		db:     db,
		config: cfg,
	}
}

// CreateRepositoryRequest 创建仓库请求
type CreateRepositoryRequest struct {
	ProjectID     uuid.UUID               `json:"project_id" validate:"required"`
	Name          string                  `json:"name" validate:"required,max=255"`
	Description   *string                 `json:"description"`
	Visibility    string                  `json:"visibility" validate:"required,oneof=private internal public"`
	DefaultBranch string                  `json:"default_branch" validate:"required,max=255"`
	Language      *string                 `json:"language" validate:"omitempty,max=50"`
	Topics        []string                `json:"topics"`
	Settings      models.RepositorySettings `json:"settings"`
}

// UpdateRepositoryRequest 更新仓库请求
type UpdateRepositoryRequest struct {
	Name          *string                 `json:"name" validate:"omitempty,max=255"`
	Description   *string                 `json:"description"`
	Visibility    *string                 `json:"visibility" validate:"omitempty,oneof=private internal public"`
	DefaultBranch *string                 `json:"default_branch" validate:"omitempty,max=255"`
	Language      *string                 `json:"language" validate:"omitempty,max=50"`
	Topics        []string                `json:"topics"`
	Settings      *models.RepositorySettings `json:"settings"`
}

// ListRepositoriesRequest 列表查询请求
type ListRepositoriesRequest struct {
	ProjectID  *uuid.UUID `json:"project_id"`
	Visibility *string    `json:"visibility"`
	Language   *string    `json:"language"`
	Search     *string    `json:"search"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	SortBy     string     `json:"sort_by"`
	SortDesc   bool       `json:"sort_desc"`
}

// RepositoryStats 仓库统计信息
type RepositoryStats struct {
	Size        int64 `json:"size"`
	CommitCount int   `json:"commit_count"`
	BranchCount int   `json:"branch_count"`
	TagCount    int   `json:"tag_count"`
}

// Create 创建仓库
func (s *repositoryService) Create(req *CreateRepositoryRequest) (*models.Repository, error) {
	// 检查同项目下仓库名称唯一性
	var existingRepo models.Repository
	if err := s.db.Where("project_id = ? AND name = ? AND deleted_at IS NULL", 
		req.ProjectID, req.Name).First(&existingRepo).Error; err == nil {
		return nil, fmt.Errorf("仓库名称 '%s' 已存在", req.Name)
	}

	// 创建仓库记录
	repo := &models.Repository{
		ProjectID:     req.ProjectID,
		Name:          req.Name,
		Description:   req.Description,
		Visibility:    req.Visibility,
		DefaultBranch: req.DefaultBranch,
		Language:      req.Language,
		Settings:      req.Settings,
		CommitCount:   0,
		BranchCount:   1, // 默认分支
		TagCount:      0,
		Size:          0,
	}

	// 生成仓库URLs
	repo.GitURL = fmt.Sprintf("git@localhost:%s/%s.git", req.ProjectID, req.Name)
	repo.HTTPUrl = fmt.Sprintf("http://localhost:8004/%s/%s.git", req.ProjectID, req.Name)
	repo.SSHUrl = fmt.Sprintf("git@localhost:2222/%s/%s.git", req.ProjectID, req.Name)

	// 处理Topics
	if req.Topics != nil {
		topicsJSON, _ := json.Marshal(req.Topics)
		repo.Topics = topicsJSON
	}

	// 保存到数据库
	if err := s.db.Create(repo).Error; err != nil {
		return nil, fmt.Errorf("创建仓库失败: %w", err)
	}

	// 初始化Git仓库
	if err := s.InitializeGitRepository(repo); err != nil {
		// 回滚数据库记录
		s.db.Delete(repo)
		return nil, fmt.Errorf("初始化Git仓库失败: %w", err)
	}

	return repo, nil
}

// GetByID 根据ID获取仓库
func (s *repositoryService) GetByID(id uuid.UUID) (*models.Repository, error) {
	var repo models.Repository
	if err := s.db.Where("id = ? AND deleted_at IS NULL", id).
		Preload("Branches").
		Preload("Tags").
		Preload("Webhooks").
		First(&repo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("仓库不存在")
		}
		return nil, fmt.Errorf("获取仓库失败: %w", err)
	}
	return &repo, nil
}

// GetByProjectID 根据项目ID获取仓库列表
func (s *repositoryService) GetByProjectID(projectID uuid.UUID) ([]models.Repository, error) {
	var repos []models.Repository
	if err := s.db.Where("project_id = ? AND deleted_at IS NULL", projectID).
		Order("created_at DESC").Find(&repos).Error; err != nil {
		return nil, fmt.Errorf("获取项目仓库失败: %w", err)
	}
	return repos, nil
}

// GetByName 根据项目ID和仓库名称获取仓库
func (s *repositoryService) GetByName(projectID uuid.UUID, name string) (*models.Repository, error) {
	var repo models.Repository
	if err := s.db.Where("project_id = ? AND name = ? AND deleted_at IS NULL", 
		projectID, name).First(&repo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("仓库不存在")
		}
		return nil, fmt.Errorf("获取仓库失败: %w", err)
	}
	return &repo, nil
}

// Update 更新仓库
func (s *repositoryService) Update(id uuid.UUID, req *UpdateRepositoryRequest) (*models.Repository, error) {
	repo, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	
	if req.Name != nil {
		// 检查名称唯一性
		var existingRepo models.Repository
		if err := s.db.Where("project_id = ? AND name = ? AND id != ? AND deleted_at IS NULL", 
			repo.ProjectID, *req.Name, id).First(&existingRepo).Error; err == nil {
			return nil, fmt.Errorf("仓库名称 '%s' 已存在", *req.Name)
		}
		updates["name"] = *req.Name
	}
	
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	
	if req.Visibility != nil {
		updates["visibility"] = *req.Visibility
	}
	
	if req.DefaultBranch != nil {
		updates["default_branch"] = *req.DefaultBranch
	}
	
	if req.Language != nil {
		updates["language"] = *req.Language
	}
	
	if req.Topics != nil {
		topicsJSON, _ := json.Marshal(req.Topics)
		updates["topics"] = topicsJSON
	}
	
	if req.Settings != nil {
		updates["settings"] = *req.Settings
	}

	updates["updated_at"] = time.Now()

	if err := s.db.Model(repo).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新仓库失败: %w", err)
	}

	return s.GetByID(id)
}

// Delete 删除仓库（软删除）
func (s *repositoryService) Delete(id uuid.UUID) error {
	repo, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// 软删除数据库记录
	now := time.Now()
	if err := s.db.Model(repo).Update("deleted_at", now).Error; err != nil {
		return fmt.Errorf("删除仓库失败: %w", err)
	}

	// TODO: 可选择性删除Git仓库文件（根据配置）
	// 生产环境中可能需要保留数据一段时间

	return nil
}

// GetStatistics 获取仓库统计信息
func (s *repositoryService) GetStatistics(id uuid.UUID) (*RepositoryStats, error) {
	repo, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	stats := &RepositoryStats{
		Size:        repo.Size,
		CommitCount: repo.CommitCount,
		BranchCount: repo.BranchCount,
		TagCount:    repo.TagCount,
	}

	return stats, nil
}

// UpdateStatistics 更新仓库统计信息
func (s *repositoryService) UpdateStatistics(id uuid.UUID) error {
	repo, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// 计算仓库大小
	repoPath := filepath.Join(s.config.Git.RepositoryRoot, 
		repo.ProjectID.String(), repo.Name)
	size, err := s.GetRepositorySize(repoPath)
	if err != nil {
		return fmt.Errorf("计算仓库大小失败: %w", err)
	}

	// 统计分支数量
	var branchCount int64
	if err := s.db.Model(&models.Branch{}).Where("repository_id = ?", id).Count(&branchCount).Error; err != nil {
		return fmt.Errorf("统计分支数量失败: %w", err)
	}

	// 统计标签数量
	var tagCount int64
	if err := s.db.Model(&models.Tag{}).Where("repository_id = ?", id).Count(&tagCount).Error; err != nil {
		return fmt.Errorf("统计标签数量失败: %w", err)
	}

	// TODO: 统计提交数量（需要解析Git历史）
	commitCount := repo.CommitCount // 暂时保持原值

	// 更新数据库
	updates := map[string]interface{}{
		"size":         size,
		"commit_count": commitCount,
		"branch_count": int(branchCount),
		"tag_count":    int(tagCount),
		"updated_at":   time.Now(),
	}

	if err := s.db.Model(repo).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新仓库统计信息失败: %w", err)
	}

	return nil
}

// List 列表查询仓库
func (s *repositoryService) List(req *ListRepositoriesRequest) ([]models.Repository, int64, error) {
	query := s.db.Model(&models.Repository{}).Where("deleted_at IS NULL")

	// 应用筛选条件
	if req.ProjectID != nil {
		query = query.Where("project_id = ?", *req.ProjectID)
	}
	
	if req.Visibility != nil {
		query = query.Where("visibility = ?", *req.Visibility)
	}
	
	if req.Language != nil {
		query = query.Where("language = ?", *req.Language)
	}
	
	if req.Search != nil {
		searchTerm := fmt.Sprintf("%%%s%%", *req.Search)
		query = query.Where("name ILIKE ? OR description ILIKE ?", searchTerm, searchTerm)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计仓库总数失败: %w", err)
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

	var repos []models.Repository
	if err := query.Find(&repos).Error; err != nil {
		return nil, 0, fmt.Errorf("查询仓库列表失败: %w", err)
	}

	return repos, total, nil
}

// InitializeGitRepository 初始化Git仓库
func (s *repositoryService) InitializeGitRepository(repo *models.Repository) error {
	repoPath := filepath.Join(s.config.Git.RepositoryRoot, 
		repo.ProjectID.String(), repo.Name)

	// 创建目录
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return fmt.Errorf("创建仓库目录失败: %w", err)
	}

	// 初始化裸仓库
	gitRepo, err := git.PlainInit(repoPath, true)
	if err != nil {
		return fmt.Errorf("初始化Git仓库失败: %w", err)
	}

	// 创建默认分支记录
	branch := &models.Branch{
		RepositoryID: repo.ID,
		Name:         repo.DefaultBranch,
		CommitSHA:    "0000000000000000000000000000000000000000", // 空仓库
		IsDefault:    true,
		IsProtected:  false,
	}

	if err := s.db.Create(branch).Error; err != nil {
		return fmt.Errorf("创建默认分支记录失败: %w", err)
	}

	// 设置仓库配置
	cfg, err := gitRepo.Config()
	if err == nil {
		// TODO: 设置仓库特定配置
		_ = cfg
	}

	return nil
}

// GetRepositorySize 计算仓库大小
func (s *repositoryService) GetRepositorySize(repoPath string) (int64, error) {
	var size int64
	
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	
	if err != nil {
		return 0, err
	}
	
	return size, nil
}