package handlers

import (
	"net/http"
	"strconv"

	"git-gateway-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebhookHandler Webhook处理器
type WebhookHandler struct {
	webhookService services.WebhookService
}

// NewWebhookHandler 创建Webhook处理器
func NewWebhookHandler(webhookService services.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
	}
}

// CreateWebhook 创建Webhook
func (h *WebhookHandler) CreateWebhook(c *gin.Context) {
	var req services.CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	webhook, err := h.webhookService.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Webhook创建成功",
		"data":    webhook,
	})
}

// GetWebhook 获取Webhook详情
func (h *WebhookHandler) GetWebhook(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的Webhook ID"})
		return
	}

	webhook, err := h.webhookService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data":    webhook,
	})
}

// UpdateWebhook 更新Webhook
func (h *WebhookHandler) UpdateWebhook(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的Webhook ID"})
		return
	}

	var req services.UpdateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	webhook, err := h.webhookService.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"data":    webhook,
	})
}

// DeleteWebhook 删除Webhook
func (h *WebhookHandler) DeleteWebhook(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的Webhook ID"})
		return
	}

	if err := h.webhookService.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// ListWebhooks 列表查询Webhook
func (h *WebhookHandler) ListWebhooks(c *gin.Context) {
	var req services.ListWebhooksRequest

	// 解析查询参数
	if repositoryIDParam := c.Query("repository_id"); repositoryIDParam != "" {
		repositoryID, err := uuid.Parse(repositoryIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
			return
		}
		req.RepositoryID = &repositoryID
	}

	if isActiveParam := c.Query("is_active"); isActiveParam != "" {
		isActive := isActiveParam == "true"
		req.IsActive = &isActive
	}

	if eventType := c.Query("event_type"); eventType != "" {
		req.EventType = &eventType
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	req.Page = page
	req.Limit = limit

	// 排序参数
	req.SortBy = c.DefaultQuery("sort_by", "created_at")
	req.SortDesc = c.DefaultQuery("sort_desc", "true") == "true"

	webhooks, total, err := h.webhookService.List(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"webhooks": webhooks,
			"total":    total,
			"page":     page,
			"limit":    limit,
		},
	})
}

// TriggerWebhook 触发Webhook事件（测试用）
func (h *WebhookHandler) TriggerWebhook(c *gin.Context) {
	repositoryIDParam := c.Param("repository_id")
	repositoryID, err := uuid.Parse(repositoryIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	var req struct {
		EventType string      `json:"event_type" binding:"required"`
		Payload   interface{} `json:"payload" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.webhookService.TriggerEvent(repositoryID, req.EventType, req.Payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Webhook事件触发成功",
	})
}