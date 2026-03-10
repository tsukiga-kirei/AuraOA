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

// SystemHandler 处理系统设置相关的 HTTP 请求（OA连接、AI模型、选项数据、系统配置）。
type SystemHandler struct {
	optionService       *service.OptionService
	oaConnectionService *service.OAConnectionService
	aiModelService      *service.AIModelService
	systemConfigService *service.SystemConfigService
}

// NewSystemHandler 创建一个新的 SystemHandler 实例。
func NewSystemHandler(
	optionService *service.OptionService,
	oaConnectionService *service.OAConnectionService,
	aiModelService *service.AIModelService,
	systemConfigService *service.SystemConfigService,
) *SystemHandler {
	return &SystemHandler{
		optionService:       optionService,
		oaConnectionService: oaConnectionService,
		aiModelService:      aiModelService,
		systemConfigService: systemConfigService,
	}
}

// ============================================================
// 选项数据接口
// ============================================================

// ListOATypes handles GET /api/admin/system/options/oa-types
func (h *SystemHandler) ListOATypes(c *gin.Context) {
	items, err := h.optionService.ListOATypes()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// ListDBDrivers handles GET /api/admin/system/options/db-drivers
func (h *SystemHandler) ListDBDrivers(c *gin.Context) {
	items, err := h.optionService.ListDBDrivers()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// ListAIDeployTypes handles GET /api/admin/system/options/ai-deploy-types
func (h *SystemHandler) ListAIDeployTypes(c *gin.Context) {
	items, err := h.optionService.ListAIDeployTypes()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// ListAIProviders handles GET /api/admin/system/options/ai-providers
func (h *SystemHandler) ListAIProviders(c *gin.Context) {
	items, err := h.optionService.ListAIProviders()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// ============================================================
// OA 数据库连接 CRUD
// ============================================================

// ListOAConnections handles GET /api/admin/system/oa-connections
func (h *SystemHandler) ListOAConnections(c *gin.Context) {
	items, err := h.oaConnectionService.List()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateOAConnection handles POST /api/admin/system/oa-connections
func (h *SystemHandler) CreateOAConnection(c *gin.Context) {
	var req dto.CreateOAConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	item, err := h.oaConnectionService.Create(&req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, item)
}

// UpdateOAConnection handles PUT /api/admin/system/oa-connections/:id
func (h *SystemHandler) UpdateOAConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateOAConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	item, err := h.oaConnectionService.Update(id, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, item)
}

// DeleteOAConnection handles DELETE /api/admin/system/oa-connections/:id
func (h *SystemHandler) DeleteOAConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.oaConnectionService.Delete(id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// TestOAConnection handles POST /api/admin/system/oa-connections/:id/test
func (h *SystemHandler) TestOAConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.oaConnectionService.TestConnection(id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, map[string]interface{}{
		"success": true,
		"message": "连接测试成功",
	})
}

// TestOAConnectionParams handles POST /api/admin/system/oa-connections/test
// 接受连接参数直接测试（用于新建/编辑模态框中的测试按钮）。
func (h *SystemHandler) TestOAConnectionParams(c *gin.Context) {
	var req dto.CreateOAConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.oaConnectionService.TestConnectionByParams(&req); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, map[string]interface{}{
		"success": true,
		"message": "连接测试成功",
	})
}

// ============================================================
// AI 模型配置 CRUD
// ============================================================

// ListAIModels handles GET /api/admin/system/ai-models
func (h *SystemHandler) ListAIModels(c *gin.Context) {
	items, err := h.aiModelService.List()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateAIModel handles POST /api/admin/system/ai-models
func (h *SystemHandler) CreateAIModel(c *gin.Context) {
	var req dto.CreateAIModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	item, err := h.aiModelService.Create(&req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, item)
}

// UpdateAIModel handles PUT /api/admin/system/ai-models/:id
func (h *SystemHandler) UpdateAIModel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateAIModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	item, err := h.aiModelService.Update(id, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, item)
}

// DeleteAIModel handles DELETE /api/admin/system/ai-models/:id
func (h *SystemHandler) DeleteAIModel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.aiModelService.Delete(id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// TestAIModelConnection handles POST /api/admin/system/ai-models/test
// 接受模型参数直接测试连接（用于新建模态框中的测试按钮）。
func (h *SystemHandler) TestAIModelConnection(c *gin.Context) {
	var req dto.CreateAIModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.aiModelService.TestConnectionByParams(&req); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, map[string]interface{}{
		"success": true,
		"message": "模型连接测试成功",
	})
}

// TestAIModelConnectionById handles POST /api/admin/system/ai-models/:id/test
// 根据已保存的模型 ID 测试连接（用于卡片上的测试按钮）。
func (h *SystemHandler) TestAIModelConnectionById(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.aiModelService.TestConnection(id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, map[string]interface{}{
		"success": true,
		"message": "模型连接测试成功",
	})
}

// ============================================================
// 系统配置
// ============================================================

// GetSystemConfigs handles GET /api/admin/system/configs
func (h *SystemHandler) GetSystemConfigs(c *gin.Context) {
	configs, err := h.systemConfigService.GetAll()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, configs)
}

// UpdateSystemConfigs handles PUT /api/admin/system/configs
func (h *SystemHandler) UpdateSystemConfigs(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.systemConfigService.UpdateConfigs(req); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}
