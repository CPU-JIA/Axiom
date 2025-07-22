package handlers

import (
	"net/http"
	"strconv"

	"git-gateway-service/internal/models"
	"git-gateway-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BranchHandler 分支处理器
type BranchHandler struct {
	branchService services.BranchService
}

// NewBranchHandler 创建分支处理器
func NewBranchHandler(branchService services.BranchService) *BranchHandler {
	return &BranchHandler{
		branchService: branchService,
	}
}

// CreateBranch 创建分支
func (h *BranchHandler) CreateBranch(c *gin.Context) {
	var req services.CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	branch, err := h.branchService.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "分支创建成功",
		"data":    branch,
	})
}

// GetBranch 获取分支详情
func (h *BranchHandler) GetBranch(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分支ID"})
		return
	}

	branch, err := h.branchService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data":    branch,
	})
}

// GetBranchByName 根据仓库ID和名称获取分支
func (h *BranchHandler) GetBranchByName(c *gin.Context) {
	repositoryIDParam := c.Param("repository_id")
	repositoryID, err := uuid.Parse(repositoryIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分支名称不能为空"})
		return
	}

	branch, err := h.branchService.GetByName(repositoryID, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data":    branch,
	})
}

// UpdateBranch 更新分支
func (h *BranchHandler) UpdateBranch(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分支ID"})
		return
	}

	var req services.UpdateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	branch, err := h.branchService.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"data":    branch,
	})
}

// DeleteBranch 删除分支
func (h *BranchHandler) DeleteBranch(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分支ID"})
		return
	}

	if err := h.branchService.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// ListBranches 列表查询分支
func (h *BranchHandler) ListBranches(c *gin.Context) {
	var req services.ListBranchesRequest

	// 解析查询参数
	if repositoryIDParam := c.Query("repository_id"); repositoryIDParam != "" {
		repositoryID, err := uuid.Parse(repositoryIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
			return
		}
		req.RepositoryID = &repositoryID
	}

	if isProtectedParam := c.Query("is_protected"); isProtectedParam != "" {
		isProtected := isProtectedParam == "true"
		req.IsProtected = &isProtected
	}

	if isDefaultParam := c.Query("is_default"); isDefaultParam != "" {
		isDefault := isDefaultParam == "true"
		req.IsDefault = &isDefault
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
	req.SortBy = c.DefaultQuery("sort_by", "")
	req.SortDesc = c.DefaultQuery("sort_desc", "false") == "true"

	branches, total, err := h.branchService.List(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"branches": branches,
			"total":    total,
			"page":     page,
			"limit":    limit,
		},
	})
}

// SetBranchProtection 设置分支保护
func (h *BranchHandler) SetBranchProtection(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分支ID"})
		return
	}

	var protection struct {
		RequireStatusChecks      bool `json:"require_status_checks"`
		RequireUpToDate          bool `json:"require_up_to_date"`
		RequirePullRequest       bool `json:"require_pull_request"`
		RequireCodeOwnerReviews  bool `json:"require_code_owner_reviews"`
		DismissStaleReviews      bool `json:"dismiss_stale_reviews"`
		RequiredReviewers        int  `json:"required_reviewers"`
		RestrictPushes           bool `json:"restrict_pushes"`
		AllowForcePushes         bool `json:"allow_force_pushes"`
		AllowDeletions           bool `json:"allow_deletions"`
	}

	if err := c.ShouldBindJSON(&protection); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	branchProtection := models.BranchProtection{
		RequireStatusChecks:     protection.RequireStatusChecks,
		RequireUpToDate:         protection.RequireUpToDate,
		RequirePullRequest:      protection.RequirePullRequest,
		RequireCodeOwnerReviews: protection.RequireCodeOwnerReviews,
		DismissStaleReviews:     protection.DismissStaleReviews,
		RequiredReviewers:       protection.RequiredReviewers,
		RestrictPushes:          protection.RestrictPushes,
		AllowForcePushes:        protection.AllowForcePushes,
		AllowDeletions:          protection.AllowDeletions,
	}

	if err := h.branchService.SetProtection(id, branchProtection); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "分支保护设置成功",
	})
}

// RemoveBranchProtection 移除分支保护
func (h *BranchHandler) RemoveBranchProtection(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分支ID"})
		return
	}

	if err := h.branchService.RemoveProtection(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "分支保护移除成功",
	})
}

// SetDefaultBranch 设置默认分支
func (h *BranchHandler) SetDefaultBranch(c *gin.Context) {
	repositoryIDParam := c.Param("repository_id")
	repositoryID, err := uuid.Parse(repositoryIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	branchIDParam := c.Param("branch_id")
	branchID, err := uuid.Parse(branchIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分支ID"})
		return
	}

	if err := h.branchService.SetDefault(repositoryID, branchID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "默认分支设置成功",
	})
}