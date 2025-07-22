package handlers

import (
	"net/http"
	"strconv"

	"git-gateway-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RepositoryHandler 仓库处理器
type RepositoryHandler struct {
	repoService services.RepositoryService
}

// NewRepositoryHandler 创建仓库处理器
func NewRepositoryHandler(repoService services.RepositoryService) *RepositoryHandler {
	return &RepositoryHandler{
		repoService: repoService,
	}
}

// CreateRepository 创建仓库
func (h *RepositoryHandler) CreateRepository(c *gin.Context) {
	var req services.CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo, err := h.repoService.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "仓库创建成功",
		"data":    repo,
	})
}

// GetRepository 获取仓库详情
func (h *RepositoryHandler) GetRepository(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	repo, err := h.repoService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data":    repo,
	})
}

// GetRepositoryByName 根据项目ID和名称获取仓库
func (h *RepositoryHandler) GetRepositoryByName(c *gin.Context) {
	projectIDParam := c.Param("project_id")
	projectID, err := uuid.Parse(projectIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仓库名称不能为空"})
		return
	}

	repo, err := h.repoService.GetByName(projectID, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data":    repo,
	})
}

// UpdateRepository 更新仓库
func (h *RepositoryHandler) UpdateRepository(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	var req services.UpdateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo, err := h.repoService.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"data":    repo,
	})
}

// DeleteRepository 删除仓库
func (h *RepositoryHandler) DeleteRepository(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	if err := h.repoService.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// ListRepositories 列表查询仓库
func (h *RepositoryHandler) ListRepositories(c *gin.Context) {
	var req services.ListRepositoriesRequest

	// 解析查询参数
	if projectIDParam := c.Query("project_id"); projectIDParam != "" {
		projectID, err := uuid.Parse(projectIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
			return
		}
		req.ProjectID = &projectID
	}

	if visibility := c.Query("visibility"); visibility != "" {
		req.Visibility = &visibility
	}

	if language := c.Query("language"); language != "" {
		req.Language = &language
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

	repos, total, err := h.repoService.List(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"repositories": repos,
			"total":        total,
			"page":         page,
			"limit":        limit,
		},
	})
}

// GetRepositoryStatistics 获取仓库统计信息
func (h *RepositoryHandler) GetRepositoryStatistics(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	stats, err := h.repoService.GetStatistics(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data":    stats,
	})
}

// UpdateRepositoryStatistics 更新仓库统计信息
func (h *RepositoryHandler) UpdateRepositoryStatistics(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	if err := h.repoService.UpdateStatistics(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "统计信息更新成功",
	})
}