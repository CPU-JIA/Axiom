package services

import (
	"fmt"
	"time"

	"git-gateway-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BranchService 分支服务接口
type BranchService interface {
	Create(req *CreateBranchRequest) (*models.Branch, error)
	GetByID(id uuid.UUID) (*models.Branch, error)
	GetByRepository(repositoryID uuid.UUID) ([]models.Branch, error)
	GetByName(repositoryID uuid.UUID, name string) (*models.Branch, error)
	Update(id uuid.UUID, req *UpdateBranchRequest) (*models.Branch, error)
	Delete(id uuid.UUID) error
	SetProtection(id uuid.UUID, protection models.BranchProtection) error
	RemoveProtection(id uuid.UUID) error
	SetDefault(repositoryID uuid.UUID, branchID uuid.UUID) error
	List(req *ListBranchesRequest) ([]models.Branch, int64, error)
}

type branchService struct {
	db *gorm.DB
}

// NewBranchService 创建分支服务实例
func NewBranchService(db *gorm.DB) BranchService {
	return &branchService{db: db}
}

// CreateBranchRequest 创建分支请求
type CreateBranchRequest struct {
	RepositoryID uuid.UUID `json:"repository_id" validate:"required"`
	Name         string    `json:"name" validate:"required,max=255"`
	CommitSHA    string    `json:"commit_sha" validate:"required,len=40"`
	IsProtected  bool      `json:"is_protected"`
	Protection   *models.BranchProtection `json:"protection"`
}

// UpdateBranchRequest 更新分支请求
type UpdateBranchRequest struct {
	Name        *string `json:"name" validate:"omitempty,max=255"`
	CommitSHA   *string `json:"commit_sha" validate:"omitempty,len=40"`
	IsProtected *bool   `json:"is_protected"`
	Protection  *models.BranchProtection `json:"protection"`
}

// ListBranchesRequest 列表查询请求
type ListBranchesRequest struct {
	RepositoryID *uuid.UUID `json:"repository_id"`
	IsProtected  *bool      `json:"is_protected"`
	IsDefault    *bool      `json:"is_default"`
	Search       *string    `json:"search"`
	Page         int        `json:"page"`
	Limit        int        `json:"limit"`
	SortBy       string     `json:"sort_by"`
	SortDesc     bool       `json:"sort_desc"`
}

// Create 创建分支
func (s *branchService) Create(req *CreateBranchRequest) (*models.Branch, error) {
	// 检查同仓库下分支名称唯一性
	var existingBranch models.Branch
	if err := s.db.Where("repository_id = ? AND name = ?", 
		req.RepositoryID, req.Name).First(&existingBranch).Error; err == nil {
		return nil, fmt.Errorf("分支名称 '%s' 已存在", req.Name)
	}

	// 创建分支记录
	branch := &models.Branch{
		RepositoryID: req.RepositoryID,
		Name:         req.Name,
		CommitSHA:    req.CommitSHA,
		IsDefault:    false, // 新建分支默认不是主分支
		IsProtected:  req.IsProtected,
	}

	// 设置分支保护
	if req.Protection != nil {
		branch.Protection = *req.Protection
	}

	if err := s.db.Create(branch).Error; err != nil {
		return nil, fmt.Errorf("创建分支失败: %w", err)
	}

	return branch, nil
}

// GetByID 根据ID获取分支
func (s *branchService) GetByID(id uuid.UUID) (*models.Branch, error) {
	var branch models.Branch
	if err := s.db.Where("id = ?", id).
		Preload("Repository").
		First(&branch).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分支不存在")
		}
		return nil, fmt.Errorf("获取分支失败: %w", err)
	}
	return &branch, nil
}

// GetByRepository 获取仓库的所有分支
func (s *branchService) GetByRepository(repositoryID uuid.UUID) ([]models.Branch, error) {
	var branches []models.Branch
	if err := s.db.Where("repository_id = ?", repositoryID).
		Order("is_default DESC, created_at DESC").Find(&branches).Error; err != nil {
		return nil, fmt.Errorf("获取仓库分支失败: %w", err)
	}
	return branches, nil
}

// GetByName 根据仓库ID和分支名称获取分支
func (s *branchService) GetByName(repositoryID uuid.UUID, name string) (*models.Branch, error) {
	var branch models.Branch
	if err := s.db.Where("repository_id = ? AND name = ?", 
		repositoryID, name).First(&branch).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分支不存在")
		}
		return nil, fmt.Errorf("获取分支失败: %w", err)
	}
	return &branch, nil
}

