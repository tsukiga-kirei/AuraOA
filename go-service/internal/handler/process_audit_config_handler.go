// 流程审核配置处理器，负责审核流程数据源配置的增删改查及连接测试。
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

// NewProcessAuditConfigHandler 创建流程审核配置处理器实例。
func NewProcessAuditConfigHandler(configService *service.ProcessAuditConfigService) *ProcessAuditConfigHandler {
	return &ProcessAuditConfigHandler{configService: configService}
}

// List 获取当前租户的所有流程审核配置列表。
// GET /api/tenant/rules/configs
// 返回：配置列表数组，包含数据源连接信息和流程类型。
func (h *ProcessAuditConfigHandler) List(c *gin.Context) {
	configs, err := h.configService.List(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, configs)
}

// Create 创建新的流程审核配置。
// POST /api/tenant/rules/configs
// 请求体：CreateProcessAuditConfigRequest（流程类型、数据源连接参数等）
// 返回：新建的配置对象。
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

// GetByID 根据 ID 获取单条流程审核配置详情。
// GET /api/tenant/rules/configs/:id
// 路径参数：id（UUID 格式）
// 返回：配置详情对象。
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

// Update 更新指定流程审核配置。
// PUT /api/tenant/rules/configs/:id
// 路径参数：id（UUID 格式）
// 请求体：UpdateProcessAuditConfigRequest
// 返回：更新后的配置对象。
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

// Delete 删除指定流程审核配置。
// DELETE /api/tenant/rules/configs/:id
// 路径参数：id（UUID 格式）
// 返回：null（成功时）。
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

// TestConnection 测试审核数据源连接是否可用（不保存配置）。
// POST /api/tenant/rules/configs/test-connection
// 请求体：TestConnectionRequest（数据库连接参数）
// 返回：连接测试结果信息。
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

// FetchFields 从已保存的审核配置对应的数据源中拉取可用字段列表。
// POST /api/tenant/rules/configs/:id/fetch-fields
// 路径参数：id（UUID 格式）
// 返回：字段列表（主表字段 + 明细表字段）。
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

// ListPromptTemplates 获取审核专用的系统提示词模板列表（audit_ 前缀）。
// GET /api/tenant/rules/prompt-templates
// 返回：提示词模板数组。
func (h *ProcessAuditConfigHandler) ListPromptTemplates(c *gin.Context) {
	templates, err := h.configService.ListPromptTemplates()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, templates)
}
