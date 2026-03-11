package dto

import "gorm.io/datatypes"

// ===================== 流程审核配置 DTO =====================

// CreateProcessAuditConfigRequest 创建流程审核配置请求
type CreateProcessAuditConfigRequest struct {
	ProcessType      string         `json:"process_type" binding:"required"`
	ProcessTypeLabel string         `json:"process_type_label"`
	MainTableName    string         `json:"main_table_name"`
	MainFields       datatypes.JSON `json:"main_fields"`
	DetailTables     datatypes.JSON `json:"detail_tables"`
	FieldMode        string         `json:"field_mode"`
	KBMode           string         `json:"kb_mode"`
	AIConfig         datatypes.JSON `json:"ai_config"`
	UserPermissions  datatypes.JSON `json:"user_permissions"`
	Status           string         `json:"status"`
}

// UpdateProcessAuditConfigRequest 更新流程审核配置请求
type UpdateProcessAuditConfigRequest struct {
	ProcessType      string         `json:"process_type"`
	ProcessTypeLabel string         `json:"process_type_label"`
	MainTableName    string         `json:"main_table_name"`
	MainFields       datatypes.JSON `json:"main_fields"`
	DetailTables     datatypes.JSON `json:"detail_tables"`
	FieldMode        string         `json:"field_mode"`
	KBMode           string         `json:"kb_mode"`
	AIConfig         datatypes.JSON `json:"ai_config"`
	UserPermissions  datatypes.JSON `json:"user_permissions"`
	Status           string         `json:"status"`
}

// TestConnectionRequest 测试 OA 流程连接请求
type TestConnectionRequest struct {
	ProcessType      string `json:"process_type" binding:"required"`
	ProcessTypeLabel string `json:"process_type_label"` // 可选，用于校验流程类型是否正确
	MainTableName    string `json:"main_table_name"` // 可选，用于校验主表名是否正确
}

// ===================== 审核规则 DTO =====================

// CreateAuditRuleRequest 创建审核规则请求
type CreateAuditRuleRequest struct {
	ConfigID    string `json:"config_id"`
	ProcessType string `json:"process_type" binding:"required"`
	RuleContent string `json:"rule_content" binding:"required"`
	RuleScope   string `json:"rule_scope"`
	Enabled     *bool  `json:"enabled"`
	Source      string `json:"source"`
	RelatedFlow bool   `json:"related_flow"`
}

// UpdateAuditRuleRequest 更新审核规则请求
type UpdateAuditRuleRequest struct {
	RuleContent string `json:"rule_content"`
	RuleScope   string `json:"rule_scope"`
	Enabled     *bool  `json:"enabled"`
	RelatedFlow *bool  `json:"related_flow"`
}

// ===================== 审核尺度预设 DTO =====================

// UpdateStrictnessPresetRequest 更新审核尺度预设请求
type UpdateStrictnessPresetRequest struct {
	ReasoningInstruction  string `json:"reasoning_instruction"`
	ExtractionInstruction string `json:"extraction_instruction"`
}

// ===================== Token 统计 DTO =====================

// TokenUsageQuery Token 消耗查询参数
type TokenUsageQuery struct {
	StartTime     string `form:"start_time" binding:"required"`
	EndTime       string `form:"end_time" binding:"required"`
	ModelConfigID string `form:"model_config_id"`
}
