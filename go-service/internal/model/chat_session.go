package model

import (
	"time"

	"github.com/google/uuid"
)

// ChatSession 表示一个 AI 对话会话
type ChatSession struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	AgentID   uuid.UUID  `gorm:"type:uuid;not null" json:"agent_id"`
	AgentCode string     `gorm:"size:64;not null" json:"agent_code"`
	Title     string     `gorm:"size:255;not null" json:"title"`
	Source    string     `gorm:"size:32;not null;default:standalone" json:"source"` // standalone | embed
	ProcessID *string    `gorm:"size:128" json:"process_id"`
	Pinned    bool       `gorm:"not null;default:false" json:"pinned"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// 关联
	Agent *AgentDefinition `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
}

func (ChatSession) TableName() string {
	return "chat_sessions"
}
