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

// ArchiveConfigHandler 处理归档复盘配置相关的 HTTP 请求。
type ArchiveConfigHandler struct {
	archiveService *service.ProcessArchiveConfigService
}

// NewArchiveConfigHandler 创建一个新的 ArchiveConfigHandler 实例。
func NewArchiveConfigHandler(archiveService *service.ProcessArchiveConfigService) *ArchiveConfigHandler {
	return &ArchiveConfigHandler{archiveService: archiveService}
}

// List 处理 GET /api/tenant/archive/configs
func (h *ArchiveConfigHandler) List(c *gin.Context) {
	cfgs, err := h.archiveService.List(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, cfgs)
}

// Create 处理 POST /api/tenant/archive/configs
func (h *ArchiveConfigHandler) Create(c *gin.Context) {
	var req dto.CreateProcessArchiveConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	cfg, err := h.archiveService.Create(c, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, cfg)
}

// GetByID 处理 GET /api/tenant/archive/configs/:id
func (h *ArchiveConfigHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "id 格式错误")
		return
	}
	cfg, err := h.archiveService.GetByID(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, cfg)
}

// Update 处理 PUT /api/tenant/archive/configs/:id
func (h *ArchiveConfigHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "id 格式错误")
		return
	}
	var req dto.UpdateProcessArchiveConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	cfg, err := h.archiveService.Update(c, id, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, cfg)
}

// Delete 处理 DELETE /api/tenant/archive/configs/:id
func (h *ArchiveConfigHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "id 格式错误")
		return
	}
	if err := h.archiveService.Delete(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// TestConnection 处理 POST /api/tenant/archive/configs/test-connection
func (h *ArchiveConfigHandler) TestConnection(c *gin.Context) {
	var req dto.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	info, err := h.archiveService.TestConnection(c, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, info)
}

// FetchFields 处理 POST /api/tenant/archive/configs/:id/fetch-fields
func (h *ArchiveConfigHandler) FetchFields(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "id 格式错误")
		return
	}
	fields, err := h.archiveService.FetchFields(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, fields)
}

// ListPromptTemplates 处理 GET /api/tenant/archive/prompt-templates
// 返回归档复盘专用系统提示词模板（archive_ 前缀）
func (h *ArchiveConfigHandler) ListPromptTemplates(c *gin.Context) {
	templates, err := h.archiveService.ListArchivePromptTemplates()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, templates)
}

// ArchiveRuleHandler 处理归档规则相关的 HTTP 请求。
type ArchiveRuleHandler struct {
	ruleService *service.ArchiveRuleService
}

// NewArchiveRuleHandler 创建一个新的 ArchiveRuleHandler 实例。
func NewArchiveRuleHandler(ruleService *service.ArchiveRuleService) *ArchiveRuleHandler {
	return &ArchiveRuleHandler{ruleService: ruleService}
}

// List 处理 GET /api/tenant/archive/audit-rules
func (h *ArchiveRuleHandler) List(c *gin.Context) {
	configIDStr := c.Query("config_id")
	if configIDStr == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "config_id 参数必填")
		return
	}

	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "config_id 格式错误")
		return
	}

	var ruleScope *string
	if v := c.Query("rule_scope"); v != "" {
		ruleScope = &v
	}

	var enabled *bool
	if v := c.Query("enabled"); v != "" {
		b := v == "true"
		enabled = &b
	}

	rules, err := h.ruleService.ListByConfigIDFilter(c, configID, ruleScope, enabled)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, rules)
}

// Create 处理 POST /api/tenant/archive/audit-rules
func (h *ArchiveRuleHandler) Create(c *gin.Context) {
	var req dto.CreateArchiveRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	rule, err := h.ruleService.Create(c, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, rule)
}

// Update 处理 PUT /api/tenant/archive/audit-rules/:id
func (h *ArchiveRuleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateArchiveRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	rule, err := h.ruleService.Update(c, id, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, rule)
}

// Delete 处理 DELETE /api/tenant/archive/audit-rules/:id
func (h *ArchiveRuleHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.ruleService.Delete(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}
