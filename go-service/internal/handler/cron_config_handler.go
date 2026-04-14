// 定时任务类型配置处理器，负责任务类型配置的查询、保存和重置。
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

// NewCronConfigHandler 创建定时任务类型配置处理器实例。
func NewCronConfigHandler(cronService *service.CronConfigService) *CronConfigHandler {
	return &CronConfigHandler{cronService: cronService}
}

// ListConfigs 获取当前租户所有任务类型的配置列表（预设与租户覆盖合并后的结果）。
// GET /api/tenant/cron/configs
// 返回：6 个任务类型配置数组，每项包含 cron 表达式、启用状态等。
func (h *CronConfigHandler) ListConfigs(c *gin.Context) {
	configs, err := h.cronService.ListConfigs(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, configs)
}

// SaveConfig 启用或更新指定任务类型的配置（租户级覆盖）。
// PUT /api/tenant/cron/configs/:taskType
// 路径参数：taskType（任务类型标识）
// 请求体：SaveCronTaskTypeConfigRequest（cron 表达式、启用状态等）
// 返回：保存后的配置对象。
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

// ResetConfig 将指定任务类型的配置重置为系统预设（删除租户覆盖配置）。
// DELETE /api/tenant/cron/configs/:taskType
// 路径参数：taskType（任务类型标识）
// 返回：重置后的预设配置对象。
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
