package model

import (
	"time"

	"github.com/google/uuid"
)

// TenantLLMMessageLog 租户大模型消息记录，记录每次 AI 调用的 Token 消耗。
type TenantLLMMessageLog struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID      uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	UserID        *uuid.UUID `gorm:"type:uuid" json:"user_id"`
	ModelConfigID *uuid.UUID `gorm:"type:uuid" json:"model_config_id"`
	RequestType   string     `gorm:"size:50;not null;default:audit" json:"request_type"`
	CallType      string     `gorm:"size:20;not null;default:reasoning" json:"call_type"` // reasoning | structured
	InputTokens   int        `gorm:"not null;default:0" json:"input_tokens"`
	OutputTokens  int        `gorm:"not null;default:0" json:"output_tokens"`
	TotalTokens   int        `gorm:"not null;default:0" json:"total_tokens"`
	DurationMs    int        `gorm:"not null;default:0" json:"duration_ms"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (TenantLLMMessageLog) TableName() string { return "tenant_llm_message_logs" }
