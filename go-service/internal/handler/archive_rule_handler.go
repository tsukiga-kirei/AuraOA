// 归档规则处理器，负责归档复盘规则的增删改查。
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

// ArchiveRuleHandler 处理归档规则相关的 HTTP 请求。
type ArchiveRuleHandler struct {
	ruleService   *service.ArchiveRuleService
	importService *service.RuleImportService
}

// NewArchiveRuleHandler 创建归档规则处理器实例。
func NewArchiveRuleHandler(ruleService *service.ArchiveRuleService, importServices ...*service.RuleImportService) *ArchiveRuleHandler {
	h := &ArchiveRuleHandler{ruleService: ruleService}
	if len(importServices) > 0 {
		h.importService = importServices[0]
	}
	return h
}

// ImportCapability 返回归档规则文件识别导入是否可用及文件限制。
// GET /api/tenant/archive/rules/import-capability
func (h *ArchiveRuleHandler) ImportCapability(c *gin.Context) {
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

// PreviewImport 识别上传文件并返回 AI 生成的归档规则草稿，不直接写库。
// POST /api/tenant/archive/rules/import-preview
// 表单参数：config_id、file。
func (h *ArchiveRuleHandler) PreviewImport(c *gin.Context) {
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
	preview, err := h.importService.Preview(c, "archive", c.PostForm("config_id"), file)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, preview)
}

// PreviewPastedImport 将粘贴文本交给 AI 并返回归档规则草稿，不经过 MinerU。
// POST /api/tenant/archive/rules/import-text-preview
func (h *ArchiveRuleHandler) PreviewPastedImport(c *gin.Context) {
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
	preview, err := h.importService.PreviewText(c, "archive", &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, preview)
}

// ConfirmImport 将管理员确认后的归档规则草稿批量写入规则库。
// POST /api/tenant/archive/rules/import-confirm
func (h *ArchiveRuleHandler) ConfirmImport(c *gin.Context) {
	if h.importService == nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "规则导入服务未初始化")
		return
	}
	var req dto.ConfirmRuleImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	rules, err := h.importService.Confirm(c, "archive", &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, rules)
}

// List 查询指定归档配置下的规则列表，支持按作用域和启用状态过滤。
// GET /api/tenant/archive/rules
// 查询参数：config_id（必填，UUID）、rule_scope（可选）、enabled（可选，true/false）
// 返回：规则列表数组。
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

// Create 创建新的归档规则。
// POST /api/tenant/archive/rules
// 请求体：CreateArchiveRuleRequest（规则内容、作用域、所属配置 ID 等）
// 返回：新建的规则对象。
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

// Update 更新指定归档规则。
// PUT /api/tenant/archive/rules/:id
// 路径参数：id（UUID 格式）
// 请求体：UpdateArchiveRuleRequest
// 返回：更新后的规则对象。
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

// Delete 删除指定归档规则。
// DELETE /api/tenant/archive/rules/:id
// 路径参数：id（UUID 格式）
// 返回：null（成功时）。
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
