package dto

import "gorm.io/datatypes"

// ===================== 用户个人配置 DTO =====================

// UpdateUserProcessConfigRequest 更新用户流程个性化配置请求
type UpdateUserProcessConfigRequest struct {
	CustomRules         []CustomRuleDTO        `json:"custom_rules"`
	FieldOverrides      []string               `json:"field_overrides"`
	FieldMode           string                 `json:"field_mode"`
	StrictnessOverride  string                 `json:"strictness_override"`
	RuleToggleOverrides []RuleToggleOverrideDTO `json:"rule_toggle_overrides"`
}

// CustomRuleDTO 用户自定义规则 DTO
type CustomRuleDTO struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Enabled bool   `json:"enabled"`
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

// ===================== 仪表板偏好 DTO =====================

// UpdateDashboardPrefRequest 更新仪表板偏好请求
type UpdateDashboardPrefRequest struct {
	EnabledWidgets datatypes.JSON `json:"enabled_widgets"`
	WidgetSizes    datatypes.JSON `json:"widget_sizes"`
}
