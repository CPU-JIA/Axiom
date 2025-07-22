package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"git-gateway-service/internal/config"
	"git-gateway-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WebhookService Webhook服务接口
type WebhookService interface {
	Create(req *CreateWebhookRequest) (*models.Webhook, error)
	GetByID(id uuid.UUID) (*models.Webhook, error)
	GetByRepository(repositoryID uuid.UUID) ([]models.Webhook, error)
	Update(id uuid.UUID, req *UpdateWebhookRequest) (*models.Webhook, error)
	Delete(id uuid.UUID) error
	TriggerEvent(repositoryID uuid.UUID, eventType string, payload interface{}) error
	DeliverWebhook(webhook *models.Webhook, eventType string, payload interface{}) error
	ProcessDeliveryQueue() error
	List(req *ListWebhooksRequest) ([]models.Webhook, int64, error)
}

type webhookService struct {
	db     *gorm.DB
	config *config.Config
	client *http.Client
}

// NewWebhookService 创建Webhook服务实例
func NewWebhookService(db *gorm.DB, cfg *config.Config) WebhookService {
	return &webhookService{
		db:     db,
		config: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.Webhook.Timeout) * time.Second,
		},
	}
}

// CreateWebhookRequest 创建Webhook请求
type CreateWebhookRequest struct {
	RepositoryID uuid.UUID `json:"repository_id" validate:"required"`
	URL          string    `json:"url" validate:"required,url,max=1024"`
	Secret       *string   `json:"secret" validate:"omitempty,max=255"`
	ContentType  string    `json:"content_type" validate:"required,oneof=application/json application/x-www-form-urlencoded"`
	Events       []string  `json:"events" validate:"required,min=1"`
	IsActive     bool      `json:"is_active"`
	SSLVerify    bool      `json:"ssl_verify"`
}

// UpdateWebhookRequest 更新Webhook请求
type UpdateWebhookRequest struct {
	URL         *string  `json:"url" validate:"omitempty,url,max=1024"`
	Secret      *string  `json:"secret" validate:"omitempty,max=255"`
	ContentType *string  `json:"content_type" validate:"omitempty,oneof=application/json application/x-www-form-urlencoded"`
	Events      []string `json:"events" validate:"omitempty,min=1"`
	IsActive    *bool    `json:"is_active"`
	SSLVerify   *bool    `json:"ssl_verify"`
}

// ListWebhooksRequest 列表查询请求
type ListWebhooksRequest struct {
	RepositoryID *uuid.UUID `json:"repository_id"`
	IsActive     *bool      `json:"is_active"`
	EventType    *string    `json:"event_type"`
	Page         int        `json:"page"`
	Limit        int        `json:"limit"`
	SortBy       string     `json:"sort_by"`
	SortDesc     bool       `json:"sort_desc"`
}

// WebhookEvent Webhook事件定义
const (
	EventTypePush         = "push"
	EventTypeTagCreate    = "tag_create"
	EventTypeTagDelete    = "tag_delete"
	EventTypeBranchCreate = "branch_create"
	EventTypeBranchDelete = "branch_delete"
	EventTypePullRequest  = "pull_request"
	EventTypeIssue        = "issue"
	EventTypeRelease      = "release"
)

// Create 创建Webhook
func (s *webhookService) Create(req *CreateWebhookRequest) (*models.Webhook, error) {
	// 检查仓库是否存在
	var repo models.Repository
	if err := s.db.Where("id = ? AND deleted_at IS NULL", req.RepositoryID).First(&repo).Error; err != nil {
		return nil, fmt.Errorf("仓库不存在")
	}

	// 验证事件类型
	validEvents := map[string]bool{
		EventTypePush:         true,
		EventTypeTagCreate:    true,
		EventTypeTagDelete:    true,
		EventTypeBranchCreate: true,
		EventTypeBranchDelete: true,
		EventTypePullRequest:  true,
		EventTypeIssue:        true,
		EventTypeRelease:      true,
	}

	for _, event := range req.Events {
		if !validEvents[event] {
			return nil, fmt.Errorf("无效的事件类型: %s", event)
		}
	}

	// 创建Webhook记录
	eventsJSON, _ := json.Marshal(req.Events)
	webhook := &models.Webhook{
		RepositoryID: req.RepositoryID,
		URL:          req.URL,
		Secret:       req.Secret,
		ContentType:  req.ContentType,
		Events:       eventsJSON,
		IsActive:     req.IsActive,
		SSLVerify:    req.SSLVerify,
	}

	if err := s.db.Create(webhook).Error; err != nil {
		return nil, fmt.Errorf("创建Webhook失败: %w", err)
	}

	return webhook, nil
}

