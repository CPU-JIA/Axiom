package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"cicd-service/internal/models"
	"cicd-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CacheHandler struct {
	cacheService services.CacheService
}

func NewCacheHandler(cacheService services.CacheService) *CacheHandler {
	return &CacheHandler{
		cacheService: cacheService,
	}
}

// StoreCache 存储构建缓存
// @Summary 存储构建缓存
// @Description 存储构建缓存文件
// @Tags cache
// @Accept json
// @Produce json
// @Param cache body services.StoreCacheRequest true "缓存请求"
// @Success 201 {object} APIResponse{data=models.BuildCache}
// @Failure 400 {object} APIResponse
// @Router /api/v1/cache [post]
func (h *CacheHandler) StoreCache(c *gin.Context) {
	var req services.StoreCacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "请求参数错误",
			Error:   err.Error(),
		})
		return
	}

	cache, err := h.cacheService.Store(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "存储缓存失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: "缓存存储成功",
		Data:    cache,
	})
}

// RetrieveCache 检索构建缓存
// @Summary 检索构建缓存
// @Description 根据键值检索构建缓存
// @Tags cache
// @Produce json
// @Param key path string true "缓存键"
// @Param project_id query string true "项目ID"
// @Success 200 {object} APIResponse{data=models.BuildCache}
// @Failure 404 {object} APIResponse
// @Router /api/v1/cache/{key} [get]
func (h *CacheHandler) RetrieveCache(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "缓存键不能为空",
		})
		return
	}

	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "项目ID不能为空",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的项目ID",
			Error:   err.Error(),
		})
		return
	}

	cache, err := h.cacheService.Retrieve(key, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Message: "缓存不存在或已过期",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    cache,
	})
}

// DeleteCache 删除构建缓存
// @Summary 删除构建缓存
// @Description 根据ID删除构建缓存
// @Tags cache
// @Produce json
// @Param id path string true "缓存ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Router /api/v1/cache/{id} [delete]
func (h *CacheHandler) DeleteCache(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的缓存ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.cacheService.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "删除缓存失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "缓存删除成功",
	})
}

// DeleteCacheByKey 根据键删除构建缓存
// @Summary 根据键删除构建缓存
// @Description 根据键值和项目ID删除构建缓存
// @Tags cache
// @Produce json
// @Param key path string true "缓存键"
// @Param project_id query string true "项目ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Router /api/v1/cache/by-key/{key} [delete]
func (h *CacheHandler) DeleteCacheByKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "缓存键不能为空",
		})
		return
	}

	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "项目ID不能为空",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的项目ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.cacheService.DeleteByKey(key, projectID); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "删除缓存失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "缓存删除成功",
	})
}

// ListCaches 列表查询构建缓存
// @Summary 列表查询构建缓存
// @Description 分页查询构建缓存列表
// @Tags cache
// @Produce json
// @Param project_id query string false "项目ID"
// @Param search query string false "搜索关键词"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param sort_by query string false "排序字段" default(created_at)
// @Param sort_desc query bool false "是否降序" default(true)
// @Success 200 {object} APIResponse{data=PagedResponse}
// @Router /api/v1/cache [get]
func (h *CacheHandler) ListCaches(c *gin.Context) {
	req := &services.ListCacheRequest{
		Page:  1,
		Limit: 20,
	}

	// 解析查询参数
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if projectID, err := uuid.Parse(projectIDStr); err == nil {
			req.ProjectID = &projectID
		}
	}

	if search := c.Query("search"); search != "" {
		req.Search = &search
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			req.Limit = limit
		}
	}

	if sortBy := c.Query("sort_by"); sortBy != "" {
		req.SortBy = sortBy
	}

	if sortDescStr := c.Query("sort_desc"); sortDescStr != "" {
		if sortDesc, err := strconv.ParseBool(sortDescStr); err == nil {
			req.SortDesc = sortDesc
		}
	}

	caches, total, err := h.cacheService.List(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "查询缓存列表失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: PagedResponse{
			Items: caches,
			Pagination: Pagination{
				Page:  req.Page,
				Limit: req.Limit,
				Total: total,
			},
		},
	})
}

