package services

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"iam-service/internal/database"
	"iam-service/internal/models"
	"iam-service/pkg/logger"
	"iam-service/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db          *gorm.DB
	redisClient *redis.Client
	jwtSecret   string
	logger      logger.Logger
}

func NewAuthService(db *gorm.DB, redisClient *redis.Client, jwtSecret string, logger logger.Logger) *AuthService {
	return &AuthService{
		db:          db,
		redisClient: redisClient,
		jwtSecret:   jwtSecret,
		logger:      logger,
	}
}

// JWTClaims JWT载荷
type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty"`
	Role     string    `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	MFACode    string `json:"mfa_code,omitempty"`
	RememberMe bool   `json:"remember_me"`
	IPAddress  string `json:"-"`
	UserAgent  string `json:"-"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
	ExpiresIn    int                 `json:"expires_in"`
	TokenType    string              `json:"token_type"`
	User         *models.UserResponse `json:"user"`
	MFARequired  bool                `json:"mfa_required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FullName  string `json:"full_name" binding:"required,min=1"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// 检查账户锁定
	if locked, err := s.isAccountLocked(ctx, req.Email); err != nil {
		s.logger.Error("Failed to check account lockout", "email", req.Email, "error", err)
		return nil, errors.New("internal server error")
	} else if locked {
		s.logger.Warn("Account locked due to too many failed attempts", "email", req.Email)
		return nil, errors.New("account temporarily locked due to too many failed login attempts")
	}

	// 查找用户
	var user models.User
	if err := s.db.WithContext(ctx).
		Preload("Authentications").
		Preload("MFASettings").
		Where("email = ? AND status = ?", req.Email, models.UserStatusActive).
		First(&user).Error; err != nil {
		
		// 记录失败的登录尝试
		s.recordLoginAttempt(ctx, nil, req.Email, req.IPAddress, req.UserAgent, false, "user not found")
		
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		s.logger.Error("Database error during login", "error", err)
		return nil, errors.New("internal server error")
	}

	// 验证密码
	if err := s.verifyPassword(&user, req.Password); err != nil {
		s.recordLoginAttempt(ctx, &user.ID, req.Email, req.IPAddress, req.UserAgent, false, "invalid password")
		return nil, errors.New("invalid credentials")
	}

	// 检查是否需要MFA
	if user.MFASettings != nil && user.MFASettings.IsEnabled {
		if req.MFACode == "" {
			return &LoginResponse{
				MFARequired: true,
			}, nil
		}

		// 验证MFA码
		if err := s.verifyMFACode(&user, req.MFACode); err != nil {
			s.recordLoginAttempt(ctx, &user.ID, req.Email, req.IPAddress, req.UserAgent, false, "invalid MFA code")
			return nil, errors.New("invalid MFA code")
		}
	}

	// 生成JWT Token
	accessToken, refreshToken, expiresIn, err := s.generateTokens(ctx, &user, req.RememberMe)
	if err != nil {
		s.logger.Error("Failed to generate tokens", "user_id", user.ID, "error", err)
		return nil, errors.New("failed to generate access token")
	}

	// 更新最后登录时间
	now := time.Now().UTC()
	user.LastLoginAt = &now
	s.db.WithContext(ctx).Model(&user).Update("last_login_at", now)

	// 记录成功的登录尝试
	s.recordLoginAttempt(ctx, &user.ID, req.Email, req.IPAddress, req.UserAgent, true, "")

	// 清除登录失败记录
	s.clearLoginAttempts(ctx, req.Email)

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		User:         user.ToResponse(),
		MFARequired:  false,
	}, nil
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*models.UserResponse, error) {
	// 检查邮箱是否已存在
	var existingUser models.User
	if err := s.db.WithContext(ctx).Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, errors.New("email already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("Database error during registration check", "error", err)
		return nil, errors.New("internal server error")
	}

	// 开始数据库事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建用户
	user := &models.User{
		Email:        req.Email,
		FullName:     req.FullName,
		Status:       models.UserStatusPending, // 待验证状态
		EmailVerified: false,
	}

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		s.logger.Error("Failed to create user", "email", req.Email, "error", err)
		return nil, errors.New("failed to create user")
	}

	// 创建本地认证信息
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		tx.Rollback()
		s.logger.Error("Failed to hash password", "error", err)
		return nil, errors.New("internal server error")
	}

	credentials := map[string]interface{}{
		"password_hash": hashedPassword,
		"created_at":    time.Now().UTC(),
	}
	credentialsJSON, _ := json.Marshal(credentials)

	auth := &models.UserAuthentication{
		UserID:       user.ID,
		Provider:     models.AuthProviderLocal,
		Credentials:  string(credentialsJSON),
		IsActive:     true,
	}

	if err := tx.Create(auth).Error; err != nil {
		tx.Rollback()
		s.logger.Error("Failed to create user authentication", "user_id", user.ID, "error", err)
		return nil, errors.New("failed to create user authentication")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		s.logger.Error("Failed to commit registration transaction", "error", err)
		return nil, errors.New("internal server error")
	}

	// 发送验证邮件（异步）
	go s.sendVerificationEmail(user.ID, user.Email, user.FullName)

	s.logger.Info("User registered successfully", "user_id", user.ID, "email", user.Email)
	return user.ToResponse(), nil
}

