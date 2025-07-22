package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"iam-service/internal/models"
	"iam-service/pkg/logger"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UserService struct {
	db          *gorm.DB
	redisClient *redis.Client
	logger      logger.Logger
}

func NewUserService(db *gorm.DB, redisClient *redis.Client, logger logger.Logger) *UserService {
	return &UserService{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
	}
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	FullName  *string `json:"full_name,omitempty" binding:"omitempty,min=1"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	MFACode        string `json:"mfa_code,omitempty"`
}

// GetUser 根据ID获取用户
func (s *UserService) GetUser(ctx context.Context, userID uuid.UUID) (*models.UserResponse, error) {
	var user models.User
	if err := s.db.WithContext(ctx).
		Preload("MFASettings").
		Where("id = ? AND status != ?", userID, models.UserStatusSuspended).
		First(&user).Error; err != nil {
		
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		s.logger.Error("Database error getting user", "user_id", userID, "error", err)
		return nil, errors.New("internal server error")
	}

	return user.ToResponse(), nil
}

// GetUsersBulk 批量获取用户
func (s *UserService) GetUsersBulk(ctx context.Context, userIDs []uuid.UUID) ([]*models.UserResponse, error) {
	if len(userIDs) == 0 {
		return []*models.UserResponse{}, nil
	}

	if len(userIDs) > 100 {
		return nil, errors.New("too many user IDs requested, maximum is 100")
	}

	var users []models.User
	if err := s.db.WithContext(ctx).
		Preload("MFASettings").
		Where("id IN ? AND status != ?", userIDs, models.UserStatusSuspended).
		Find(&users).Error; err != nil {
		
		s.logger.Error("Database error getting users bulk", "error", err)
		return nil, errors.New("internal server error")
	}

	responses := make([]*models.UserResponse, len(users))
	for i, user := range users {
		responses[i] = user.ToResponse()
	}

	return responses, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, userID uuid.UUID, req *UpdateUserRequest) (*models.UserResponse, error) {
	var user models.User
	if err := s.db.WithContext(ctx).
		Where("id = ? AND status = ?", userID, models.UserStatusActive).
		First(&user).Error; err != nil {
		
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		s.logger.Error("Database error finding user for update", "user_id", userID, "error", err)
		return nil, errors.New("internal server error")
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	
	if req.FullName != nil {
		updates["full_name"] = *req.FullName
	}
	
	if req.AvatarURL != nil {
		updates["avatar_url"] = *req.AvatarURL
	}

	if len(updates) == 0 {
		// 没有字段需要更新，直接返回当前用户信息
		return s.GetUser(ctx, userID)
	}

	// 执行更新
	if err := s.db.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
		s.logger.Error("Failed to update user", "user_id", userID, "error", err)
		return nil, errors.New("failed to update user")
	}

	s.logger.Info("User updated successfully", "user_id", userID, "updates", updates)
	return s.GetUser(ctx, userID)
}

// ChangePassword 修改密码
func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, req *ChangePasswordRequest) error {
	// 获取用户及认证信息
	var user models.User
	if err := s.db.WithContext(ctx).
		Preload("Authentications").
		Preload("MFASettings").
		Where("id = ? AND status = ?", userID, models.UserStatusActive).
		First(&user).Error; err != nil {
		
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		s.logger.Error("Database error finding user for password change", "user_id", userID, "error", err)
		return errors.New("internal server error")
	}

	// 创建AuthService实例以验证当前密码
	authService := &AuthService{
		db:          s.db,
		redisClient: s.redisClient,
		logger:      s.logger,
	}

	// 验证当前密码
	if err := authService.verifyPassword(&user, req.CurrentPassword); err != nil {
		return errors.New("current password is incorrect")
	}

	// 如果启用了MFA，验证MFA码
	if user.MFASettings != nil && user.MFASettings.IsEnabled {
		if req.MFACode == "" {
			return errors.New("MFA code is required")
		}
		
		if err := authService.verifyMFACode(&user, req.MFACode); err != nil {
			return errors.New("invalid MFA code")
		}
	}

	// 加密新密码
	hashedPassword, err := authService.hashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("Failed to hash new password", "user_id", userID, "error", err)
		return errors.New("internal server error")
	}

	// 开始事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新本地认证信息
	for _, auth := range user.Authentications {
		if auth.Provider == models.AuthProviderLocal && auth.IsActive {
			credentials := map[string]interface{}{
				"password_hash": hashedPassword,
				"updated_at":    time.Now().UTC(),
			}
			credentialsJSON, _ := json.Marshal(credentials)
			
			if err := tx.Model(&auth).Update("credentials", string(credentialsJSON)).Error; err != nil {
				tx.Rollback()
				s.logger.Error("Failed to update password", "user_id", userID, "error", err)
				return errors.New("failed to update password")
			}
			break
		}
	}

	// 撤销所有刷新令牌（强制重新登录）
	if err := tx.Model(&models.RefreshToken{}).
		Where("user_id = ? AND is_revoked = false", userID).
		Update("is_revoked", true).Error; err != nil {
		tx.Rollback()
		s.logger.Error("Failed to revoke refresh tokens", "user_id", userID, "error", err)
		return errors.New("internal server error")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		s.logger.Error("Failed to commit password change transaction", "user_id", userID, "error", err)
		return errors.New("internal server error")
	}

	s.logger.Info("Password changed successfully", "user_id", userID)
	return nil
}

// UploadAvatar 上传头像
func (s *UserService) UploadAvatar(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader) (string, error) {
	// 验证文件类型和大小
	if err := s.validateAvatarFile(file); err != nil {
		return "", err
	}

	// 这里应该实现文件上传逻辑（如上传到S3）
	// 暂时返回一个模拟的URL
	avatarURL := fmt.Sprintf("https://storage.example.com/avatars/%s.jpg", userID.String())

	// 更新用户头像URL
	if err := s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("avatar_url", avatarURL).Error; err != nil {
		
		s.logger.Error("Failed to update avatar URL", "user_id", userID, "error", err)
		return "", errors.New("failed to update avatar")
	}

	s.logger.Info("Avatar uploaded successfully", "user_id", userID, "url", avatarURL)
	return avatarURL, nil
}

// DeleteAccount 删除用户账户（软删除）
func (s *UserService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	// 开始事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 撤销所有刷新令牌
	if err := tx.Model(&models.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true).Error; err != nil {
		tx.Rollback()
		s.logger.Error("Failed to revoke refresh tokens during account deletion", "user_id", userID, "error", err)
		return errors.New("internal server error")
	}

	// 软删除用户
	if err := tx.Where("id = ?", userID).Delete(&models.User{}).Error; err != nil {
		tx.Rollback()
		s.logger.Error("Failed to delete user", "user_id", userID, "error", err)
		return errors.New("failed to delete account")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		s.logger.Error("Failed to commit account deletion transaction", "user_id", userID, "error", err)
		return errors.New("internal server error")
	}

	s.logger.Info("Account deleted successfully", "user_id", userID)
	return nil
}

// validateAvatarFile 验证头像文件
func (s *UserService) validateAvatarFile(file *multipart.FileHeader) error {
	// 检查文件大小（最大5MB）
	if file.Size > 5*1024*1024 {
		return errors.New("file size exceeds maximum allowed size (5MB)")
	}

	// 检查文件类型
	contentType := file.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	if !allowedTypes[contentType] {
		return errors.New("invalid file type, only JPEG, PNG, GIF and WebP are allowed")
	}

	return nil
}