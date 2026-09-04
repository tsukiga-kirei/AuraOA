package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ChatMessage 表示会话中的单条消息
type ChatMessage struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"session_id"`
	TenantID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Role             string         `gorm:"size:32;not null" json:"role"` // user | assistant | system | tool
	Content          string         `gorm:"type:text;not null" json:"content"`
	ReasoningContent string         `gorm:"type:text" json:"reasoning_content,omitempty"`
	Status           string         `gorm:"size:32;not null;default:success" json:"status"` // success | error | interrupted
	ToolCalls        datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"tool_calls"`
	TokenUsage       datatypes.JSON `gorm:"type:jsonb" json:"token_usage,omitempty"`
	LLMLogID         *uuid.UUID     `gorm:"type:uuid" json:"llm_log_id,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

func (ChatMessage) TableName() string {
	return "chat_messages"
}
