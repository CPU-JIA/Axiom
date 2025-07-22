package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"tenant-service/internal/models"
	"tenant-service/pkg/logger"
	"tenant-service/pkg/utils"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type TenantService struct {
	db          *gorm.DB
	redisClient *redis.Client
	logger      logger.Logger
}

func NewTenantService(db *gorm.DB, redisClient *redis.Client, logger logger.Logger) *TenantService {
	return &TenantService{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
	}
}

// CreateTenantRequest 创建租户请求
type CreateTenantRequest struct {
	Name        string `json:"name" binding:"required,min=3,max=50"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	PlanType    string `json:"plan_type,omitempty"`
}

// UpdateTenantRequest 更新租户请求
type UpdateTenantRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	Description *string `json:"description,omitempty"`
	LogoURL     *string `json:"logo_url,omitempty"`
}

// CreateTenant 创建租户
func (s *TenantService) CreateTenant(ctx context.Context, ownerID uuid.UUID, req *CreateTenantRequest) (*models.TenantResponse, error) {
	// 验证租户名称
	if !utils.IsValidTenantName(req.Name) {
		return nil, errors.New("invalid tenant name format")
	}

	// 生成slug
	slug := utils.GenerateSlug(req.Name)

	// 检查名称和slug是否已存在
	var existingTenant models.Tenant
	err := s.db.WithContext(ctx).Where("name = ? OR slug = ?", req.Name, slug).First(&existingTenant).Error
	if err == nil {
		return nil, errors.New("tenant name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("Database error checking tenant existence", "error", err)
		return nil, errors.New("internal server error")
	}

	// 设置计划类型
	planType := models.PlanType(req.PlanType)
	if planType == "" {
		planType = models.PlanTypeFree
	}

	// 开始事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建租户
	tenant := &models.Tenant{
		Name:        req.Name,
		Slug:        slug,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Status:      models.TenantStatusActive,
		PlanType:    planType,
		MaxMembers:  s.getDefaultMaxMembers(planType),
		MaxProjects: s.getDefaultMaxProjects(planType),
		StorageQuota: s.getDefaultStorageQuota(planType),
		ActivatedAt: &time.Time{},
	}
	*tenant.ActivatedAt = time.Now().UTC()

	if err := tx.Create(tenant).Error; err != nil {
		tx.Rollback()
		s.logger.Error("Failed to create tenant", "error", err)
		return nil, errors.New("failed to create tenant")
	}

	// 添加创建者为所有者
	member := &models.TenantMember{
		TenantID:    tenant.ID,
		UserID:      ownerID,
		Role:        models.MemberRoleOwner,
		Status:      models.MemberStatusActive,
		JoinedAt:    time.Now().UTC(),
		ActivatedAt: tenant.ActivatedAt,
	}

	if err := tx.Create(member).Error; err != nil {
		tx.Rollback()
		s.logger.Error("Failed to create tenant owner", "tenant_id", tenant.ID, "user_id", ownerID, "error", err)
		return nil, errors.New("failed to create tenant owner")
	}

	// 记录审计日志
	auditLog := &models.TenantAuditLog{
		TenantID:     tenant.ID,
		UserID:       &ownerID,
		Action:       "tenant.created",
		ResourceType: "tenant",
		ResourceID:   &tenant.ID,
		Details:      fmt.Sprintf(`{"name":"%s","plan_type":"%s"}`, tenant.Name, tenant.PlanType),
		IPAddress:    "system",
		UserAgent:    "tenant-service",
		Status:       "success",
	}

	if err := tx.Create(auditLog).Error; err != nil {
		// 审计日志失败不影响主流程
		s.logger.Warn("Failed to create audit log", "error", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		s.logger.Error("Failed to commit tenant creation transaction", "error", err)
		return nil, errors.New("internal server error")
	}

	s.logger.Info("Tenant created successfully", "tenant_id", tenant.ID, "owner_id", ownerID)

	return tenant.ToResponse(), nil
}

// GetTenant 获取租户信息
func (s *TenantService) GetTenant(ctx context.Context, tenantID uuid.UUID) (*models.TenantResponse, error) {
	var tenant models.Tenant
	err := s.db.WithContext(ctx).
		Where("id = ? AND status != ?", tenantID, models.TenantStatusInactive).
		First(&tenant).Error
		
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tenant not found")
		}
		s.logger.Error("Database error getting tenant", "tenant_id", tenantID, "error", err)
		return nil, errors.New("internal server error")
	}

	response := tenant.ToResponse()
	
	// 获取统计信息
	s.loadTenantStats(ctx, response)
	
	return response, nil
}

// UpdateTenant 更新租户信息
func (s *TenantService) UpdateTenant(ctx context.Context, tenantID uuid.UUID, req *UpdateTenantRequest) (*models.TenantResponse, error) {
	var tenant models.Tenant
	err := s.db.WithContext(ctx).
		Where("id = ? AND status = ?", tenantID, models.TenantStatusActive).
		First(&tenant).Error
		
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tenant not found")
		}
		s.logger.Error("Database error finding tenant for update", "tenant_id", tenantID, "error", err)
		return nil, errors.New("internal server error")
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	
	if req.LogoURL != nil {
		updates["logo_url"] = *req.LogoURL
	}

	if len(updates) == 0 {
		// 没有字段需要更新
		return s.GetTenant(ctx, tenantID)
	}

	// 执行更新
	if err := s.db.WithContext(ctx).Model(&tenant).Updates(updates).Error; err != nil {
		s.logger.Error("Failed to update tenant", "tenant_id", tenantID, "error", err)
		return nil, errors.New("failed to update tenant")
	}

	s.logger.Info("Tenant updated successfully", "tenant_id", tenantID, "updates", updates)
	
	return s.GetTenant(ctx, tenantID)
}

// GetMember 获取租户成员信息
func (s *TenantService) GetMember(ctx context.Context, tenantID, userID uuid.UUID) (*models.TenantMember, error) {
	var member models.TenantMember
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND status = ?", tenantID, userID, models.MemberStatusActive).
		First(&member).Error
		
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("member not found")
		}
		s.logger.Error("Database error getting member", "tenant_id", tenantID, "user_id", userID, "error", err)
		return nil, errors.New("internal server error")
	}

	return &member, nil
}

// ListMembers 获取租户成员列表
func (s *TenantService) ListMembers(ctx context.Context, tenantID uuid.UUID, page, size int) ([]*models.TenantMemberResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size

	var members []models.TenantMember
	var total int64

	// 获取总数
	if err := s.db.WithContext(ctx).Model(&models.TenantMember{}).
		Where("tenant_id = ?", tenantID).
		Count(&total).Error; err != nil {
		s.logger.Error("Failed to count tenant members", "tenant_id", tenantID, "error", err)
		return nil, 0, errors.New("internal server error")
	}

	// 获取成员列表
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("role DESC, joined_at ASC").
		Offset(offset).
		Limit(size).
		Find(&members).Error; err != nil {
		s.logger.Error("Failed to list tenant members", "tenant_id", tenantID, "error", err)
		return nil, 0, errors.New("internal server error")
	}

	responses := make([]*models.TenantMemberResponse, len(members))
	for i, member := range members {
		responses[i] = member.ToResponse()
		// 这里可以调用IAM服务获取用户详细信息
	}

	return responses, total, nil
}

// InviteMember 邀请成员
func (s *TenantService) InviteMember(ctx context.Context, tenantID, inviterID uuid.UUID, email string, role models.MemberRole) (*models.TenantInvitation, error) {
	// 验证角色
	if !s.isValidRole(role) {
		return nil, errors.New("invalid role")
	}

	// 检查是否已经是成员
	var existingMember models.TenantMember
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id IN (SELECT id FROM users WHERE email = ?)", tenantID, email).
		First(&existingMember).Error
		
	if err == nil {
		return nil, errors.New("user is already a member")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("Database error checking member existence", "error", err)
		return nil, errors.New("internal server error")
	}

	// 检查是否已有待处理的邀请
	var existingInvite models.TenantInvitation
	err = s.db.WithContext(ctx).
		Where("tenant_id = ? AND email = ? AND status = ?", tenantID, email, models.InviteStatusPending).
		First(&existingInvite).Error
		
	if err == nil {
		return nil, errors.New("invitation already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("Database error checking invitation existence", "error", err)
		return nil, errors.New("internal server error")
	}

	// 生成邀请令牌
	token, err := s.generateInviteToken()
	if err != nil {
		s.logger.Error("Failed to generate invite token", "error", err)
		return nil, errors.New("internal server error")
	}

	// 创建邀请
	invitation := &models.TenantInvitation{
		TenantID:  tenantID,
		InviterID: inviterID,
		Email:     email,
		Role:      role,
		Status:    models.InviteStatusPending,
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour), // 7天过期
	}

	if err := s.db.WithContext(ctx).Create(invitation).Error; err != nil {
		s.logger.Error("Failed to create invitation", "error", err)
		return nil, errors.New("failed to create invitation")
	}

	s.logger.Info("Member invited successfully", "tenant_id", tenantID, "email", email, "inviter_id", inviterID)

	return invitation, nil
}

// 辅助方法

func (s *TenantService) getDefaultMaxMembers(planType models.PlanType) int {
	switch planType {
	case models.PlanTypeFree:
		return 5
	case models.PlanTypeStarter:
		return 20
	case models.PlanTypePro:
		return 100
	case models.PlanTypeEnterprise:
		return 1000
	default:
		return 5
	}
}

func (s *TenantService) getDefaultMaxProjects(planType models.PlanType) int {
	switch planType {
	case models.PlanTypeFree:
		return 3
	case models.PlanTypeStarter:
		return 10
	case models.PlanTypePro:
		return 50
	case models.PlanTypeEnterprise:
		return 500
	default:
		return 3
	}
}

func (s *TenantService) getDefaultStorageQuota(planType models.PlanType) int64 {
	switch planType {
	case models.PlanTypeFree:
		return 1 * 1024 * 1024 * 1024 // 1GB
	case models.PlanTypeStarter:
		return 10 * 1024 * 1024 * 1024 // 10GB
	case models.PlanTypePro:
		return 100 * 1024 * 1024 * 1024 // 100GB
	case models.PlanTypeEnterprise:
		return 1000 * 1024 * 1024 * 1024 // 1TB
	default:
		return 1 * 1024 * 1024 * 1024
	}
}

func (s *TenantService) isValidRole(role models.MemberRole) bool {
	switch role {
	case models.MemberRoleOwner, models.MemberRoleAdmin, models.MemberRoleMaintainer, 
		 models.MemberRoleDeveloper, models.MemberRoleGuest:
		return true
	default:
		return false
	}
}

func (s *TenantService) generateInviteToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *TenantService) loadTenantStats(ctx context.Context, tenant *models.TenantResponse) {
	// 获取成员数量
	var memberCount int64
	if err := s.db.WithContext(ctx).Model(&models.TenantMember{}).
		Where("tenant_id = ? AND status = ?", tenant.ID, models.MemberStatusActive).
		Count(&memberCount).Error; err == nil {
		tenant.MemberCount = int(memberCount)
	}

	// 项目数量暂时设为0，等项目服务实现后再获取
	tenant.ProjectCount = 0
}