package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/service"
)

// GetEmbedContext 嵌入页：按 requestid 获取流程上下文与审核结论。
// GET /api/embed/context?process_id=
func (h *AuditHandler) GetEmbedContext(c *gin.Context) {
	processID := c.Query("process_id")
	if processID == "" {
		processID = c.Query("requestid")
	}
	if processID == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "process_id 不能为空")
		return
	}
	data, err := h.auditService.GetEmbedContext(c, processID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// ExecuteEmbed 嵌入页发起审核（自动或手动重新审核）。
// POST /api/embed/execute
func (h *AuditHandler) ExecuteEmbed(c *gin.Context) {
	var req service.EmbedExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败: "+err.Error())
		return
	}
	result, err := h.auditService.ExecuteEmbed(c, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if result.Status == model.JobStatusPending {
		c.JSON(http.StatusAccepted, response.Response{
			Code:    0,
			Message: "accepted",
			Data:    result,
		})
		return
	}
	response.Success(c, result)
}
