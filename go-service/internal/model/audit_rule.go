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
	RuleScope   string     `gorm:"size:20;not null;default:default_on" json:"rule_scope"` // 规则作用域：mandatory=强制/default_on=默认开/default_off=默认关
	Enabled     *bool      `gorm:"not null;default:true" json:"enabled"`
	Source      string     `gorm:"size:20;not null;default:manual" json:"source"` // 规则来源：manual=手动创建/file_import=文件导入
	RelatedFlow bool       `gorm:"not null;default:false" json:"related_flow"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (AuditRule) TableName() string { return "audit_rules" }

// GetID 返回规则 ID 字符串，实现 MergeableRule 接口。
func (r AuditRule) GetID() string { return r.ID.String() }

// GetRuleContent 返回规则内容，实现 MergeableRule 接口。
func (r AuditRule) GetRuleContent() string { return r.RuleContent }

// GetRuleScope 返回规则作用域，实现 MergeableRule 接口。
func (r AuditRule) GetRuleScope() string { return r.RuleScope }

// IsEnabled 返回规则是否启用，实现 MergeableRule 接口。
func (r AuditRule) IsEnabled() bool { return r.Enabled == nil || *r.Enabled }
