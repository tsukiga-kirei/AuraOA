package model

import (
	"time"

	"github.com/google/uuid"
)

// ArchiveRule 归档复盘规则（独立于审核规则）。
type ArchiveRule struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"` // 主键UUID
	TenantID    uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`                       // 所属租户ID
	ConfigID    *uuid.UUID `gorm:"type:uuid" json:"config_id"`                                // 所属归档配置ID（NULL 表示通用规则）
	ProcessType string     `gorm:"size:200;not null" json:"process_type"`                     // 适用流程类型
	RuleContent string     `gorm:"type:text;not null" json:"rule_content"`                    // 规则内容（自然语言描述）
	RuleScope   string     `gorm:"size:20;not null;default:default_on" json:"rule_scope"`     // 规则作用域：mandatory/default_on/default_off
	Enabled     bool       `gorm:"not null;default:true" json:"enabled"`                      // 是否启用
	Source      string     `gorm:"size:20;not null;default:manual" json:"source"`             // 规则来源：manual/file_import
	RelatedFlow bool       `gorm:"not null;default:false" json:"related_flow"`                // 是否关联审批流
	CreatedAt   time.Time  `json:"created_at"`                                                // 创建时间
	UpdatedAt   time.Time  `json:"updated_at"`                                                // 最后更新时间
}

func (ArchiveRule) TableName() string { return "archive_rules" }
