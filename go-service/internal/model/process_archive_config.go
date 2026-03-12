package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ProcessArchiveConfig 归档复盘配置（租户级别）。
// 结构参考 ProcessAuditConfig，新增 AccessControl 字段用于访问权限控制。
type ProcessArchiveConfig struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"` // 主键UUID
	TenantID         uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`                       // 所属租户ID
	ProcessType      string         `gorm:"size:200;not null" json:"process_type"`                     // 流程类型标识
	ProcessTypeLabel string         `gorm:"size:200;default:''" json:"process_type_label"`             // 流程类型显示名称
	MainTableName    string         `gorm:"size:200;default:''" json:"main_table_name"`                // OA主表名称
	MainFields       datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"main_fields"`       // 主表字段配置列表
	DetailTables     datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"detail_tables"`     // 明细子表配置列表
	FieldMode        string         `gorm:"size:20;not null;default:all" json:"field_mode"`            // 字段提取模式：all/selected
	KBMode           string         `gorm:"column:kb_mode;size:20;not null;default:rules_only" json:"kb_mode"` // 知识库模式
	AIConfig         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"ai_config"`         // AI复核配置
	UserPermissions  datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"user_permissions"`  // 用户权限配置
	AccessControl    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"access_control"`    // 访问控制（roles/members/departments）
	Status           string         `gorm:"size:20;not null;default:active" json:"status"`             // 配置状态：active/inactive
	CreatedAt        time.Time      `json:"created_at"`                                                // 创建时间
	UpdatedAt        time.Time      `json:"updated_at"`                                                // 最后更新时间
}

func (ProcessArchiveConfig) TableName() string { return "process_archive_configs" }

// ArchiveAIConfigData AI配置的结构化表示（归档复盘版）
type ArchiveAIConfigData struct {
	AuditStrictness        string `json:"audit_strictness"`
	SystemReasoningPrompt  string `json:"system_reasoning_prompt"`
	SystemExtractionPrompt string `json:"system_extraction_prompt"`
	UserReasoningPrompt    string `json:"user_reasoning_prompt"`
	UserExtractionPrompt   string `json:"user_extraction_prompt"`
}

// ArchiveUserPermissionsData 归档复盘用户权限配置
type ArchiveUserPermissionsData struct {
	AllowCustomFields     bool `json:"allow_custom_fields"`
	AllowCustomRules      bool `json:"allow_custom_rules"`
	AllowCustomFlowRules  bool `json:"allow_custom_flow_rules"`
	AllowModifyStrictness bool `json:"allow_modify_strictness"`
}

// AccessControlData 访问控制配置（归档复盘专用）
type AccessControlData struct {
	AllowedRoles       []string `json:"allowed_roles"`
	AllowedMembers     []string `json:"allowed_members"`
	AllowedDepartments []string `json:"allowed_departments"`
}
