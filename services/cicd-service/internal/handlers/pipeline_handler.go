package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"cicd-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PipelineHandler struct {
	pipelineService services.PipelineService
}

func NewPipelineHandler(pipelineService services.PipelineService) *PipelineHandler {
	return &PipelineHandler{
		pipelineService: pipelineService,
	}
}

// CreatePipeline 创建流水线
// @Summary 创建流水线
// @Description 创建新的CI/CD流水线
// @Tags pipelines
// @Accept json
// @Produce json
// @Param pipeline body services.CreatePipelineRequest true "流水线信息"
// @Success 201 {object} APIResponse{data=models.Pipeline}
// @Failure 400 {object} APIResponse
// @Router /api/v1/pipelines [post]
func (h *PipelineHandler) CreatePipeline(c *gin.Context) {
	var req services.CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "请求参数错误",
			Error:   err.Error(),
		})
		return
	}

	pipeline, err := h.pipelineService.Create(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "创建流水线失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: "流水线创建成功",
		Data:    pipeline,
	})
}

// GetPipeline 获取流水线详情
// @Summary 获取流水线详情
// @Description 根据ID获取流水线详细信息
// @Tags pipelines
// @Produce json
// @Param id path string true "流水线ID"
// @Success 200 {object} APIResponse{data=models.Pipeline}
// @Failure 404 {object} APIResponse
// @Router /api/v1/pipelines/{id} [get]
func (h *PipelineHandler) GetPipeline(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的流水线ID",
			Error:   err.Error(),
		})
		return
	}

	pipeline, err := h.pipelineService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Message: "流水线不存在",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    pipeline,
	})
}

// UpdatePipeline 更新流水线
// @Summary 更新流水线
// @Description 更新流水线配置
// @Tags pipelines
// @Accept json
// @Produce json
// @Param id path string true "流水线ID"
// @Param pipeline body services.UpdatePipelineRequest true "更新信息"
// @Success 200 {object} APIResponse{data=models.Pipeline}
// @Failure 400 {object} APIResponse
// @Router /api/v1/pipelines/{id} [put]
func (h *PipelineHandler) UpdatePipeline(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的流水线ID",
			Error:   err.Error(),
		})
		return
	}

	var req services.UpdatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "请求参数错误",
			Error:   err.Error(),
		})
		return
	}

	pipeline, err := h.pipelineService.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "更新流水线失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "流水线更新成功",
		Data:    pipeline,
	})
}

// DeletePipeline 删除流水线
// @Summary 删除流水线
// @Description 软删除流水线
// @Tags pipelines
// @Produce json
// @Param id path string true "流水线ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Router /api/v1/pipelines/{id} [delete]
func (h *PipelineHandler) DeletePipeline(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的流水线ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.pipelineService.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "删除流水线失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "流水线删除成功",
	})
}

// ListPipelines 列表查询流水线
// @Summary 列表查询流水线
// @Description 分页查询流水线列表
// @Tags pipelines
// @Produce json
// @Param project_id query string false "项目ID"
// @Param status query string false "状态筛选"
// @Param search query string false "搜索关键词"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param sort_by query string false "排序字段" default(created_at)
// @Param sort_desc query bool false "是否降序" default(true)
// @Success 200 {object} APIResponse{data=PagedResponse}
// @Router /api/v1/pipelines [get]
func (h *PipelineHandler) ListPipelines(c *gin.Context) {
	req := &services.ListPipelinesRequest{
		Page:  1,
		Limit: 20,
	}

	// 解析查询参数
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if projectID, err := uuid.Parse(projectIDStr); err == nil {
			req.ProjectID = &projectID
		}
	}

	if status := c.Query("status"); status != "" {
		req.Status = &status
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

	pipelines, total, err := h.pipelineService.List(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "查询流水线列表失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: PagedResponse{
			Items: pipelines,
			Pagination: Pagination{
				Page:  req.Page,
				Limit: req.Limit,
				Total: total,
			},
		},
	})
}

// EnablePipeline 启用流水线
// @Summary 启用流水线
// @Description 启用被禁用的流水线
// @Tags pipelines
// @Produce json
// @Param id path string true "流水线ID"
// @Success 200 {object} APIResponse
// @Router /api/v1/pipelines/{id}/enable [post]
func (h *PipelineHandler) EnablePipeline(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的流水线ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.pipelineService.Enable(id); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "启用流水线失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "流水线已启用",
	})
}

// DisablePipeline 禁用流水线
// @Summary 禁用流水线
// @Description 禁用流水线，不再触发执行
// @Tags pipelines
// @Produce json
// @Param id path string true "流水线ID"
// @Success 200 {object} APIResponse
// @Router /api/v1/pipelines/{id}/disable [post]
func (h *PipelineHandler) DisablePipeline(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的流水线ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.pipelineService.Disable(id); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "禁用流水线失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "流水线已禁用",
	})
}

// ClonePipeline 克隆流水线
// @Summary 克隆流水线
// @Description 基于现有流水线创建副本
// @Tags pipelines
// @Accept json
// @Produce json
// @Param id path string true "源流水线ID"
// @Param clone_request body ClonePipelineRequest true "克隆请求"
// @Success 201 {object} APIResponse{data=models.Pipeline}
// @Router /api/v1/pipelines/{id}/clone [post]
func (h *PipelineHandler) ClonePipeline(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的流水线ID",
			Error:   err.Error(),
		})
		return
	}

	var req ClonePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "请求参数错误",
			Error:   err.Error(),
		})
		return
	}

	pipeline, err := h.pipelineService.Clone(id, req.NewName)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "克隆流水线失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: "流水线克隆成功",
		Data:    pipeline,
	})
}

// GetPipelineStatistics 获取流水线统计
// @Summary 获取流水线统计
// @Description 获取流水线运行统计信息
// @Tags pipelines
// @Produce json
// @Param project_id query string false "项目ID"
// @Success 200 {object} APIResponse{data=services.PipelineStats}
// @Router /api/v1/pipelines/statistics [get]
func (h *PipelineHandler) GetPipelineStatistics(c *gin.Context) {
	var projectID *uuid.UUID
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if id, err := uuid.Parse(projectIDStr); err == nil {
			projectID = &id
		}
	}

	stats, err := h.pipelineService.GetStatistics(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "获取统计信息失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// GetPipelineByProject 获取项目的流水线列表
// @Summary 获取项目的流水线列表
// @Description 获取指定项目的所有流水线
// @Tags pipelines
// @Produce json
// @Param project_id path string true "项目ID"
// @Success 200 {object} APIResponse{data=[]models.Pipeline}
// @Router /api/v1/projects/{project_id}/pipelines [get]
func (h *PipelineHandler) GetPipelinesByProject(c *gin.Context) {
	projectIDStr := c.Param("project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的项目ID",
			Error:   err.Error(),
		})
		return
	}

	pipelines, err := h.pipelineService.GetByProject(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "获取项目流水线失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    pipelines,
	})
}

// ClonePipelineRequest 克隆流水线请求
type ClonePipelineRequest struct {
	NewName string `json:"new_name" binding:"required" validate:"max=255"`
}

// APIResponse 统一API响应格式
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// PagedResponse 分页响应
type PagedResponse struct {
	Items      interface{} `json:"items"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination 分页信息
type Pagination struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}