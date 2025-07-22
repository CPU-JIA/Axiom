package handlers

import (
	"net/http"
	"strconv"
	"time"

	"cicd-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PipelineRunHandler struct {
	pipelineRunService services.PipelineRunService
}

func NewPipelineRunHandler(pipelineRunService services.PipelineRunService) *PipelineRunHandler {
	return &PipelineRunHandler{
		pipelineRunService: pipelineRunService,
	}
}

// CreatePipelineRun 创建流水线运行
// @Summary 创建流水线运行
// @Description 手动触发流水线执行
// @Tags pipeline-runs
// @Accept json
// @Produce json
// @Param run body services.CreatePipelineRunRequest true "运行请求"
// @Success 201 {object} APIResponse{data=models.PipelineRun}
// @Failure 400 {object} APIResponse
// @Router /api/v1/pipeline-runs [post]
func (h *PipelineRunHandler) CreatePipelineRun(c *gin.Context) {
	var req services.CreatePipelineRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "请求参数错误",
			Error:   err.Error(),
		})
		return
	}

	// 从上下文获取用户ID（假设中间件已设置）
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			req.TriggerBy = &uid
		}
	}

	run, err := h.pipelineRunService.Create(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "创建流水线运行失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: "流水线运行创建成功",
		Data:    run,
	})
}

// GetPipelineRun 获取流水线运行详情
// @Summary 获取流水线运行详情
// @Description 根据ID获取流水线运行详细信息
// @Tags pipeline-runs
// @Produce json
// @Param id path string true "运行ID"
// @Success 200 {object} APIResponse{data=models.PipelineRun}
// @Failure 404 {object} APIResponse
// @Router /api/v1/pipeline-runs/{id} [get]
func (h *PipelineRunHandler) GetPipelineRun(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的运行ID",
			Error:   err.Error(),
		})
		return
	}

	run, err := h.pipelineRunService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Message: "流水线运行不存在",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    run,
	})
}

// ListPipelineRuns 列表查询流水线运行
// @Summary 列表查询流水线运行
// @Description 分页查询流水线运行列表
// @Tags pipeline-runs
// @Produce json
// @Param pipeline_id query string false "流水线ID"
// @Param project_id query string false "项目ID"
// @Param status query string false "状态筛选"
// @Param trigger_type query string false "触发类型"
// @Param trigger_by query string false "触发用户ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param sort_by query string false "排序字段" default(created_at)
// @Param sort_desc query bool false "是否降序" default(true)
// @Success 200 {object} APIResponse{data=PagedResponse}
// @Router /api/v1/pipeline-runs [get]
func (h *PipelineRunHandler) ListPipelineRuns(c *gin.Context) {
	req := &services.ListPipelineRunsRequest{
		Page:  1,
		Limit: 20,
	}

	// 解析查询参数
	if pipelineIDStr := c.Query("pipeline_id"); pipelineIDStr != "" {
		if pipelineID, err := uuid.Parse(pipelineIDStr); err == nil {
			req.PipelineID = &pipelineID
		}
	}

	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if projectID, err := uuid.Parse(projectIDStr); err == nil {
			req.ProjectID = &projectID
		}
	}

	if status := c.Query("status"); status != "" {
		req.Status = &status
	}

	if triggerType := c.Query("trigger_type"); triggerType != "" {
		req.TriggerType = &triggerType
	}

	if triggerByStr := c.Query("trigger_by"); triggerByStr != "" {
		if triggerBy, err := uuid.Parse(triggerByStr); err == nil {
			req.TriggerBy = &triggerBy
		}
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &endTime
		}
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

	runs, total, err := h.pipelineRunService.List(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "查询流水线运行列表失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: PagedResponse{
			Items: runs,
			Pagination: Pagination{
				Page:  req.Page,
				Limit: req.Limit,
				Total: total,
			},
		},
	})
}

// CancelPipelineRun 取消流水线运行
// @Summary 取消流水线运行
// @Description 取消正在运行的流水线
// @Tags pipeline-runs
// @Accept json
// @Produce json
// @Param id path string true "运行ID"
// @Param cancel_request body CancelRunRequest true "取消请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Router /api/v1/pipeline-runs/{id}/cancel [post]
func (h *PipelineRunHandler) CancelPipelineRun(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的运行ID",
			Error:   err.Error(),
		})
		return
	}

	var req CancelRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有提供reason，使用默认值
		req.Reason = "用户手动取消"
	}

	if err := h.pipelineRunService.Cancel(id, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "取消流水线运行失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "流水线运行已取消",
	})
}

