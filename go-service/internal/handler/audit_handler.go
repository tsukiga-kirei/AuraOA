package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

// AuditHandler 审核工作台相关 HTTP 请求处理。
type AuditHandler struct {
	auditService *service.AuditExecuteService
}

func NewAuditHandler(auditService *service.AuditExecuteService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// ListProcesses GET /api/audit/processes?tab=pending_ai&keyword=...
func (h *AuditHandler) ListProcesses(c *gin.Context) {
	tab := c.DefaultQuery("tab", "pending_ai")
	username := getUsername(c)
	if username == "" {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "用户信息缺失")
		return
	}

	items, err := h.auditService.ListProcesses(c, tab, username)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// GetStats GET /api/audit/stats
func (h *AuditHandler) GetStats(c *gin.Context) {
	stats, err := h.auditService.GetStats(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// Execute POST /api/audit/execute
func (h *AuditHandler) Execute(c *gin.Context) {
	var req service.AuditExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败: "+err.Error())
		return
	}

	result, err := h.auditService.Execute(c, &req)
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

// GetJobStatus GET /api/audit/jobs/:id
func (h *AuditHandler) GetJobStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	data, err := h.auditService.GetAuditJobStatus(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// CancelJob POST /api/audit/cancel/:id
func (h *AuditHandler) CancelJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	if err := h.auditService.CancelJob(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"status": "cancelled"})
}

// BatchExecute POST /api/audit/batch
func (h *AuditHandler) BatchExecute(c *gin.Context) {
	var req struct {
		Items []service.AuditExecuteRequest `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败: "+err.Error())
		return
	}

	result, err := h.auditService.BatchExecute(c, req.Items)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, result)
}

// GetJobStream GET /api/audit/stream/:id
func (h *AuditHandler) GetJobStream(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}

	ch, closeSub, err := h.auditService.SubscribeJobStream(c, id)
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

// GetAuditChain GET /api/audit/chain/:processId
func (h *AuditHandler) GetAuditChain(c *gin.Context) {
	processID := c.Param("processId")
	if processID == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "流程ID不能为空")
		return
	}

	chain, err := h.auditService.GetAuditChain(c, processID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, chain)
}

func getUsername(c *gin.Context) string {
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		return ""
	}
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		return ""
	}
	return claims.Username
}
