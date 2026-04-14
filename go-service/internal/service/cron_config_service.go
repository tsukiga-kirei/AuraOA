package service

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// CronConfigService 处理定时任务类型配置的业务逻辑。
type CronConfigService struct {
	presetRepo *repository.CronTaskTypePresetRepo
	configRepo *repository.CronTaskTypeConfigRepo
}

// NewCronConfigService 创建一个新的 CronConfigService 实例。
func NewCronConfigService(
	presetRepo *repository.CronTaskTypePresetRepo,
	configRepo *repository.CronTaskTypeConfigRepo,
) *CronConfigService {
	return &CronConfigService{
		presetRepo: presetRepo,
		configRepo: configRepo,
	}
}

// ListConfigs 返回所有 6 个任务类型的当前配置合并结果（预设 + 租户覆盖）。
// 若租户未启用某类型，is_enabled=false，配置值取自预设默认值。
func (s *CronConfigService) ListConfigs(c *gin.Context) ([]dto.CronTaskTypeConfigResponse, error) {
	// 1. 获取所有系统预设
	presets, err := s.presetRepo.ListAll()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 2. 获取当前租户已启用的配置
	tenantConfigs, err := s.configRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 3. 构建租户配置 map（key: task_type → config）
	configMap := make(map[string]model.CronTaskTypeConfig)
	for _, cfg := range tenantConfigs {
		configMap[cfg.TaskType] = cfg
	}

	// 4. 合并预设和租户配置
	result := make([]dto.CronTaskTypeConfigResponse, 0, len(presets))
	for _, preset := range presets {
		resp := dto.CronTaskTypeConfigResponse{
			TaskType:              preset.TaskType,
			Module:                preset.Module,
			LabelZh:               preset.LabelZh,
			LabelEn:               preset.LabelEn,
			DescriptionZh:         preset.DescriptionZh,
			DescriptionEn:         preset.DescriptionEn,
			DefaultCron:           preset.DefaultCron,
			PresetPushFormat:      preset.PushFormat,
			PresetContentTemplate: datatypes.JSON(preset.ContentTemplate),
			SortOrder:             preset.SortOrder,
			// 默认：未启用
			IsEnabled:       false,
			PushFormat:      preset.PushFormat,
			ContentTemplate: datatypes.JSON(preset.ContentTemplate),
		}

		// 如果租户已有覆盖配置，合并进来
		if cfg, ok := configMap[preset.TaskType]; ok {
			resp.IsEnabled = true
			resp.PushFormat = cfg.PushFormat
			resp.ContentTemplate = cfg.ContentTemplate
			resp.BatchLimit = cfg.BatchLimit
		}

		result = append(result, resp)
	}

	return result, nil
}

// SaveConfig 保存（启用/更新）租户任务类型配置。
func (s *CronConfigService) SaveConfig(c *gin.Context, taskType string, req *dto.SaveCronTaskTypeConfigRequest) (*dto.CronTaskTypeConfigResponse, error) {
	// 校验任务类型是否存在于系统预设
	preset, err := s.presetRepo.GetByTaskType(taskType)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, fmt.Sprintf("任务类型 %s 不存在", taskType))
	}

	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 处理内容模板：若未提供则使用预设
	contentTemplate := req.ContentTemplate
	if contentTemplate == nil || string(contentTemplate) == "null" || len(contentTemplate) == 0 {
		contentTemplate = datatypes.JSON(preset.ContentTemplate)
	}

	cfg := &model.CronTaskTypeConfig{
		TenantID:        tenantID,
		TaskType:        taskType,
		PushFormat:      defaultStr(req.PushFormat, preset.PushFormat),
		ContentTemplate: contentTemplate,
		BatchLimit:      req.BatchLimit,
	}

	// Upsert：已有则更新，没有则创建
	if err := s.configRepo.Upsert(c, cfg); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 返回合并后的配置
	resp := &dto.CronTaskTypeConfigResponse{
		TaskType:              preset.TaskType,
		Module:                preset.Module,
		LabelZh:               preset.LabelZh,
		LabelEn:               preset.LabelEn,
		DescriptionZh:         preset.DescriptionZh,
		DescriptionEn:         preset.DescriptionEn,
		DefaultCron:           preset.DefaultCron,
		PresetPushFormat:      preset.PushFormat,
		PresetContentTemplate: datatypes.JSON(preset.ContentTemplate),
		SortOrder:             preset.SortOrder,
		IsEnabled:             true,
		PushFormat:            cfg.PushFormat,
		ContentTemplate:       cfg.ContentTemplate,
		BatchLimit:            cfg.BatchLimit,
	}

	return resp, nil
}

// ResetConfig 重置租户任务类型配置（删除覆盖记录，回到系统预设）。
func (s *CronConfigService) ResetConfig(c *gin.Context, taskType string) (*dto.CronTaskTypeConfigResponse, error) {
	// 校验任务类型是否存在
	preset, err := s.presetRepo.GetByTaskType(taskType)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, fmt.Sprintf("任务类型 %s 不存在", taskType))
	}

	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 删除租户覆盖配置（如果存在）
	if err := s.configRepo.Delete(c, tenantID, taskType); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 返回预设值（恢复默认状态）
	resp := &dto.CronTaskTypeConfigResponse{
		TaskType:              preset.TaskType,
		Module:                preset.Module,
		LabelZh:               preset.LabelZh,
		LabelEn:               preset.LabelEn,
		DescriptionZh:         preset.DescriptionZh,
		DescriptionEn:         preset.DescriptionEn,
		DefaultCron:           preset.DefaultCron,
		PresetPushFormat:      preset.PushFormat,
		PresetContentTemplate: datatypes.JSON(preset.ContentTemplate),
		SortOrder:             preset.SortOrder,
		IsEnabled:             false,
		PushFormat:            preset.PushFormat,
		ContentTemplate:       datatypes.JSON(preset.ContentTemplate),
	}

	return resp, nil
}

// marshalJSON 将任意对象序列化为 datatypes.JSON（工具函数）。
func marshalJSON(v interface{}) datatypes.JSON {
	b, _ := json.Marshal(v)
	return datatypes.JSON(b)
}
