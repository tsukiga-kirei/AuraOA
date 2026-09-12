package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditComment 保存审核结果下的用户评论；正文按纯文本展示。
type AuditComment struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID   uuid.UUID `json:"-"`
	AuditLogID uuid.UUID `json:"audit_log_id"`
	OAUserID   string    `json:"oa_user_id"`
	Username   string    `json:"username"`
	Feedback   *string   `json:"feedback"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	CanManage  bool      `gorm:"-" json:"can_manage"`
}