// RetryPipelineRun 重试流水线运行
// @Summary 重试流水线运行
// @Description 重新执行失败或取消的流水线
// @Tags pipeline-runs
// @Produce json
// @Param id path string true "原运行ID"
// @Success 201 {object} APIResponse{data=models.PipelineRun}
// @Failure 400 {object} APIResponse
// @Router /api/v1/pipeline-runs/{id}/retry [post]
func (h *PipelineRunHandler) RetryPipelineRun(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的运行ID",
			Error:   err.Error(),
		})
		return
	}

	newRun, err := h.pipelineRunService.Retry(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "重试流水线运行失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: "流水线运行重试成功",
		Data:    newRun,
	})
}

// GetPipelineRunsByPipeline 获取流水线的运行历史
// @Summary 获取流水线的运行历史
// @Description 获取指定流水线的运行历史记录
// @Tags pipeline-runs
// @Produce json
// @Param pipeline_id path string true "流水线ID"
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} APIResponse{data=[]models.PipelineRun}
// @Router /api/v1/pipelines/{pipeline_id}/runs [get]
func (h *PipelineRunHandler) GetPipelineRunsByPipeline(c *gin.Context) {
	pipelineIDStr := c.Param("pipeline_id")
	pipelineID, err := uuid.Parse(pipelineIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的流水线ID",
			Error:   err.Error(),
		})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	runs, err := h.pipelineRunService.GetByPipeline(pipelineID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "获取运行历史失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    runs,
	})
}

// GetPipelineRunStatistics 获取流水线运行统计
// @Summary 获取流水线运行统计
// @Description 获取流水线运行统计信息
// @Tags pipeline-runs
// @Produce json
// @Param pipeline_id query string false "流水线ID"
// @Param project_id query string false "项目ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param group_by query string false "分组方式" Enums(day,hour,status)
// @Success 200 {object} APIResponse{data=services.PipelineRunStats}
// @Router /api/v1/pipeline-runs/statistics [get]
func (h *PipelineRunHandler) GetPipelineRunStatistics(c *gin.Context) {
	req := &services.PipelineRunStatsRequest{}

	if pipelineIDStr := c.Query("pipeline_id"); pipelineIDStr != "" {
		if pipelineID, err := uuid.Parse(pipelineIDStr); err == nil {
			req.PipelineID = &pipelineID
		}
	}

	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if projectID, err := uuid.Parse(projectIDStr); err == nil {
			req.ProjectID = &projectID
		}
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &endTime
		}
	}

	if groupBy := c.Query("group_by"); groupBy != "" {
		req.GroupBy = groupBy
	}

	stats, err := h.pipelineRunService.GetStatistics(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "获取运行统计失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// TriggerPipelineByPipeline 触发流水线执行（简化接口）
// @Summary 触发流水线执行
// @Description 直接触发指定流水线执行
// @Tags pipelines
// @Accept json
// @Produce json
// @Param pipeline_id path string true "流水线ID"
// @Param trigger_request body TriggerPipelineRequest true "触发请求"
// @Success 201 {object} APIResponse{data=models.PipelineRun}
// @Router /api/v1/pipelines/{pipeline_id}/trigger [post]
func (h *PipelineRunHandler) TriggerPipelineByPipeline(c *gin.Context) {
	pipelineIDStr := c.Param("pipeline_id")
	pipelineID, err := uuid.Parse(pipelineIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "无效的流水线ID",
			Error:   err.Error(),
		})
		return
	}

	var triggerReq TriggerPipelineRequest
	if err := c.ShouldBindJSON(&triggerReq); err != nil {
		// 如果没有请求体，使用默认值
		triggerReq = TriggerPipelineRequest{
			TriggerType: "manual",
			Parameters:  make(map[string]interface{}),
		}
	}

	// 构建创建请求
	req := services.CreatePipelineRunRequest{
		PipelineID:  pipelineID,
		TriggerType: triggerReq.TriggerType,
		Parameters:  triggerReq.Parameters,
	}

	if triggerReq.Environment != "" {
		req.Environment = &triggerReq.Environment
	}

	if triggerReq.ScheduledAt != nil {
		req.ScheduledAt = triggerReq.ScheduledAt
	}

	// 从上下文获取用户ID
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			req.TriggerBy = &uid
		}
	}

	run, err := h.pipelineRunService.Create(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "触发流水线执行失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: "流水线已触发执行",
		Data:    run,
	})
}

// 请求结构体
type CancelRunRequest struct {
	Reason string `json:"reason"`
}

type TriggerPipelineRequest struct {
	TriggerType string                 `json:"trigger_type" binding:"required"`
	Parameters  map[string]interface{} `json:"parameters"`
	Environment string                 `json:"environment"`
	ScheduledAt *time.Time             `json:"scheduled_at"`
}