// 归档复盘配置处理器，负责归档流程数据源配置的增删改查及连接测试。
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

// NewArchiveConfigHandler 创建归档复盘配置处理器实例。
func NewArchiveConfigHandler(archiveService *service.ProcessArchiveConfigService) *ArchiveConfigHandler {
	return &ArchiveConfigHandler{archiveService: archiveService}
}

// List 获取当前租户的所有归档复盘配置列表。
// GET /api/tenant/archive/configs
// 返回：配置列表数组，包含数据源连接信息和流程类型。
func (h *ArchiveConfigHandler) List(c *gin.Context) {
	cfgs, err := h.archiveService.List(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, cfgs)
}

// Create 创建新的归档复盘配置。
// POST /api/tenant/archive/configs
// 请求体：CreateProcessArchiveConfigRequest（流程类型、数据源连接参数等）
// 返回：新建的配置对象。
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

// GetByID 根据 ID 获取单条归档复盘配置详情。
// GET /api/tenant/archive/configs/:id
// 路径参数：id（UUID 格式）
// 返回：配置详情对象。
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

// Update 更新指定归档复盘配置。
// PUT /api/tenant/archive/configs/:id
// 路径参数：id（UUID 格式）
// 请求体：UpdateProcessArchiveConfigRequest
// 返回：更新后的配置对象。
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

// Delete 删除指定归档复盘配置。
// DELETE /api/tenant/archive/configs/:id
// 路径参数：id（UUID 格式）
// 返回：null（成功时）。
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

// TestConnection 测试归档数据源连接是否可用（不保存配置）。
// POST /api/tenant/archive/configs/test-connection
// 请求体：TestConnectionRequest（数据库连接参数）
// 返回：连接测试结果信息。
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

// FetchFields 从已保存的归档配置对应的数据源中拉取可用字段列表。
// POST /api/tenant/archive/configs/:id/fetch-fields
// 路径参数：id（UUID 格式）
// 返回：字段列表（主表字段 + 明细表字段）。
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

// ListPromptTemplates 获取归档复盘专用的系统提示词模板列表（archive_ 前缀）。
// GET /api/tenant/archive/prompt-templates
// 返回：提示词模板数组。
func (h *ArchiveConfigHandler) ListPromptTemplates(c *gin.Context) {
	templates, err := h.archiveService.ListArchivePromptTemplates()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, templates)
}
