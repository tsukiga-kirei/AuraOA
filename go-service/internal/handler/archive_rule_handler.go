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

// NewArchiveRuleHandler 创建一个新的 ArchiveRuleHandler 实例。
func NewArchiveRuleHandler(ruleService *service.ArchiveRuleService) *ArchiveRuleHandler {
	return &ArchiveRuleHandler{ruleService: ruleService}
}

// List 处理 GET /api/tenant/archive/rules
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

// Create 处理 POST /api/tenant/archive/rules
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

// Update 处理 PUT /api/tenant/archive/rules/:id
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

// Delete 处理 DELETE /api/tenant/archive/rules/:id
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
