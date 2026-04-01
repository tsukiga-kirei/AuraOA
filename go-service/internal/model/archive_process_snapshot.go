package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ArchiveProcessSnapshot 流程级有效归档复盘结论快照。
type ArchiveProcessSnapshot struct {
	ID                       uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID                 uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	ProcessID                string         `gorm:"size:100;not null" json:"process_id"`
	ValidArchiveLogIDs       datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"valid_archive_log_ids"`
	LatestValidArchiveLogID  uuid.UUID      `gorm:"type:uuid;not null" json:"latest_valid_archive_log_id"`
	Title                    string         `gorm:"size:500;not null" json:"title"`
	ProcessType              string         `gorm:"size:200;not null" json:"process_type"`
	Compliance               string         `gorm:"size:30;not null" json:"compliance"`
	ComplianceScore          int            `gorm:"not null;default:0" json:"compliance_score"`
	Confidence               int            `gorm:"not null;default:0" json:"confidence"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

func (ArchiveProcessSnapshot) TableName() string { return "archive_process_snapshots" }
