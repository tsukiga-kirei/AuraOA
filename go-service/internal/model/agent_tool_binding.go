package model

import (
	"time"

	"github.com/google/uuid"
)

// AgentToolBinding 表示智能体与工具/MCP/Skills的绑定关系
type AgentToolBinding struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenant_id,omitempty"`
	AgentID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"agent_id"`
	ToolType  string     `gorm:"size:32;not null;default:system" json:"tool_type"` // system | mcp | skill
	ToolCode  string     `gorm:"size:128;not null" json:"tool_code"`
	CreatedAt time.Time  `json:"created_at"`
}

func (AgentToolBinding) TableName() string {
	return "agent_tool_bindings"
}
