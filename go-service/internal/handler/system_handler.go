// 系统设置处理器，负责 OA 连接、AI 模型、选项数据及系统配置的管理。
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

// SystemHandler 处理系统设置相关的 HTTP 请求（OA 连接、AI 模型、选项数据、系统配置）。
type SystemHandler struct {
	optionService       *service.OptionService
	oaConnectionService *service.OAConnectionService
	aiModelService      *service.AIModelService
	systemConfigService *service.SystemConfigService
}

// NewSystemHandler 创建系统设置处理器实例。
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

// ── 选项数据接口 ──────────────────────────────────────────────────────────

// ListOATypes 获取所有 OA 系统类型选项。
// GET /api/admin/system/options/oa-types
// 返回：OA 类型选项数组（value + label）。
func (h *SystemHandler) ListOATypes(c *gin.Context) {
	items, err := h.optionService.ListOATypes()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// ListDBDrivers 获取所有数据库驱动类型选项。
// GET /api/admin/system/options/db-drivers
// 返回：数据库驱动选项数组（value + label）。
func (h *SystemHandler) ListDBDrivers(c *gin.Context) {
	items, err := h.optionService.ListDBDrivers()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// ListAIDeployTypes 获取所有 AI 部署类型选项。
// GET /api/admin/system/options/ai-deploy-types
// 返回：AI 部署类型选项数组（value + label）。
func (h *SystemHandler) ListAIDeployTypes(c *gin.Context) {
	items, err := h.optionService.ListAIDeployTypes()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// ListAIProviders 获取所有 AI 服务商选项。
// GET /api/admin/system/options/ai-providers
// 返回：AI 服务商选项数组（value + label）。
func (h *SystemHandler) ListAIProviders(c *gin.Context) {
	items, err := h.optionService.ListAIProviders()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// ── OA 数据库连接管理 ─────────────────────────────────────────────────────

// ListOAConnections 获取所有 OA 数据库连接配置列表。
// GET /api/admin/system/oa-connections
// 返回：OA 连接配置数组。
func (h *SystemHandler) ListOAConnections(c *gin.Context) {
	items, err := h.oaConnectionService.List()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateOAConnection 创建新的 OA 数据库连接配置。
// POST /api/admin/system/oa-connections
// 请求体：CreateOAConnectionRequest（连接类型、主机、端口、数据库名等）
// 返回：新建的连接配置对象。
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

// UpdateOAConnection 更新指定 OA 数据库连接配置。
// PUT /api/admin/system/oa-connections/:id
// 路径参数：id（UUID 格式）
// 请求体：UpdateOAConnectionRequest
// 返回：更新后的连接配置对象。
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

// DeleteOAConnection 删除指定 OA 数据库连接配置。
// DELETE /api/admin/system/oa-connections/:id
// 路径参数：id（UUID 格式）
// 返回：null（成功时）。
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

// TestOAConnection 测试已保存的 OA 数据库连接是否可用。
// POST /api/admin/system/oa-connections/:id/test
// 路径参数：id（UUID 格式）
// 返回：{"success": true, "message": "连接测试成功"}。
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

// TestOAConnectionParams 使用请求体中的连接参数直接测试连接（用于新建/编辑模态框中的测试按钮）。
// POST /api/admin/system/oa-connections/test
// 请求体：CreateOAConnectionRequest（连接参数）
// 返回：{"success": true, "message": "连接测试成功"}。
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

// ── AI 模型配置管理 ───────────────────────────────────────────────────────

// ListAIModels 获取所有 AI 模型配置列表。
// GET /api/admin/system/ai-models
// 返回：AI 模型配置数组。
func (h *SystemHandler) ListAIModels(c *gin.Context) {
	items, err := h.aiModelService.List()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateAIModel 创建新的 AI 模型配置。
// POST /api/admin/system/ai-models
// 请求体：CreateAIModelRequest（服务商、模型名称、API Key 等）
// 返回：新建的模型配置对象。
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

// UpdateAIModel 更新指定 AI 模型配置。
// PUT /api/admin/system/ai-models/:id
// 路径参数：id（UUID 格式）
// 请求体：UpdateAIModelRequest
// 返回：更新后的模型配置对象。
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

// DeleteAIModel 删除指定 AI 模型配置。
// DELETE /api/admin/system/ai-models/:id
// 路径参数：id（UUID 格式）
// 返回：null（成功时）。
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

// TestAIModelConnection 使用请求体中的模型参数直接测试连接（用于新建模态框中的测试按钮）。
// POST /api/admin/system/ai-models/test
// 请求体：CreateAIModelRequest（模型参数）
// 返回：{"success": true, "message": "模型连接测试成功"}。
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

// TestAIModelConnectionById 根据已保存的模型 ID 测试连接（用于卡片上的测试按钮）。
// POST /api/admin/system/ai-models/:id/test
// 路径参数：id（UUID 格式）
// 返回：{"success": true, "message": "模型连接测试成功"}。
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

// ── 系统配置管理 ──────────────────────────────────────────────────────────

// GetSystemConfigs 获取所有系统配置项（键值对形式）。
// GET /api/admin/system/configs
// 返回：系统配置键值对数组。
func (h *SystemHandler) GetSystemConfigs(c *gin.Context) {
	configs, err := h.systemConfigService.GetAll()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, configs)
}

// UpdateSystemConfigs 批量更新系统配置项。
// PUT /api/admin/system/configs
// 请求体：map[string]string（键值对，值不合法时返回 400）
// 返回：null（成功时）。
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