// GetCacheStatistics 获取缓存统计信息
// @Summary 获取缓存统计信息
// @Description 获取构建缓存统计数据
// @Tags cache
// @Produce json
// @Param project_id query string false "项目ID"
// @Success 200 {object} APIResponse{data=services.CacheStats}
// @Router /api/v1/cache/statistics [get]
func (h *CacheHandler) GetCacheStatistics(c *gin.Context) {
	var projectID *uuid.UUID
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if id, err := uuid.Parse(projectIDStr); err == nil {
			projectID = &id
		}
	}

	stats, err := h.cacheService.GetStatistics(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "获取缓存统计失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: stats,
	})
}

// CleanupCaches 清理过期缓存
// @Summary 清理过期缓存
// @Description 手动触发清理过期和超限缓存
// @Tags cache
// @Produce json
// @Success 200 {object} APIResponse
// @Router /api/v1/cache/cleanup [post]
func (h *CacheHandler) CleanupCaches(c *gin.Context) {
	if err := h.cacheService.Cleanup(); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "清理缓存失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "缓存清理完成",
	})
}

// ValidateCache 验证缓存完整性
// @Summary 验证缓存完整性
// @Description 检查缓存文件是否完整有效
// @Tags cache
// @Produce json
// @Param id path string true "缓存ID"
// @Success 200 {object} APIResponse{data=ValidateCacheResponse}
// @Router /api/v1/cache/{id}/validate [get]
func (h *CacheHandler) ValidateCache(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的缓存ID",
			Error:   err.Error(),
		})
		return
	}

	// 先获取缓存信息
	cache, err := h.getCacheByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Message: "缓存不存在",
			Error:   err.Error(),
		})
		return
	}

	// 验证缓存
	isValid := h.cacheService.ValidateCache(cache)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: ValidateCacheResponse{
			CacheID: id,
			IsValid: isValid,
		},
	})
}

// GetCachePath 获取缓存路径
// @Summary 获取缓存路径
// @Description 获取缓存文件的实际存储路径
// @Tags cache
// @Produce json
// @Param id path string true "缓存ID"
// @Success 200 {object} APIResponse{data=CachePathResponse}
// @Router /api/v1/cache/{id}/path [get]
func (h *CacheHandler) GetCachePath(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的缓存ID",
			Error:   err.Error(),
		})
		return
	}

	// 先获取缓存信息
	cache, err := h.getCacheByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Message: "缓存不存在",
			Error:   err.Error(),
		})
		return
	}

	path := h.cacheService.GetCachePath(cache)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: CachePathResponse{
			CacheID: id,
			Path:    path,
		},
	})
}

// CalculateCacheChecksum 计算缓存校验和
// @Summary 计算缓存校验和
// @Description 重新计算缓存文件的校验和
// @Tags cache
// @Produce json
// @Param id path string true "缓存ID"
// @Success 200 {object} APIResponse{data=ChecksumResponse}
// @Router /api/v1/cache/{id}/checksum [get]
func (h *CacheHandler) CalculateCacheChecksum(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的缓存ID",
			Error:   err.Error(),
		})
		return
	}

	// 先获取缓存信息
	cache, err := h.getCacheByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Message: "缓存不存在",
			Error:   err.Error(),
		})
		return
	}

	path := h.cacheService.GetCachePath(cache)
	checksum, err := h.cacheService.CalculateChecksum(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "计算校验和失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: ChecksumResponse{
			CacheID:          id,
			CurrentChecksum:  checksum,
			StoredChecksum:   cache.Checksum,
			ChecksumMatches:  checksum == cache.Checksum,
		},
	})
}

// 辅助方法 - 获取缓存信息（这里需要实际的实现，可能需要添加到CacheService接口）
func (h *CacheHandler) getCacheByID(id uuid.UUID) (*models.BuildCache, error) {
	// 这里需要实现通过ID获取缓存的逻辑
	// 目前services.CacheService接口没有GetByID方法，可能需要添加
	// 暂时返回错误，表示未实现
	return nil, fmt.Errorf("getCacheByID方法未实现，需要在CacheService接口中添加GetByID方法")
}

// 响应结构体
type ValidateCacheResponse struct {
	CacheID uuid.UUID `json:"cache_id"`
	IsValid bool      `json:"is_valid"`
}

type CachePathResponse struct {
	CacheID uuid.UUID `json:"cache_id"`
	Path    string    `json:"path"`
}

type ChecksumResponse struct {
	CacheID         uuid.UUID `json:"cache_id"`
	CurrentChecksum string    `json:"current_checksum"`
	StoredChecksum  string    `json:"stored_checksum"`
	ChecksumMatches bool      `json:"checksum_matches"`
}