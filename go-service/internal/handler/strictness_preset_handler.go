package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

// StrictnessPresetHandler 处理审核尺度预设相关的 HTTP 请求。
type StrictnessPresetHandler struct {
	presetService *service.StrictnessPresetService
}

// NewStrictnessPresetHandler 创建一个新的 StrictnessPresetHandler 实例。
func NewStrictnessPresetHandler(presetService *service.StrictnessPresetService) *StrictnessPresetHandler {
	return &StrictnessPresetHandler{presetService: presetService}
}

// List 处理 GET /api/tenant/rules/strictness-presets
func (h *StrictnessPresetHandler) List(c *gin.Context) {
	presets, err := h.presetService.ListByTenant(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, presets)
}

// Update 处理 PUT /api/tenant/rules/strictness-presets/:strictness
func (h *StrictnessPresetHandler) Update(c *gin.Context) {
	strictness := c.Param("strictness")
	if strictness == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateStrictnessPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	preset, err := h.presetService.UpdateByStrictness(c, strictness, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, preset)
}
