package dto

import "gorm.io/datatypes"

// ===================== Cron 任务类型预设 DTO =====================

// CronTaskTypePresetResponse 系统预设响应（前端展示用）
type CronTaskTypePresetResponse struct {
	TaskType        string         `json:"task_type"`
	Module          string         `json:"module"`
	LabelZh         string         `json:"label_zh"`
	LabelEn         string         `json:"label_en"`
	DescriptionZh   string         `json:"description_zh"`
	DescriptionEn   string         `json:"description_en"`
	DefaultCron     string         `json:"default_cron"`
	PushFormat      string         `json:"push_format"`
	ContentTemplate datatypes.JSON `json:"content_template"`
	SortOrder       int            `json:"sort_order"`
}

// CronTaskTypeConfigResponse 合并后的 Cron 任务类型配置响应（预设 + 租户覆盖）
type CronTaskTypeConfigResponse struct {
	TaskType        string         `json:"task_type"`
	Module          string         `json:"module"`
	LabelZh         string         `json:"label_zh"`
	LabelEn         string         `json:"label_en"`
	DescriptionZh   string         `json:"description_zh"`
	DescriptionEn   string         `json:"description_en"`
	// 预设默认值（供"恢复默认"使用）
	DefaultCron          string         `json:"default_cron"`
	PresetPushFormat     string         `json:"preset_push_format"`
	PresetContentTemplate datatypes.JSON `json:"preset_content_template"`
	SortOrder            int            `json:"sort_order"`
	// 租户当前配置（若未启用则为 null）
	IsEnabled       bool           `json:"is_enabled"`                  // 租户是否已启用该任务类型
	PushFormat      string         `json:"push_format"`                 // 当前生效的推送格式
	ContentTemplate datatypes.JSON `json:"content_template"`            // 当前生效的内容模板
	BatchLimit      *int           `json:"batch_limit"`                 // 当前批处理限制
}

// SaveCronTaskTypeConfigRequest 保存（启用/更新）定时任务类型配置请求
type SaveCronTaskTypeConfigRequest struct {
	PushFormat      string         `json:"push_format"`
	ContentTemplate datatypes.JSON `json:"content_template"`
	BatchLimit      *int           `json:"batch_limit"`
}

// ===================== 归档复盘配置 DTO =====================

// CreateProcessArchiveConfigRequest 创建归档复盘配置请求
type CreateProcessArchiveConfigRequest struct {
	ProcessType      string         `json:"process_type" binding:"required"`
	ProcessTypeLabel string         `json:"process_type_label"`
	MainTableName    string         `json:"main_table_name"`
	MainFields       datatypes.JSON `json:"main_fields"`
	DetailTables     datatypes.JSON `json:"detail_tables"`
	FieldMode        string         `json:"field_mode"`
	KBMode           string         `json:"kb_mode"`
	AIConfig         datatypes.JSON `json:"ai_config"`
	UserPermissions  datatypes.JSON `json:"user_permissions"`
	AccessControl    datatypes.JSON `json:"access_control"`
	Status           string         `json:"status"`
}

// UpdateProcessArchiveConfigRequest 更新归档复盘配置请求
type UpdateProcessArchiveConfigRequest struct {
	ProcessType      string         `json:"process_type"`
	ProcessTypeLabel string         `json:"process_type_label"`
	MainTableName    string         `json:"main_table_name"`
	MainFields       datatypes.JSON `json:"main_fields"`
	DetailTables     datatypes.JSON `json:"detail_tables"`
	FieldMode        string         `json:"field_mode"`
	KBMode           string         `json:"kb_mode"`
	AIConfig         datatypes.JSON `json:"ai_config"`
	UserPermissions  datatypes.JSON `json:"user_permissions"`
	AccessControl    datatypes.JSON `json:"access_control"`
	Status           string         `json:"status"`
}

// ===================== 归档规则 DTO =====================

// CreateArchiveRuleRequest 创建归档规则请求
type CreateArchiveRuleRequest struct {
	ConfigID    string `json:"config_id"`
	ProcessType string `json:"process_type" binding:"required"`
	RuleContent string `json:"rule_content" binding:"required"`
	RuleScope   string `json:"rule_scope"`
	Enabled     *bool  `json:"enabled"`
	Source      string `json:"source"`
	RelatedFlow bool   `json:"related_flow"`
}

// UpdateArchiveRuleRequest 更新归档规则请求
type UpdateArchiveRuleRequest struct {
	RuleContent string `json:"rule_content"`
	RuleScope   string `json:"rule_scope"`
	Enabled     *bool  `json:"enabled"`
	RelatedFlow *bool  `json:"related_flow"`
}

// ListArchiveRulesQuery 归档规则列表查询参数
type ListArchiveRulesQuery struct {
	ConfigID  string `form:"config_id" binding:"required"`
	RuleScope string `form:"rule_scope"`
	Enabled   *bool  `form:"enabled"`
}