// GetByID 根据ID获取Webhook
func (s *webhookService) GetByID(id uuid.UUID) (*models.Webhook, error) {
	var webhook models.Webhook
	if err := s.db.Where("id = ?", id).
		Preload("Repository").
		Preload("Deliveries", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(10)
		}).
		First(&webhook).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("Webhook不存在")
		}
		return nil, fmt.Errorf("获取Webhook失败: %w", err)
	}
	return &webhook, nil
}

// GetByRepository 获取仓库的所有Webhook
func (s *webhookService) GetByRepository(repositoryID uuid.UUID) ([]models.Webhook, error) {
	var webhooks []models.Webhook
	if err := s.db.Where("repository_id = ?", repositoryID).
		Order("created_at DESC").Find(&webhooks).Error; err != nil {
		return nil, fmt.Errorf("获取仓库Webhook失败: %w", err)
	}
	return webhooks, nil
}

// Update 更新Webhook
func (s *webhookService) Update(id uuid.UUID, req *UpdateWebhookRequest) (*models.Webhook, error) {
	webhook, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	
	if req.URL != nil {
		updates["url"] = *req.URL
	}
	
	if req.Secret != nil {
		updates["secret"] = *req.Secret
	}
	
	if req.ContentType != nil {
		updates["content_type"] = *req.ContentType
	}
	
	if req.Events != nil {
		// 验证事件类型
		validEvents := map[string]bool{
			EventTypePush:         true,
			EventTypeTagCreate:    true,
			EventTypeTagDelete:    true,
			EventTypeBranchCreate: true,
			EventTypeBranchDelete: true,
			EventTypePullRequest:  true,
			EventTypeIssue:        true,
			EventTypeRelease:      true,
		}

		for _, event := range req.Events {
			if !validEvents[event] {
				return nil, fmt.Errorf("无效的事件类型: %s", event)
			}
		}

		eventsJSON, _ := json.Marshal(req.Events)
		updates["events"] = eventsJSON
	}
	
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	
	if req.SSLVerify != nil {
		updates["ssl_verify"] = *req.SSLVerify
	}

	updates["updated_at"] = time.Now()

	if err := s.db.Model(webhook).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新Webhook失败: %w", err)
	}

	return s.GetByID(id)
}

// Delete 删除Webhook
func (s *webhookService) Delete(id uuid.UUID) error {
	webhook, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.db.Delete(webhook).Error; err != nil {
		return fmt.Errorf("删除Webhook失败: %w", err)
	}

	return nil
}

// TriggerEvent 触发事件，向所有订阅的Webhook发送
func (s *webhookService) TriggerEvent(repositoryID uuid.UUID, eventType string, payload interface{}) error {
	// 获取仓库的所有活跃Webhook
	webhooks, err := s.GetByRepository(repositoryID)
	if err != nil {
		return err
	}

	// 过滤订阅该事件的Webhook
	var targetWebhooks []models.Webhook
	for _, webhook := range webhooks {
		if !webhook.IsActive {
			continue
		}

		var events []string
		if err := json.Unmarshal(webhook.Events, &events); err != nil {
			continue
		}

		for _, event := range events {
			if event == eventType {
				targetWebhooks = append(targetWebhooks, webhook)
				break
			}
		}
	}

	// 异步发送Webhook
	for _, webhook := range targetWebhooks {
		go func(w models.Webhook) {
			if err := s.DeliverWebhook(&w, eventType, payload); err != nil {
				// 记录错误日志
				fmt.Printf("Webhook投递失败 [%s]: %v\n", w.URL, err)
			}
		}(webhook)
	}

	return nil
}

