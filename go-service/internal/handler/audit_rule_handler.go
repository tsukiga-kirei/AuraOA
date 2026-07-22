// 审核规则处理器，负责审核规则的增删改查。
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/service"
)

// AuditRuleHandler 处理审核规则相关的 HTTP 请求。
type AuditRuleHandler struct {
	ruleService   *service.AuditRuleService
	importService *service.RuleImportService
}

// NewAuditRuleHandler 创建审核规则处理器实例。
func NewAuditRuleHandler(ruleService *service.AuditRuleService, importServices ...*service.RuleImportService) *AuditRuleHandler {
	h := &AuditRuleHandler{ruleService: ruleService}
	if len(importServices) > 0 {
		h.importService = importServices[0]
	}
	return h
}

// ImportCapability 返回文件识别导入是否可用及文件限制。
// GET /api/tenant/rules/audit-rules/import-capability
func (h *AuditRuleHandler) ImportCapability(c *gin.Context) {
	if h.importService == nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "规则导入服务未初始化")
		return
	}
	capability, err := h.importService.Capability()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, capability)
}

// PreviewImport 识别上传文件并返回 AI 生成的审核规则草稿，不直接写库。
// POST /api/tenant/rules/audit-rules/import-preview
// 表单参数：config_id、file。
func (h *AuditRuleHandler) PreviewImport(c *gin.Context) {
	if h.importService == nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "规则导入服务未初始化")
		return
	}
	capability, err := h.importService.Capability()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if !capability.Enabled {
		response.Error(c, http.StatusForbidden, errcode.ErrPermissionDenied, capability.Reason)
		return
	}
	maxBytes := int64(capability.MaxFileSizeMB+1) * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "请选择符合大小限制的文件")
		return
	}
	preview, err := h.importService.Preview(c, "audit", c.PostForm("config_id"), file)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, preview)
}

// PreviewPastedImport 将粘贴文本交给 AI 并返回审核规则草稿，不经过 MinerU。
// POST /api/tenant/rules/audit-rules/import-text-preview
func (h *AuditRuleHandler) PreviewPastedImport(c *gin.Context) {
	if h.importService == nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "规则导入服务未初始化")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024*1024)
	var req dto.PreviewPastedRuleImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	preview, err := h.importService.PreviewText(c, "audit", &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, preview)
}

// ConfirmImport 将管理员确认后的审核规则草稿批量写入规则库。
// POST /api/tenant/rules/audit-rules/import-confirm
func (h *AuditRuleHandler) ConfirmImport(c *gin.Context) {
	if h.importService == nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "规则导入服务未初始化")
		return
	}
	var req dto.ConfirmRuleImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	rules, err := h.importService.Confirm(c, "audit", &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, rules)
}

// List 查询指定审核配置下的规则列表，支持按作用域和启用状态过滤。
// GET /api/tenant/rules/audit-rules
// 查询参数：config_id（必填，UUID）、rule_scope（可选）、enabled（可选，true/false）
// 返回：规则列表数组。
func (h *AuditRuleHandler) List(c *gin.Context) {
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

// Create 创建新的审核规则。
// POST /api/tenant/rules/audit-rules
// 请求体：CreateAuditRuleRequest（规则内容、作用域、所属配置 ID 等）
// 返回：新建的规则对象。
func (h *AuditRuleHandler) Create(c *gin.Context) {
	var req dto.CreateAuditRuleRequest
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

// Update 更新指定审核规则。
// PUT /api/tenant/rules/audit-rules/:id
// 路径参数：id（UUID 格式）
// 请求体：UpdateAuditRuleRequest
// 返回：更新后的规则对象。
func (h *AuditRuleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateAuditRuleRequest
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

// Delete 删除指定审核规则。
// DELETE /api/tenant/rules/audit-rules/:id
// 路径参数：id（UUID 格式）
// 返回：null（成功时）。
func (h *AuditRuleHandler) Delete(c *gin.Context) {
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

// BatchDelete 批量删除当前审核配置下选中的规则。
// POST /api/tenant/rules/audit-rules/batch-delete
// 请求体：config_id、rule_ids（1–5000 个 UUID）。
func (h *AuditRuleHandler) BatchDelete(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 512*1024)
	var req dto.BatchDeleteRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	configID, err := uuid.Parse(req.ConfigID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "config_id 格式错误")
		return
	}
	ruleIDs := make([]uuid.UUID, 0, len(req.RuleIDs))
	for _, idStr := range req.RuleIDs {
		id, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "rule_ids 包含无效 UUID")
			return
		}
		ruleIDs = append(ruleIDs, id)
	}

	deletedCount, err := h.ruleService.BatchDelete(c, configID, ruleIDs)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, &dto.BatchDeleteRulesResponse{DeletedCount: deletedCount})
}