// Update 更新分支
func (s *branchService) Update(id uuid.UUID, req *UpdateBranchRequest) (*models.Branch, error) {
	branch, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	
	if req.Name != nil {
		// 检查名称唯一性
		var existingBranch models.Branch
		if err := s.db.Where("repository_id = ? AND name = ? AND id != ?", 
			branch.RepositoryID, *req.Name, id).First(&existingBranch).Error; err == nil {
			return nil, fmt.Errorf("分支名称 '%s' 已存在", *req.Name)
		}
		updates["name"] = *req.Name
	}
	
	if req.CommitSHA != nil {
		updates["commit_sha"] = *req.CommitSHA
	}
	
	if req.IsProtected != nil {
		updates["is_protected"] = *req.IsProtected
	}
	
	if req.Protection != nil {
		updates["protection"] = *req.Protection
	}

	updates["updated_at"] = time.Now()

	if err := s.db.Model(branch).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新分支失败: %w", err)
	}

	return s.GetByID(id)
}

// Delete 删除分支
func (s *branchService) Delete(id uuid.UUID) error {
	branch, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// 不能删除默认分支
	if branch.IsDefault {
		return fmt.Errorf("不能删除默认分支")
	}

	if err := s.db.Delete(branch).Error; err != nil {
		return fmt.Errorf("删除分支失败: %w", err)
	}

	return nil
}

// SetProtection 设置分支保护
func (s *branchService) SetProtection(id uuid.UUID, protection models.BranchProtection) error {
	branch, err := s.GetByID(id)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"is_protected": true,
		"protection":   protection,
		"updated_at":   time.Now(),
	}

	if err := s.db.Model(branch).Updates(updates).Error; err != nil {
		return fmt.Errorf("设置分支保护失败: %w", err)
	}

	return nil
}

// RemoveProtection 移除分支保护
func (s *branchService) RemoveProtection(id uuid.UUID) error {
	branch, err := s.GetByID(id)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"is_protected": false,
		"protection":   models.BranchProtection{}, // 重置为默认值
		"updated_at":   time.Now(),
	}

	if err := s.db.Model(branch).Updates(updates).Error; err != nil {
		return fmt.Errorf("移除分支保护失败: %w", err)
	}

	return nil
}

// SetDefault 设置默认分支
func (s *branchService) SetDefault(repositoryID uuid.UUID, branchID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 取消当前默认分支
		if err := tx.Model(&models.Branch{}).
			Where("repository_id = ? AND is_default = ?", repositoryID, true).
			Update("is_default", false).Error; err != nil {
			return fmt.Errorf("取消当前默认分支失败: %w", err)
		}

		// 设置新的默认分支
		if err := tx.Model(&models.Branch{}).
			Where("id = ?", branchID).
			Updates(map[string]interface{}{
				"is_default": true,
				"updated_at": time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("设置默认分支失败: %w", err)
		}

		// 更新仓库的默认分支字段
		var branch models.Branch
		if err := tx.Where("id = ?", branchID).First(&branch).Error; err != nil {
			return fmt.Errorf("获取分支信息失败: %w", err)
		}

		if err := tx.Model(&models.Repository{}).
			Where("id = ?", repositoryID).
			Update("default_branch", branch.Name).Error; err != nil {
			return fmt.Errorf("更新仓库默认分支失败: %w", err)
		}

		return nil
	})
}

// List 列表查询分支
func (s *branchService) List(req *ListBranchesRequest) ([]models.Branch, int64, error) {
	query := s.db.Model(&models.Branch{})

	// 应用筛选条件
	if req.RepositoryID != nil {
		query = query.Where("repository_id = ?", *req.RepositoryID)
	}
	
	if req.IsProtected != nil {
		query = query.Where("is_protected = ?", *req.IsProtected)
	}
	
	if req.IsDefault != nil {
		query = query.Where("is_default = ?", *req.IsDefault)
	}
	
	if req.Search != nil {
		searchTerm := fmt.Sprintf("%%%s%%", *req.Search)
		query = query.Where("name ILIKE ?", searchTerm)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计分支总数失败: %w", err)
	}

	// 应用排序
	sortBy := "is_default DESC, created_at"
	if req.SortBy != "" {
		sortBy = req.SortBy
		if req.SortDesc {
			sortBy += " DESC"
		} else {
			sortBy += " ASC"
		}
	}
	query = query.Order(sortBy)

	// 应用分页
	if req.Page > 0 && req.Limit > 0 {
		offset := (req.Page - 1) * req.Limit
		query = query.Offset(offset).Limit(req.Limit)
	}

	var branches []models.Branch
	if err := query.Preload("Repository").Find(&branches).Error; err != nil {
		return nil, 0, fmt.Errorf("查询分支列表失败: %w", err)
	}

	return branches, total, nil
}