// DeliverWebhook 投递Webhook
func (s *webhookService) DeliverWebhook(webhook *models.Webhook, eventType string, payload interface{}) error {
	deliveryID := uuid.New().String()[:8]
	
	// 创建投递记录
	delivery := &models.WebhookDelivery{
		WebhookID:     webhook.ID,
		EventType:     eventType,
		DeliveryID:    deliveryID,
		AttemptCount:  1,
		LastAttemptAt: time.Now(),
	}

	// 序列化payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		delivery.Success = false
		s.db.Create(delivery)
		return fmt.Errorf("序列化payload失败: %w", err)
	}
	delivery.RequestBody = payloadBytes

	// 创建HTTP请求
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		delivery.Success = false
		s.db.Create(delivery)
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	headers := map[string]string{
		"Content-Type":     webhook.ContentType,
		"User-Agent":       "Git-Gateway-Webhook/1.0",
		"X-Event-Type":     eventType,
		"X-Delivery-ID":    deliveryID,
		"X-Request-ID":     uuid.New().String(),
	}

	// 添加签名（如果配置了secret）
	if webhook.Secret != nil && *webhook.Secret != "" {
		signature := s.generateSignature(payloadBytes, *webhook.Secret)
		headers["X-Hub-Signature-256"] = signature
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 记录请求头
	requestHeaders, _ := json.Marshal(headers)
	delivery.RequestHeaders = requestHeaders

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		delivery.Success = false
		s.db.Create(delivery)
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 记录响应
	delivery.ResponseStatus = &resp.StatusCode
	
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}
	responseHeadersJSON, _ := json.Marshal(responseHeaders)
	delivery.ResponseHeaders = responseHeadersJSON

	// 读取响应体（限制大小）
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	if n > 0 {
		responseBody := string(buf[:n])
		delivery.ResponseBody = &responseBody
	}

	// 判断是否成功
	delivery.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	// 更新Webhook状态
	updates := map[string]interface{}{
		"last_delivery": time.Now(),
		"updated_at":    time.Now(),
	}

	if delivery.Success {
		updates["last_status"] = "success"
		updates["last_error"] = nil
	} else {
		updates["last_status"] = "failed"
		errorMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		updates["last_error"] = errorMsg
		
		// 如果失败且配置了重试，安排重试
		if s.config.Webhook.MaxRetries > 1 {
			nextAttempt := time.Now().Add(time.Duration(s.config.Webhook.RetryInterval) * time.Second)
			delivery.NextAttemptAt = &nextAttempt
		}
	}

	// 保存投递记录
	s.db.Create(delivery)

	// 更新Webhook记录
	s.db.Model(webhook).Updates(updates)

	if !delivery.Success {
		return fmt.Errorf("Webhook投递失败: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ProcessDeliveryQueue 处理待重试的投递队列
func (s *webhookService) ProcessDeliveryQueue() error {
	// 查找需要重试的投递记录
	var deliveries []models.WebhookDelivery
	if err := s.db.Where("success = ? AND next_attempt_at <= ? AND attempt_count < ?", 
		false, time.Now(), s.config.Webhook.MaxRetries).
		Preload("Webhook").Find(&deliveries).Error; err != nil {
		return fmt.Errorf("查找重试队列失败: %w", err)
	}

	for _, delivery := range deliveries {
		// 增加重试次数
		delivery.AttemptCount++
		delivery.LastAttemptAt = time.Now()

		// 重新投递
		var payload interface{}
		if err := json.Unmarshal(delivery.RequestBody, &payload); err != nil {
			continue
		}

		if err := s.DeliverWebhook(delivery.Webhook, delivery.EventType, payload); err != nil {
			// 投递仍然失败，更新重试时间
			if delivery.AttemptCount < s.config.Webhook.MaxRetries {
				nextAttempt := time.Now().Add(time.Duration(s.config.Webhook.RetryInterval*delivery.AttemptCount) * time.Second)
				delivery.NextAttemptAt = &nextAttempt
			} else {
				delivery.NextAttemptAt = nil // 停止重试
			}
			s.db.Save(&delivery)
		}
	}

	return nil
}

// List 列表查询Webhook
func (s *webhookService) List(req *ListWebhooksRequest) ([]models.Webhook, int64, error) {
	query := s.db.Model(&models.Webhook{})

	// 应用筛选条件
	if req.RepositoryID != nil {
		query = query.Where("repository_id = ?", *req.RepositoryID)
	}
	
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	if req.EventType != nil {
		query = query.Where("events::text LIKE ?", fmt.Sprintf("%%%s%%", *req.EventType))
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计Webhook总数失败: %w", err)
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

	var webhooks []models.Webhook
	if err := query.Preload("Repository").Find(&webhooks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询Webhook列表失败: %w", err)
	}

	return webhooks, total, nil
}

// generateSignature 生成Webhook签名
func (s *webhookService) generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))
	return "sha256=" + signature
}