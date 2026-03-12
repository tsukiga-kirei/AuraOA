package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

// CronConfigHandler 处理定时任务类型配置相关的 HTTP 请求。
type CronConfigHandler struct {
	cronService *service.CronConfigService
}

// NewCronConfigHandler 创建一个新的 CronConfigHandler 实例。
func NewCronConfigHandler(cronService *service.CronConfigService) *CronConfigHandler {
	return &CronConfigHandler{cronService: cronService}
}

// ListConfigs 处理 GET /api/tenant/cron/configs
// 返回所有 6 个任务类型配置（预设 + 租户覆盖合并）
func (h *CronConfigHandler) ListConfigs(c *gin.Context) {
	configs, err := h.cronService.ListConfigs(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, configs)
}

// SaveConfig 处理 PUT /api/tenant/cron/configs/:taskType
// 启用或更新指定任务类型的配置
func (h *CronConfigHandler) SaveConfig(c *gin.Context) {
	taskType := c.Param("taskType")
	if taskType == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "taskType 参数必填")
		return
	}

	var req dto.SaveCronTaskTypeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	config, err := h.cronService.SaveConfig(c, taskType, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, config)
}

// ResetConfig 处理 DELETE /api/tenant/cron/configs/:taskType
// 重置指定任务类型为系统预设（关闭该任务类型配置）
func (h *CronConfigHandler) ResetConfig(c *gin.Context) {
	taskType := c.Param("taskType")
	if taskType == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "taskType 参数必填")
		return
	}

	config, err := h.cronService.ResetConfig(c, taskType)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, config)
}
