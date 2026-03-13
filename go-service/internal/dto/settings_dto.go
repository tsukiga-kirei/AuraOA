package dto

import "gorm.io/datatypes"

// ===================== 用户个人配置 DTO =====================

// UpdateUserProcessConfigRequest 更新用户流程个性化配置请求
type UpdateUserProcessConfigRequest struct {
	ConfigID    string              `json:"config_id"`
	FieldConfig UserFieldConfigDTO  `json:"field_config"`
	RuleConfig  UserRuleConfigDTO   `json:"rule_config"`
	AIConfig    UserAIConfigDTO     `json:"ai_config"`
}

// UserFieldConfigDTO 用户字段配置 DTO
type UserFieldConfigDTO struct {
	FieldMode      string   `json:"field_mode"`
	FieldOverrides []string `json:"field_overrides"`
}

// UserRuleConfigDTO 用户规则配置 DTO
type UserRuleConfigDTO struct {
	CustomRules         []CustomRuleDTO         `json:"custom_rules"`
	RuleToggleOverrides []RuleToggleOverrideDTO `json:"rule_toggle_overrides"`
}

// UserAIConfigDTO 用户 AI 个性化配置 DTO
type UserAIConfigDTO struct {
	StrictnessOverride string `json:"strictness_override"`
}

// CustomRuleDTO 用户自定义规则 DTO
type CustomRuleDTO struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	Enabled     bool   `json:"enabled"`
	RelatedFlow bool   `json:"related_flow"`
}

// RuleToggleOverrideDTO 规则开关覆盖 DTO
type RuleToggleOverrideDTO struct {
	RuleID  string `json:"rule_id"`
	Enabled bool   `json:"enabled"`
}

// ProcessListItem 用户可见的流程列表项
type ProcessListItem struct {
	ProcessType      string `json:"process_type"`
	ProcessTypeLabel string `json:"process_type_label"`
	ConfigID         string `json:"config_id"`
}

// ===================== 合并视图 DTO（工作台/归档完整配置） =====================

// TenantFieldDTO 租户字段配置项（含用户选中状态）
type TenantFieldDTO struct {
	FieldKey   string `json:"field_key"`
	FieldName  string `json:"field_name"`
	FieldType  string `json:"field_type"`
	Selected   bool   `json:"selected"` // 是否被用户选中
}

// DetailTableDTO 明细表配置（含字段选中状态）
type DetailTableDTO struct {
	TableName  string           `json:"table_name"`
	TableLabel string           `json:"table_label"`
	Fields     []TenantFieldDTO `json:"fields"`
}

// TenantRuleDTO 租户规则（含用户开关状态）
type TenantRuleDTO struct {
	ID          string `json:"id"`
	RuleContent string `json:"rule_content"`
	RuleScope   string `json:"rule_scope"`
	RelatedFlow bool   `json:"related_flow"`
	Enabled     bool   `json:"enabled"` // 已应用用户开关覆盖后的有效状态
}

// UserPermissionsDTO 用户权限配置
type UserPermissionsDTO struct {
	AllowCustomFields     bool `json:"allow_custom_fields"`
	AllowCustomRules      bool `json:"allow_custom_rules"`
	AllowModifyStrictness bool `json:"allow_modify_strictness"`
}

// AIConfigDTO AI配置
type AIConfigDTO struct {
	AuditStrictness string `json:"audit_strictness"`
	KBMode          string `json:"kb_mode"`
}

// FullAuditProcessConfigResponse 审核工作台完整配置响应（租户配置+用户覆盖合并）
type FullAuditProcessConfigResponse struct {
	ProcessType      string             `json:"process_type"`
	ProcessTypeLabel string             `json:"process_type_label"`
	ConfigID         string             `json:"config_id"`
	FieldMode        string             `json:"field_mode"`         // 租户设置的字段传输模式
	KBMode           string             `json:"kb_mode"`            // 租户设置的知识库模式
	AuditStrictness  string             `json:"audit_strictness"`   // 有效严格度（用户覆盖优先）
	UserPermissions  UserPermissionsDTO `json:"user_permissions"`   // 用户权限
	MainFields       []TenantFieldDTO   `json:"main_fields"`        // 主表字段（含选中状态）
	DetailTables     []DetailTableDTO   `json:"detail_tables"`      // 明细表（含字段选中状态）
	TenantRules      []TenantRuleDTO    `json:"tenant_rules"`       // 租户规则（含开关状态）
	CustomRules      []CustomRuleDTO    `json:"custom_rules"`       // 用户自定义规则
}

// ===================== Cron 偏好 DTO =====================

// CronPrefsResponse 用户定时任务个人偏好响应
type CronPrefsResponse struct {
	DefaultEmail string `json:"default_email"`
}

// UpdateCronPrefsRequest 更新用户定时任务个人偏好请求
type UpdateCronPrefsRequest struct {
	DefaultEmail string `json:"default_email"`
}

// ===================== 归档复盘用户端 DTO =====================

// ArchiveUserPermissionsDTO 归档复盘用户权限配置
type ArchiveUserPermissionsDTO struct {
	AllowCustomFields     bool `json:"allow_custom_fields"`
	AllowCustomRules      bool `json:"allow_custom_rules"`
	AllowModifyStrictness bool `json:"allow_modify_strictness"`
}

// AccessibleArchiveConfigItem 用户可访问的归档配置列表项
type AccessibleArchiveConfigItem struct {
	ProcessType      string `json:"process_type"`
	ProcessTypeLabel string `json:"process_type_label"`
	ConfigID         string `json:"config_id"`
}

// FullArchiveConfigResponse 归档复盘完整配置响应（租户配置+用户覆盖合并）
type FullArchiveConfigResponse struct {
	ProcessType      string                    `json:"process_type"`
	ProcessTypeLabel string                    `json:"process_type_label"`
	ConfigID         string                    `json:"config_id"`
	FieldMode        string                    `json:"field_mode"`
	KBMode           string                    `json:"kb_mode"`
	AuditStrictness  string                    `json:"audit_strictness"`
	UserPermissions  ArchiveUserPermissionsDTO `json:"user_permissions"`
	MainFields       []TenantFieldDTO          `json:"main_fields"`
	DetailTables     []DetailTableDTO          `json:"detail_tables"`
	TenantRules      []TenantRuleDTO           `json:"tenant_rules"`
	CustomRules      []CustomRuleDTO           `json:"custom_rules"`
}

// UpdateArchiveConfigRequest 更新用户归档复盘个人配置请求
type UpdateArchiveConfigRequest struct {
	ConfigID    string              `json:"config_id"`
	FieldConfig UserFieldConfigDTO  `json:"field_config"`
	RuleConfig  UserRuleConfigDTO   `json:"rule_config"`
	AIConfig    UserAIConfigDTO     `json:"ai_config"`
}

// ===================== 仪表板偏好 DTO =====================

// UpdateDashboardPrefRequest 更新仪表板偏好请求
type UpdateDashboardPrefRequest struct {
	EnabledWidgets datatypes.JSON `json:"enabled_widgets"`
	WidgetSizes    datatypes.JSON `json:"widget_sizes"`
}
