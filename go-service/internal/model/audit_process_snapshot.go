package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AuditProcessSnapshot 流程级有效审核结论快照（仅成功解析的 audit_logs id 链）。
type AuditProcessSnapshot struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID          uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	ProcessID         string         `gorm:"size:100;not null" json:"process_id"`
	ValidLogIDs       datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"valid_log_ids"`
	LatestValidLogID  uuid.UUID      `gorm:"type:uuid;not null" json:"latest_valid_log_id"`
	Title             string         `gorm:"size:500;not null" json:"title"`
	ProcessType       string         `gorm:"size:200;not null" json:"process_type"`
	Recommendation    string         `gorm:"size:20;not null" json:"recommendation"`
	Score             int            `gorm:"not null;default:0" json:"score"`
	Confidence        int            `gorm:"not null;default:0" json:"confidence"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

func (AuditProcessSnapshot) TableName() string { return "audit_process_snapshots" }
