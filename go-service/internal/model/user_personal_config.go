package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// UserPersonalConfig 用户个人配置，按 tenant_id + user_id 唯一约束。
type UserPersonalConfig struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	AuditDetails   datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"audit_details"`
	CronDetails    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"cron_details"`
	ArchiveDetails datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"archive_details"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (UserPersonalConfig) TableName() string { return "user_personal_configs" }

// AuditDetailItem 用户审核配置中单个流程的个性化设置
type AuditDetailItem struct {
	ConfigID    uuid.UUID    `json:"config_id"`    // 对应 process_audit_configs.id
	ProcessType string       `json:"process_type"`
	FieldConfig FieldConfig  `json:"field_config"`
	RuleConfig  RuleConfig   `json:"rule_config"`
	AIConfig    UserAIConfig `json:"ai_config"`
}

// FieldConfig 字段配置
type FieldConfig struct {
	FieldMode      string   `json:"field_mode"`      // 为空时继承租户模式
	FieldOverrides []string `json:"field_overrides"` // 用户在租户选择基础上额外增加的字段 key
}

// RuleConfig 规则配置
type RuleConfig struct {
	CustomRules         []CustomRule         `json:"custom_rules"`
	RuleToggleOverrides []RuleToggleOverride `json:"rule_toggle_overrides"`
}

// UserAIConfig AI 个性化配置
type UserAIConfig struct {
	StrictnessOverride string `json:"strictness_override"`
}

// CustomRule 用户自定义的私有审核规则
type CustomRule struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	Enabled     bool   `json:"enabled"`
	RelatedFlow bool   `json:"related_flow"`
}

// RuleToggleOverride 用户对租户规则的开关覆盖
type RuleToggleOverride struct {
	RuleID  string `json:"rule_id"`
	Enabled bool   `json:"enabled"`
}

// CronDetailItem 用户定时任务相关个人偏好（存储在 cron_details 字段）
type CronDetailItem struct {
	DefaultEmail string `json:"default_email"` // 默认推送邮箱（多个逗号分隔）
}

// ArchiveDetailItem 用户归档复盘中单个流程的个性化设置
type ArchiveDetailItem struct {
	ConfigID    uuid.UUID    `json:"config_id"`    // 对应 process_archive_configs.id
	ProcessType string       `json:"process_type"`
	FieldConfig FieldConfig  `json:"field_config"`
	RuleConfig  RuleConfig   `json:"rule_config"`
	AIConfig    UserAIConfig `json:"ai_config"`
}
