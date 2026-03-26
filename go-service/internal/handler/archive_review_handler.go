package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
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
	resp, err := h.archiveService.ListProcesses(c)
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
