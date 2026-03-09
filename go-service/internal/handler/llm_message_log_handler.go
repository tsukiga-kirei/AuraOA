package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

// LLMMessageLogHandler 处理大模型消息记录相关的 HTTP 请求。
type LLMMessageLogHandler struct {
	logService *service.LLMMessageLogService
}

// NewLLMMessageLogHandler 创建一个新的 LLMMessageLogHandler 实例。
func NewLLMMessageLogHandler(logService *service.LLMMessageLogService) *LLMMessageLogHandler {
	return &LLMMessageLogHandler{logService: logService}
}

// QueryTokenUsage 处理 GET /api/tenant/stats/token-usage
func (h *LLMMessageLogHandler) QueryTokenUsage(c *gin.Context) {
	var query dto.TokenUsageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	startTime, err := time.Parse(time.RFC3339, query.StartTime)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "start_time 格式无效")
		return
	}
	endTime, err := time.Parse(time.RFC3339, query.EndTime)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "end_time 格式无效")
		return
	}

	var modelConfigID *uuid.UUID
	if query.ModelConfigID != "" {
		id, err := uuid.Parse(query.ModelConfigID)
		if err == nil {
			modelConfigID = &id
		}
	}

	summaries, err := h.logService.QueryTokenUsage(c, startTime, endTime, modelConfigID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, summaries)
}

// QueryAllTenantsTokenUsage 处理 GET /api/admin/stats/token-usage
func (h *LLMMessageLogHandler) QueryAllTenantsTokenUsage(c *gin.Context) {
	var query dto.TokenUsageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	startTime, err := time.Parse(time.RFC3339, query.StartTime)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "start_time 格式无效")
		return
	}
	endTime, err := time.Parse(time.RFC3339, query.EndTime)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "end_time 格式无效")
		return
	}

	summaries, err := h.logService.QueryAllTenantsTokenUsage(startTime, endTime)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, summaries)
}
