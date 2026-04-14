// 归档规则处理器，负责归档复盘规则的增删改查。
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

// ArchiveRuleHandler 处理归档规则相关的 HTTP 请求。
type ArchiveRuleHandler struct {
	ruleService *service.ArchiveRuleService
}

// NewArchiveRuleHandler 创建归档规则处理器实例。
func NewArchiveRuleHandler(ruleService *service.ArchiveRuleService) *ArchiveRuleHandler {
	return &ArchiveRuleHandler{ruleService: ruleService}
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
