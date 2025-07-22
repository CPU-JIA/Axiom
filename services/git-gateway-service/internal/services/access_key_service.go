package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"git-gateway-service/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// AccessKeyService 访问密钥服务接口
type AccessKeyService interface {
	Create(req *CreateAccessKeyRequest) (*models.AccessKey, error)
	GetByID(id uuid.UUID) (*models.AccessKey, error)
	GetByUser(userID uuid.UUID) ([]models.AccessKey, error)
	GetByRepository(repositoryID uuid.UUID) ([]models.AccessKey, error)
	GetByFingerprint(fingerprint string) (*models.AccessKey, error)
	Update(id uuid.UUID, req *UpdateAccessKeyRequest) (*models.AccessKey, error)
	Delete(id uuid.UUID) error
	ValidatePublicKey(publicKeyStr string) (*KeyInfo, error)
	UpdateLastUsed(id uuid.UUID) error
	List(req *ListAccessKeysRequest) ([]models.AccessKey, int64, error)
}

type accessKeyService struct {
	db *gorm.DB
}

// NewAccessKeyService 创建访问密钥服务实例
func NewAccessKeyService(db *gorm.DB) AccessKeyService {
	return &accessKeyService{db: db}
}

// CreateAccessKeyRequest 创建访问密钥请求
type CreateAccessKeyRequest struct {
	RepositoryID *uuid.UUID `json:"repository_id"` // 可选，为空则为全局密钥
	UserID       uuid.UUID  `json:"user_id" validate:"required"`
	Title        string     `json:"title" validate:"required,max=255"`
	PublicKey    string     `json:"public_key" validate:"required"`
	AccessLevel  string     `json:"access_level" validate:"required,oneof=read write admin"`
}

// UpdateAccessKeyRequest 更新访问密钥请求
type UpdateAccessKeyRequest struct {
	Title       *string `json:"title" validate:"omitempty,max=255"`
	AccessLevel *string `json:"access_level" validate:"omitempty,oneof=read write admin"`
}

// ListAccessKeysRequest 列表查询请求
type ListAccessKeysRequest struct {
	RepositoryID *uuid.UUID `json:"repository_id"`
	UserID       *uuid.UUID `json:"user_id"`
	AccessLevel  *string    `json:"access_level"`
	KeyType      *string    `json:"key_type"`
	Search       *string    `json:"search"`
	Page         int        `json:"page"`
	Limit        int        `json:"limit"`
	SortBy       string     `json:"sort_by"`
	SortDesc     bool       `json:"sort_desc"`
}

// KeyInfo 密钥信息
type KeyInfo struct {
	KeyType     string `json:"key_type"`
	Fingerprint string `json:"fingerprint"`
	KeySize     int    `json:"key_size"`
}

// Create 创建访问密钥
func (s *accessKeyService) Create(req *CreateAccessKeyRequest) (*models.AccessKey, error) {
	// 验证公钥格式
	keyInfo, err := s.ValidatePublicKey(req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("公钥格式无效: %w", err)
	}

	// 检查公钥是否已存在
	var existingKey models.AccessKey
	if err := s.db.Where("public_key = ?", req.PublicKey).First(&existingKey).Error; err == nil {
		return nil, fmt.Errorf("公钥已存在")
	}

	// 检查指纹是否已存在
	if err := s.db.Where("fingerprint = ?", keyInfo.Fingerprint).First(&existingKey).Error; err == nil {
		return nil, fmt.Errorf("相同指纹的密钥已存在")
	}

	// 如果指定了仓库ID，检查仓库是否存在
	if req.RepositoryID != nil {
		var repo models.Repository
		if err := s.db.Where("id = ? AND deleted_at IS NULL", *req.RepositoryID).First(&repo).Error; err != nil {
			return nil, fmt.Errorf("仓库不存在")
		}
	}

	// 创建访问密钥记录
	accessKey := &models.AccessKey{
		RepositoryID: req.RepositoryID,
		UserID:       req.UserID,
		Title:        req.Title,
		PublicKey:    req.PublicKey,
		Fingerprint:  keyInfo.Fingerprint,
		KeyType:      keyInfo.KeyType,
		AccessLevel:  req.AccessLevel,
	}

	if err := s.db.Create(accessKey).Error; err != nil {
		return nil, fmt.Errorf("创建访问密钥失败: %w", err)
	}

	return accessKey, nil
}

