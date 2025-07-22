package handlers

import (
	"net/http"
	"strconv"

	"git-gateway-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AccessKeyHandler 访问密钥处理器
type AccessKeyHandler struct {
	accessKeyService services.AccessKeyService
}

// NewAccessKeyHandler 创建访问密钥处理器
func NewAccessKeyHandler(accessKeyService services.AccessKeyService) *AccessKeyHandler {
	return &AccessKeyHandler{
		accessKeyService: accessKeyService,
	}
}

// CreateAccessKey 创建访问密钥
func (h *AccessKeyHandler) CreateAccessKey(c *gin.Context) {
	var req services.CreateAccessKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessKey, err := h.accessKeyService.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "访问密钥创建成功",
		"data":    accessKey,
	})
}

// GetAccessKey 获取访问密钥详情
func (h *AccessKeyHandler) GetAccessKey(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的访问密钥ID"})
		return
	}

	accessKey, err := h.accessKeyService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data":    accessKey,
	})
}

// UpdateAccessKey 更新访问密钥
func (h *AccessKeyHandler) UpdateAccessKey(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的访问密钥ID"})
		return
	}

	var req services.UpdateAccessKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessKey, err := h.accessKeyService.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"data":    accessKey,
	})
}

// DeleteAccessKey 删除访问密钥
func (h *AccessKeyHandler) DeleteAccessKey(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的访问密钥ID"})
		return
	}

	if err := h.accessKeyService.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// ListAccessKeys 列表查询访问密钥
func (h *AccessKeyHandler) ListAccessKeys(c *gin.Context) {
	var req services.ListAccessKeysRequest

	// 解析查询参数
	if repositoryIDParam := c.Query("repository_id"); repositoryIDParam != "" {
		repositoryID, err := uuid.Parse(repositoryIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
			return
		}
		req.RepositoryID = &repositoryID
	}

	if userIDParam := c.Query("user_id"); userIDParam != "" {
		userID, err := uuid.Parse(userIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
			return
		}
		req.UserID = &userID
	}

	if accessLevel := c.Query("access_level"); accessLevel != "" {
		req.AccessLevel = &accessLevel
	}

	if keyType := c.Query("key_type"); keyType != "" {
		req.KeyType = &keyType
	}

	if search := c.Query("search"); search != "" {
		req.Search = &search
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	req.Page = page
	req.Limit = limit

	// 排序参数
	req.SortBy = c.DefaultQuery("sort_by", "created_at")
	req.SortDesc = c.DefaultQuery("sort_desc", "true") == "true"

	accessKeys, total, err := h.accessKeyService.List(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"access_keys": accessKeys,
			"total":       total,
			"page":        page,
			"limit":       limit,
		},
	})
}

// ValidatePublicKey 验证公钥
func (h *AccessKeyHandler) ValidatePublicKey(c *gin.Context) {
	var req struct {
		PublicKey string `json:"public_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	keyInfo, err := h.accessKeyService.ValidatePublicKey(req.PublicKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "公钥验证成功",
		"data":    keyInfo,
	})
}