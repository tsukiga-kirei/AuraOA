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