// GetByID 根据ID获取访问密钥
func (s *accessKeyService) GetByID(id uuid.UUID) (*models.AccessKey, error) {
	var accessKey models.AccessKey
	if err := s.db.Where("id = ?", id).
		Preload("Repository").
		First(&accessKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("访问密钥不存在")
		}
		return nil, fmt.Errorf("获取访问密钥失败: %w", err)
	}
	return &accessKey, nil
}

// GetByUser 获取用户的所有访问密钥
func (s *accessKeyService) GetByUser(userID uuid.UUID) ([]models.AccessKey, error) {
	var accessKeys []models.AccessKey
	if err := s.db.Where("user_id = ?", userID).
		Preload("Repository").
		Order("created_at DESC").Find(&accessKeys).Error; err != nil {
		return nil, fmt.Errorf("获取用户访问密钥失败: %w", err)
	}
	return accessKeys, nil
}

// GetByRepository 获取仓库的所有访问密钥
func (s *accessKeyService) GetByRepository(repositoryID uuid.UUID) ([]models.AccessKey, error) {
	var accessKeys []models.AccessKey
	if err := s.db.Where("repository_id = ?", repositoryID).
		Order("created_at DESC").Find(&accessKeys).Error; err != nil {
		return nil, fmt.Errorf("获取仓库访问密钥失败: %w", err)
	}
	return accessKeys, nil
}

// GetByFingerprint 根据指纹获取访问密钥
func (s *accessKeyService) GetByFingerprint(fingerprint string) (*models.AccessKey, error) {
	var accessKey models.AccessKey
	if err := s.db.Where("fingerprint = ?", fingerprint).
		Preload("Repository").
		First(&accessKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("访问密钥不存在")
		}
		return nil, fmt.Errorf("获取访问密钥失败: %w", err)
	}
	return &accessKey, nil
}

// Update 更新访问密钥
func (s *accessKeyService) Update(id uuid.UUID, req *UpdateAccessKeyRequest) (*models.AccessKey, error) {
	accessKey, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	
	if req.AccessLevel != nil {
		updates["access_level"] = *req.AccessLevel
	}

	updates["updated_at"] = time.Now()

	if err := s.db.Model(accessKey).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新访问密钥失败: %w", err)
	}

	return s.GetByID(id)
}

// Delete 删除访问密钥
func (s *accessKeyService) Delete(id uuid.UUID) error {
	accessKey, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.db.Delete(accessKey).Error; err != nil {
		return fmt.Errorf("删除访问密钥失败: %w", err)
	}

	return nil
}

// ValidatePublicKey 验证公钥格式并提取信息
func (s *accessKeyService) ValidatePublicKey(publicKeyStr string) (*KeyInfo, error) {
	// 清理公钥字符串
	publicKeyStr = strings.TrimSpace(publicKeyStr)
	
	// 解析SSH公钥
	publicKey, comment, options, rest, err := ssh.ParseAuthorizedKey([]byte(publicKeyStr))
	if err != nil {
		return nil, fmt.Errorf("解析SSH公钥失败: %w", err)
	}

	// 检查是否有剩余字符（表示格式可能不正确）
	if len(rest) > 0 {
		return nil, fmt.Errorf("公钥格式不正确，存在多余字符")
	}

	// 检查选项（通常SSH密钥不应该有限制选项用于Git）
	if len(options) > 0 {
		return nil, fmt.Errorf("不支持带选项的SSH密钥")
	}

	// 获取密钥类型
	keyType := publicKey.Type()

	// 计算指纹
	fingerprint := s.calculateFingerprint(publicKey)

	// 获取密钥长度
	keySize := s.getKeySize(publicKey)

	// 验证密钥强度
	if err := s.validateKeyStrength(keyType, keySize); err != nil {
		return nil, err
	}

	keyInfo := &KeyInfo{
		KeyType:     keyType,
		Fingerprint: fingerprint,
		KeySize:     keySize,
	}

	// 记录注释（如果有）
	_ = comment

	return keyInfo, nil
}