// RefreshToken 刷新访问令牌
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// 查找并验证刷新令牌
	tokenHash := utils.HashString(refreshToken)
	
	var tokenRecord models.RefreshToken
	if err := s.db.WithContext(ctx).
		Preload("User").
		Preload("User.MFASettings").
		Where("token_hash = ? AND is_revoked = false AND expires_at > ?", tokenHash, time.Now().UTC()).
		First(&tokenRecord).Error; err != nil {
		
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid refresh token")
		}
		s.logger.Error("Database error during token refresh", "error", err)
		return nil, errors.New("internal server error")
	}

	// 检查用户状态
	if tokenRecord.User.Status != models.UserStatusActive {
		return nil, errors.New("user account is not active")
	}

	// 撤销旧的刷新令牌
	s.db.WithContext(ctx).Model(&tokenRecord).Update("is_revoked", true)

	// 生成新的令牌
	accessToken, newRefreshToken, expiresIn, err := s.generateTokens(ctx, &tokenRecord.User, false)
	if err != nil {
		s.logger.Error("Failed to generate tokens during refresh", "user_id", tokenRecord.User.ID, "error", err)
		return nil, errors.New("failed to generate access token")
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		User:         tokenRecord.User.ToResponse(),
	}, nil
}

// 生成JWT和刷新令牌
func (s *AuthService) generateTokens(ctx context.Context, user *models.User, rememberMe bool) (string, string, int, error) {
	// 生成访问令牌
	expiresIn := 60 * 60 // 1小时
	claims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresIn) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "cloud-platform-iam",
			Subject:   user.ID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to sign access token: %w", err)
	}

	// 生成刷新令牌
	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return "", "", 0, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshToken := base32.StdEncoding.EncodeToString(refreshTokenBytes)

	// 设置刷新令牌过期时间
	var refreshExpiry time.Time
	if rememberMe {
		refreshExpiry = time.Now().AddDate(0, 0, 30) // 30天
	} else {
		refreshExpiry = time.Now().AddDate(0, 0, 7)  // 7天
	}

	// 保存刷新令牌到数据库
	tokenRecord := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: utils.HashString(refreshToken),
		ExpiresAt: refreshExpiry,
	}

	if err := s.db.WithContext(ctx).Create(tokenRecord).Error; err != nil {
		return "", "", 0, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return accessToken, refreshToken, expiresIn, nil
}

// 验证密码
func (s *AuthService) verifyPassword(user *models.User, password string) error {
	for _, auth := range user.Authentications {
		if auth.Provider == models.AuthProviderLocal && auth.IsActive {
			var credentials map[string]interface{}
			if err := json.Unmarshal([]byte(auth.Credentials), &credentials); err != nil {
				continue
			}

			if passwordHash, ok := credentials["password_hash"].(string); ok {
				if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err == nil {
					return nil
				}
			}
		}
	}
	return errors.New("invalid password")
}

// 加密密码
func (s *AuthService) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// 验证MFA代码
func (s *AuthService) verifyMFACode(user *models.User, code string) error {
	if user.MFASettings == nil || !user.MFASettings.IsEnabled {
		return errors.New("MFA not enabled")
	}

	// 这里应该实现TOTP验证逻辑
	// 暂时简化处理
	if len(code) != 6 {
		return errors.New("invalid MFA code format")
	}

	return nil
}

// 记录登录尝试
func (s *AuthService) recordLoginAttempt(ctx context.Context, userID *uuid.UUID, email, ip, userAgent string, success bool, failReason string) {
	attempt := &models.LoginAttempt{
		UserID:     userID,
		Email:      email,
		IPAddress:  ip,
		UserAgent:  userAgent,
		Success:    success,
		FailReason: failReason,
	}

	if err := s.db.WithContext(ctx).Create(attempt).Error; err != nil {
		s.logger.Error("Failed to record login attempt", "error", err)
	}
}

// 检查账户是否被锁定
func (s *AuthService) isAccountLocked(ctx context.Context, email string) (bool, error) {
	key := database.BuildKey(database.KeyUserLockout, email)
	
	result := s.redisClient.Get(ctx, key)
	if errors.Is(result.Err(), redis.Nil) {
		return false, nil
	}
	
	return result.Err() == nil, result.Err()
}

// 清除登录失败记录
func (s *AuthService) clearLoginAttempts(ctx context.Context, email string) {
	key := database.BuildKey(database.KeyUserLockout, email)
	s.redisClient.Del(ctx, key)
}

// 发送验证邮件
func (s *AuthService) sendVerificationEmail(userID uuid.UUID, email, fullName string) {
	// 这里应该实现邮件发送逻辑
	s.logger.Info("Sending verification email", "user_id", userID, "email", email)
}