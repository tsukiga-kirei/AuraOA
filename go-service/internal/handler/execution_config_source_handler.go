package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/service"
)

// ExecutionConfigSourceHandler 提供配置页所需的当前执行版本状态。
type ExecutionConfigSourceHandler struct {
	service *service.ExecutionConfigSourceService
}

func NewExecutionConfigSourceHandler(service *service.ExecutionConfigSourceService) *ExecutionConfigSourceHandler {
	return &ExecutionConfigSourceHandler{service: service}
}

// GetStatus 返回当前租户配置是否已进入某个不可变执行版本。
// GET /api/tenant/execution-config-versions/status?module=audit&source_config_id=...
func (h *ExecutionConfigSourceHandler) GetStatus(c *gin.Context) {
	configID, err := uuid.Parse(c.Query("source_config_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	status, err := h.service.GetStatus(c, c.Query("module"), configID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, status)
}

type publishConfigRequest struct {
	Module         string    `json:"module"`
	SourceConfigID uuid.UUID `json:"source_config_id"`
}

// Publish 固化当前配置并发布为新版本。
// POST /api/tenant/execution-config-versions/publish
func (h *ExecutionConfigSourceHandler) Publish(c *gin.Context) {
	var req publishConfigRequest
	_ = c.ShouldBindJSON(&req)

	module := req.Module
	if module == "" {
		module = c.Query("module")
	}
	configID := req.SourceConfigID
	if configID == uuid.Nil {
		parsed, err := uuid.Parse(c.Query("source_config_id"))
		if err == nil {
			configID = parsed
		}
	}
	if module == "" || configID == uuid.Nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	status, err := h.service.Publish(c, module, configID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, status)
}

// ListHistory 查询历史发布版本列表。
// GET /api/tenant/execution-config-versions/history?module=audit&source_config_id=...
func (h *ExecutionConfigSourceHandler) ListHistory(c *gin.Context) {
	configID, err := uuid.Parse(c.Query("source_config_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	items, err := h.service.ListHistory(c, c.Query("module"), configID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

type versionActionRequest struct {
	Module         string    `json:"module" binding:"required"`
	SourceConfigID uuid.UUID `json:"source_config_id" binding:"required"`
	VersionNo      int       `json:"version_no" binding:"required"`
}

// Activate 将指定版本切换为当前可用版本（Active Version）。
// POST /api/tenant/execution-config-versions/activate
func (h *ExecutionConfigSourceHandler) Activate(c *gin.Context) {
	var req versionActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	status, err := h.service.ActivateVersion(c, req.Module, req.SourceConfigID, req.VersionNo)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, status)
}

type saveVersionRequest struct {
	Module         string      `json:"module" binding:"required"`
	SourceConfigID uuid.UUID   `json:"source_config_id" binding:"required"`
	VersionNo      int         `json:"version_no" binding:"required"`
	Snapshot       interface{} `json:"snapshot" binding:"required"`
}

// SaveVersion 直接修改并保存指定版本的快照内容。
// POST /api/tenant/execution-config-versions/save-version
func (h *ExecutionConfigSourceHandler) SaveVersion(c *gin.Context) {
	var req saveVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	status, err := h.service.SaveVersion(c, req.Module, req.SourceConfigID, req.VersionNo, req.Snapshot)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, status)
}

