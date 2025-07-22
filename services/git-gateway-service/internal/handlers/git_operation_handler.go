package handlers

import (
	"net/http"
	"strconv"
	"time"

	"git-gateway-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GitOperationHandler Git操作审计处理器
type GitOperationHandler struct {
	gitOpService services.GitOperationService
}

// NewGitOperationHandler 创建Git操作审计处理器
func NewGitOperationHandler(gitOpService services.GitOperationService) *GitOperationHandler {
	return &GitOperationHandler{
		gitOpService: gitOpService,
	}
}

// GetOperation 获取操作记录详情
func (h *GitOperationHandler) GetOperation(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的操作记录ID"})
		return
	}

	operation, err := h.gitOpService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data":    operation,
	})
}

// ListOperations 列表查询操作记录
func (h *GitOperationHandler) ListOperations(c *gin.Context) {
	var req services.ListOperationsRequest

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

	if operation := c.Query("operation"); operation != "" {
		req.Operation = &operation
	}

	if protocol := c.Query("protocol"); protocol != "" {
		req.Protocol = &protocol
	}

	if successParam := c.Query("success"); successParam != "" {
		success := successParam == "true"
		req.Success = &success
	}

	if clientIP := c.Query("client_ip"); clientIP != "" {
		req.ClientIP = &clientIP
	}

	// 时间范围参数
	if startTimeParam := c.Query("start_time"); startTimeParam != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始时间格式"})
			return
		}
		req.StartTime = &startTime
	}

	if endTimeParam := c.Query("end_time"); endTimeParam != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束时间格式"})
			return
		}
		req.EndTime = &endTime
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	req.Page = page
	req.Limit = limit

	// 排序参数
	req.SortBy = c.DefaultQuery("sort_by", "created_at")
	req.SortDesc = c.DefaultQuery("sort_desc", "true") == "true"

	operations, total, err := h.gitOpService.List(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"operations": operations,
			"total":      total,
			"page":       page,
			"limit":      limit,
		},
	})
}

// GetOperationStats 获取操作统计信息
func (h *GitOperationHandler) GetOperationStats(c *gin.Context) {
	var req services.OperationStatsRequest

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

	// 时间范围参数
	if startTimeParam := c.Query("start_time"); startTimeParam != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始时间格式"})
			return
		}
		req.StartTime = &startTime
	}

	if endTimeParam := c.Query("end_time"); endTimeParam != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束时间格式"})
			return
		}
		req.EndTime = &endTime
	}

	// 分组参数
	req.GroupBy = c.DefaultQuery("group_by", "")

	stats, err := h.gitOpService.GetOperationStats(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data":    stats,
	})
}

// CleanupOldRecords 清理旧的操作记录
func (h *GitOperationHandler) CleanupOldRecords(c *gin.Context) {
	retentionDays, _ := strconv.Atoi(c.DefaultQuery("retention_days", "90"))
	
	if retentionDays < 1 || retentionDays > 365 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "保留天数必须在1-365天之间",
		})
		return
	}

	if err := h.gitOpService.CleanupOldRecords(retentionDays); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "清理完成",
	})
}