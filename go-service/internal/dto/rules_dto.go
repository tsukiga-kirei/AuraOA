package dto

import "gorm.io/datatypes"

// ===================== 流程审核配置 DTO =====================

// CreateProcessAuditConfigRequest 创建流程审核配置请求。
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
	AccessControl    datatypes.JSON `json:"access_control"`
	EmbedEnabled     *bool          `json:"embed_enabled"`
	EmbedConfig      datatypes.JSON `json:"embed_config"`
	Status           string         `json:"status"`
}

// UpdateProcessAuditConfigRequest 更新流程审核配置请求。
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
	AccessControl    datatypes.JSON `json:"access_control"`
	EmbedEnabled     *bool          `json:"embed_enabled"`
	EmbedConfig      datatypes.JSON `json:"embed_config"`
	Status           string         `json:"status"`
}

// TestConnectionRequest 测试 OA 流程连接请求。
type TestConnectionRequest struct {
	ProcessType      string `json:"process_type" binding:"required"`
	ProcessTypeLabel string `json:"process_type_label"` // 可选，用于校验流程类型
	MainTableName    string `json:"main_table_name"`    // 可选，用于校验主表名
}

// ===================== 审核规则 DTO =====================

// CreateAuditRuleRequest 创建审核规则请求。
type CreateAuditRuleRequest struct {
	ConfigID       string         `json:"config_id"`
	ProcessType    string         `json:"process_type" binding:"required"`
	RuleContent    string         `json:"rule_content" binding:"required"`
	RuleScope      string         `json:"rule_scope"`
	Enabled        *bool          `json:"enabled"`
	Source         string         `json:"source"`
	RelatedFlow    bool           `json:"related_flow"`
	ContextEnabled bool           `json:"context_enabled"`
	ContextMounts  datatypes.JSON `json:"context_mounts"`
}

// UpdateAuditRuleRequest 更新审核规则请求。
type UpdateAuditRuleRequest struct {
	RuleContent    string         `json:"rule_content"`
	RuleScope      string         `json:"rule_scope"`
	Enabled        *bool          `json:"enabled"`
	RelatedFlow    *bool          `json:"related_flow"`
	ContextEnabled *bool          `json:"context_enabled"`
	ContextMounts  datatypes.JSON `json:"context_mounts"`
}

// BatchDeleteRulesRequest 批量删除规则请求。
type BatchDeleteRulesRequest struct {
	ConfigID string   `json:"config_id" binding:"required,uuid"`
	RuleIDs  []string `json:"rule_ids" binding:"required,min=1,max=5000,dive,uuid"`
}

// BatchDeleteRulesResponse 批量删除规则响应。
type BatchDeleteRulesResponse struct {
	DeletedCount int64 `json:"deleted_count"`
}

// RuleImportCapabilityResponse 文件识别导入能力状态。
type RuleImportCapabilityResponse struct {
	Enabled        bool     `json:"enabled"`
	MaxFileSizeMB  int      `json:"max_file_size_mb"`
	SupportedTypes []string `json:"supported_types"`
	Reason         string   `json:"reason,omitempty"`
}

// RuleImportDraft AI 从制度文件中提取的单条规则草稿。
// AI 仅给出建议值，最终由租户管理员确认后才会写入规则库。
type RuleImportDraft struct {
	RuleContent        string  `json:"rule_content" binding:"required"`
	RuleScope          string  `json:"rule_scope"`
	RelatedFlow        bool    `json:"related_flow"`
	ContextRecommended bool    `json:"context_recommended"`
	Confidence         float64 `json:"confidence"`
	Reasoning          string  `json:"reasoning"`
}

// RuleImportPreviewResponse 文件识别与 AI 结构化后的预览结果。
type RuleImportPreviewResponse struct {
	FileName string            `json:"file_name"`
	Rules    []RuleImportDraft `json:"rules"`
	Warnings []string          `json:"warnings"`
}

// ConfirmRuleImportRequest 确认批量导入规则请求。
type ConfirmRuleImportRequest struct {
	ConfigID string            `json:"config_id" binding:"required"`
	Source   string            `json:"source" binding:"omitempty,oneof=file_import paste_import"`
	Rules    []RuleImportDraft `json:"rules" binding:"required,min=1,max=100,dive"`
}

// PreviewPastedRuleImportRequest 粘贴文本生成规则草稿请求。
type PreviewPastedRuleImportRequest struct {
	ConfigID string `json:"config_id" binding:"required"`
	Text     string `json:"text" binding:"required"`
}

// ===================== Token 统计 DTO =====================

// TokenUsageQuery Token 消耗查询参数。
type TokenUsageQuery struct {
	StartTime     string `form:"start_time" binding:"required"`
	EndTime       string `form:"end_time" binding:"required"`
	ModelConfigID string `form:"model_config_id"`
}