// UpdateLastUsed 更新最后使用时间
func (s *accessKeyService) UpdateLastUsed(id uuid.UUID) error {
	if err := s.db.Model(&models.AccessKey{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error; err != nil {
		return fmt.Errorf("更新最后使用时间失败: %w", err)
	}
	return nil
}

// List 列表查询访问密钥
func (s *accessKeyService) List(req *ListAccessKeysRequest) ([]models.AccessKey, int64, error) {
	query := s.db.Model(&models.AccessKey{})

	// 应用筛选条件
	if req.RepositoryID != nil {
		query = query.Where("repository_id = ?", *req.RepositoryID)
	}
	
	if req.UserID != nil {
		query = query.Where("user_id = ?", *req.UserID)
	}
	
	if req.AccessLevel != nil {
		query = query.Where("access_level = ?", *req.AccessLevel)
	}
	
	if req.KeyType != nil {
		query = query.Where("key_type = ?", *req.KeyType)
	}
	
	if req.Search != nil {
		searchTerm := fmt.Sprintf("%%%s%%", *req.Search)
		query = query.Where("title ILIKE ? OR fingerprint ILIKE ?", searchTerm, searchTerm)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计访问密钥总数失败: %w", err)
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

	var accessKeys []models.AccessKey
	if err := query.Preload("Repository").Find(&accessKeys).Error; err != nil {
		return nil, 0, fmt.Errorf("查询访问密钥列表失败: %w", err)
	}

	return accessKeys, total, nil
}

// calculateFingerprint 计算SSH密钥指纹
func (s *accessKeyService) calculateFingerprint(publicKey ssh.PublicKey) string {
	hash := sha256.Sum256(publicKey.Marshal())
	return "SHA256:" + base64.StdEncoding.EncodeToString(hash[:])
}

// getKeySize 获取密钥长度
func (s *accessKeyService) getKeySize(publicKey ssh.PublicKey) int {
	switch key := publicKey.(type) {
	case *rsa.PublicKey:
		return key.N.BitLen()
	case ssh.CryptoPublicKey:
		if rsaKey, ok := key.CryptoPublicKey().(*rsa.PublicKey); ok {
			return rsaKey.N.BitLen()
		}
	}
	
	// 对于其他类型的密钥（如Ed25519），返回固定长度
	keyType := publicKey.Type()
	switch keyType {
	case "ssh-ed25519":
		return 256
	case "ecdsa-sha2-nistp256":
		return 256
	case "ecdsa-sha2-nistp384":
		return 384
	case "ecdsa-sha2-nistp521":
		return 521
	default:
		return 0
	}
}

// validateKeyStrength 验证密钥强度
func (s *accessKeyService) validateKeyStrength(keyType string, keySize int) error {
	switch keyType {
	case "ssh-rsa":
		if keySize < 2048 {
			return fmt.Errorf("RSA密钥长度至少需要2048位，当前为%d位", keySize)
		}
	case "ssh-ed25519":
		// Ed25519密钥固定256位，始终安全
		return nil
	case "ecdsa-sha2-nistp256":
		if keySize < 256 {
			return fmt.Errorf("ECDSA密钥长度不足")
		}
	case "ecdsa-sha2-nistp384":
		if keySize < 384 {
			return fmt.Errorf("ECDSA密钥长度不足")
		}
	case "ecdsa-sha2-nistp521":
		if keySize < 521 {
			return fmt.Errorf("ECDSA密钥长度不足")
		}
	case "ssh-dss":
		return fmt.Errorf("DSA密钥已不安全，不支持使用")
	default:
		return fmt.Errorf("不支持的密钥类型: %s", keyType)
	}
	
	return nil
}