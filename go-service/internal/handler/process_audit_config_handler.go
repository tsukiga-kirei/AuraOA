package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

// ProcessAuditConfigHandler 处理流程审核配置相关的 HTTP 请求。
type ProcessAuditConfigHandler struct {
	configService *service.ProcessAuditConfigService
}

// NewProcessAuditConfigHandler 创建一个新的 ProcessAuditConfigHandler 实例。
func NewProcessAuditConfigHandler(configService *service.ProcessAuditConfigService) *ProcessAuditConfigHandler {
	return &ProcessAuditConfigHandler{configService: configService}
}

// List 处理 GET /api/tenant/rules/configs
func (h *ProcessAuditConfigHandler) List(c *gin.Context) {
	configs, err := h.configService.List(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, configs)
}

// Create 处理 POST /api/tenant/rules/configs
func (h *ProcessAuditConfigHandler) Create(c *gin.Context) {
	var req dto.CreateProcessAuditConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	cfg, err := h.configService.Create(c, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, cfg)
}

// GetByID 处理 GET /api/tenant/rules/configs/:id
func (h *ProcessAuditConfigHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	cfg, err := h.configService.GetByID(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, cfg)
}

// Update 处理 PUT /api/tenant/rules/configs/:id
func (h *ProcessAuditConfigHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateProcessAuditConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	cfg, err := h.configService.Update(c, id, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, cfg)
}

// Delete 处理 DELETE /api/tenant/rules/configs/:id
func (h *ProcessAuditConfigHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.configService.Delete(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// TestConnection 处理 POST /api/tenant/rules/configs/test-connection
func (h *ProcessAuditConfigHandler) TestConnection(c *gin.Context) {
	var req dto.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	info, err := h.configService.TestConnection(c, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, info)
}

// FetchFields 处理 POST /api/tenant/rules/configs/:id/fetch-fields
func (h *ProcessAuditConfigHandler) FetchFields(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	fields, err := h.configService.FetchFields(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, fields)
}
