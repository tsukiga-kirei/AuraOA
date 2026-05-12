package model

import (
	"time"

	"github.com/google/uuid"
)

// OperationAuditLog 用户 HTTP 操作审计（受 system.enable_audit_trail 控制）。
type OperationAuditLog struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	TenantID   *uuid.UUID `gorm:"type:uuid;index" json:"tenant_id,omitempty"`
	Method     string     `gorm:"size:16;not null" json:"method"`
	Path       string     `gorm:"type:text;not null" json:"path"`
	StatusCode int        `gorm:"not null" json:"status_code"`
	LatencyMs  int        `gorm:"not null" json:"latency_ms"`
	ClientIP   string     `gorm:"size:64" json:"client_ip"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (OperationAuditLog) TableName() string {
	return "operation_audit_logs"
}
