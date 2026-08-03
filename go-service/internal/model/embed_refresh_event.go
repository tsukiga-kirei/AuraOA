package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	EmbedRefreshEventPending   = "pending"
	EmbedRefreshEventScheduled = "scheduled"
	EmbedRefreshEventExpired   = "expired"
	EmbedRefreshEventAmbiguous = "ambiguous"
	EmbedRefreshEventFailed    = "failed"
)

// EmbedRefreshEvent 持久化 OA 保存/提交请求及首次新建流程的 requestid 解析状态。
type EmbedRefreshEvent struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID          uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	UserID            uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	EventID           string     `gorm:"size:120;not null" json:"event_id"`
	Action            string     `gorm:"size:30;not null" json:"action"`
	ProcessID         string     `gorm:"size:100;not null;default:''" json:"process_id"`
	WorkflowID        string     `gorm:"size:100;not null;default:''" json:"workflow_id"`
	OABelongUserID    string     `gorm:"size:100;not null;default:''" json:"oa_belong_user_id"`
	OACurrentUserID   string     `gorm:"size:100;not null;default:''" json:"oa_current_user_id"`
	BaselineRequestID int64      `gorm:"not null;default:0" json:"baseline_request_id"`
	Status            string     `gorm:"size:20;not null;default:pending" json:"status"`
	Attempt           int        `gorm:"not null;default:0" json:"attempt"`
	NextAttemptAt     *time.Time `json:"next_attempt_at"`
	LastError         string     `gorm:"type:text;not null;default:''" json:"last_error"`
	ReceivedAt        time.Time  `gorm:"not null" json:"received_at"`
	ResolvedAt        *time.Time `json:"resolved_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (EmbedRefreshEvent) TableName() string { return "embed_refresh_events" }
