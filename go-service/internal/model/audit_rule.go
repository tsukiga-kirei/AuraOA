package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditRule 审核规则，按租户隔离的审核检查项。
type AuditRule struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	ConfigID    *uuid.UUID `gorm:"type:uuid" json:"config_id"`
	ProcessType string     `gorm:"size:200;not null" json:"process_type"`
	RuleContent string     `gorm:"type:text;not null" json:"rule_content"`
	RuleScope   string     `gorm:"size:20;not null;default:default_on" json:"rule_scope"` // mandatory | default_on | default_off
	Enabled     bool       `gorm:"not null;default:true" json:"enabled"`
	Source      string     `gorm:"size:20;not null;default:manual" json:"source"` // manual | file_import
	RelatedFlow bool       `gorm:"not null;default:false" json:"related_flow"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (AuditRule) TableName() string { return "audit_rules" }
