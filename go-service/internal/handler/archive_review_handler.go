package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/repository"
	"oa-smart-audit/go-service/internal/service"
)

// ArchiveReviewHandler 处理归档复盘运行时请求。
type ArchiveReviewHandler struct {
	archiveService *service.ArchiveReviewService
}

func NewArchiveReviewHandler(archiveService *service.ArchiveReviewService) *ArchiveReviewHandler {
	return &ArchiveReviewHandler{archiveService: archiveService}
}

func (h *ArchiveReviewHandler) ListProcesses(c *gin.Context) {
	params := dto.ArchiveListParams{
		Keyword:     c.Query("keyword"),
		Applicant:   c.Query("applicant"),
		ProcessType: c.Query("process_type"),
		Department:  c.Query("department"),
		AuditStatus: c.Query("audit_status"),
		Page:        parseIntQuery(c, "page", 1),
		PageSize:    parseIntQuery(c, "page_size", 20),
	}
	resp, err := h.archiveService.ListProcessesPaged(c, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, resp)
}

func (h *ArchiveReviewHandler) GetStats(c *gin.Context) {
	stats, err := h.archiveService.GetStats(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *ArchiveReviewHandler) Execute(c *gin.Context) {
	var req dto.ArchiveReviewExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败: "+err.Error())
		return
	}

	result, err := h.archiveService.Execute(c, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if result.Status == model.AuditStatusPending {
		c.JSON(http.StatusAccepted, response.Response{
			Code:    0,
			Message: "accepted",
			Data:    result,
		})
		return
	}
	response.Success(c, result)
}

func (h *ArchiveReviewHandler) BatchExecute(c *gin.Context) {
	var req dto.ArchiveBatchExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败: "+err.Error())
		return
	}
	result, err := h.archiveService.BatchExecute(c, req.Items)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ArchiveReviewHandler) CancelJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	if err := h.archiveService.CancelJob(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"status": "cancelled"})
}

func (h *ArchiveReviewHandler) GetJobStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	data, err := h.archiveService.GetArchiveJobStatus(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func (h *ArchiveReviewHandler) GetJobStream(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}

	ch, closeSub, err := h.archiveService.SubscribeJobStream(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	defer closeSub()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			c.SSEvent("message", msg)
			c.Writer.Flush()
		}
	}
}

func (h *ArchiveReviewHandler) GetHistory(c *gin.Context) {
	processID := c.Param("processId")
	if processID == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "流程ID不能为空")
		return
	}
	items, err := h.archiveService.GetArchiveHistory(c, processID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *ArchiveReviewHandler) GetResult(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "记录 ID 无效")
		return
	}
	data, err := h.archiveService.GetArchiveResult(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// ListLogs GET /api/archive/logs (tenant_admin)
func (h *ArchiveReviewHandler) ListLogs(c *gin.Context) {
	filter, page, pageSize := parseArchiveLogQuery(c)
	items, total, err := h.archiveService.ListArchiveLogs(c, filter, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetLogStats GET /api/archive/logs/stats (tenant_admin)
func (h *ArchiveReviewHandler) GetLogStats(c *gin.Context) {
	stats, err := h.archiveService.GetArchiveLogStats(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// ExportLogs GET /api/archive/logs/export (tenant_admin) — CSV 下载
func (h *ArchiveReviewHandler) ExportLogs(c *gin.Context) {
	filter, _, _ := parseArchiveLogQuery(c)
	items, _, err := h.archiveService.ListArchiveLogs(c, filter, 1, 5000)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	filename := fmt.Sprintf("archive_logs_%s.csv", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Writer.Write([]byte("\xef\xbb\xbf"))

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"记录ID", "流程编号", "流程标题", "操作人", "流程类型", "合规性", "评分", "状态", "创建时间"})
	for _, item := range items {
		_ = w.Write([]string{
			item.ID.String(),
			item.ProcessID,
			item.Title,
			item.UserName,
			item.ProcessType,
			item.Compliance,
			fmt.Sprintf("%d", item.ComplianceScore),
			item.Status,
			item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	w.Flush()
}

func parseArchiveLogQuery(c *gin.Context) (repository.ArchiveLogFilter, int, int) {
	filter := repository.ArchiveLogFilter{
		Keyword:     c.Query("keyword"),
		ProcessType: c.Query("process_type"),
		Compliance:  c.Query("compliance"),
	}
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.StartDate = &t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			end := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &end
		}
	}
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)
	return filter, page, pageSize
}
