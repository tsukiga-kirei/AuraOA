package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ProcessAuditConfig 流程审核配置，租户级别的审核流程定义。
type ProcessAuditConfig struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID         uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	ProcessType      string         `gorm:"size:200;not null" json:"process_type"`
	ProcessTypeLabel string         `gorm:"size:200;default:''" json:"process_type_label"`
	MainTableName    string         `gorm:"size:200;default:''" json:"main_table_name"`
	MainFields       datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"main_fields"`
	DetailTables     datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"detail_tables"`
	FieldMode        string         `gorm:"size:20;not null;default:all" json:"field_mode"`
	KBMode           string         `gorm:"column:kb_mode;size:20;not null;default:rules_only" json:"kb_mode"`
	AIConfig         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"ai_config"`
	UserPermissions  datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"user_permissions"`
	AccessControl    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"access_control"`
	EmbedEnabled     bool           `gorm:"not null;default:false" json:"embed_enabled"`
	EmbedConfig      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"embed_config"`
	Status           string         `gorm:"size:20;not null;default:active" json:"status"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

func (ProcessAuditConfig) TableName() string { return "process_audit_configs" }

// AIConfigData AI配置的结构化表示
type AIConfigData struct {
	AuditStrictness        string `json:"audit_strictness"`
	EnableThinking         bool   `json:"enable_thinking"`
	SystemReasoningPrompt  string `json:"system_reasoning_prompt"`
	SystemExtractionPrompt string `json:"system_extraction_prompt"`
	UserReasoningPrompt    string `json:"user_reasoning_prompt"`
	UserExtractionPrompt   string `json:"user_extraction_prompt"`
}

// EmbedConfigData OA 嵌入页行为配置。
type EmbedConfigData struct {
	AutoAuditOnOpen           bool `json:"auto_audit_on_open"`
	AutoAuditOnStale          bool `json:"auto_audit_on_stale,omitempty"` // 兼容旧配置，不再直接用于决策
	AutoAuditOnDataChange     bool `json:"auto_audit_on_data_change"`
	AutoAuditOnReturnResubmit bool `json:"auto_audit_on_return_resubmit"`
	AutoAuditOnFlowChange     bool `json:"auto_audit_on_flow_change"`
	ScheduledRefreshEnabled   bool `json:"scheduled_refresh_enabled"`
	ScheduledLookbackDays     int  `json:"scheduled_refresh_lookback_days"`
	ScheduledIntervalMinutes  int  `json:"scheduled_refresh_interval_minutes"`
}

// UserPermissionsData 用户权限配置的结构化表示
type UserPermissionsData struct {
	AllowCustomFields     bool `json:"allow_custom_fields"`
	AllowCustomRules      bool `json:"allow_custom_rules"`
	AllowModifyStrictness bool `json:"allow_modify_strictness"`
}
