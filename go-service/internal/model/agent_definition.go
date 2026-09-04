package model

import (
	"time"

	"github.com/google/uuid"
)

// AgentDefinition 表示智能体定义（平台种子或租户自定义）
type AgentDefinition struct {
	ID           uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID     *uuid.UUID         `gorm:"type:uuid;index" json:"tenant_id,omitempty"`
	AgentCode    string             `gorm:"size:64;not null;index" json:"agent_code"`
	Name         string             `gorm:"size:128;not null" json:"name"`
	Description  string             `gorm:"type:text" json:"description"`
	SystemPrompt string             `gorm:"type:text;not null;default:''" json:"system_prompt"`
	Enabled      bool               `gorm:"not null;default:true" json:"enabled"`
	IsSystem     bool               `gorm:"not null;default:false" json:"is_system"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`

	// 关联工具绑定
	ToolBindings []AgentToolBinding `gorm:"foreignKey:AgentID" json:"tool_bindings,omitempty"`
}

func (AgentDefinition) TableName() string {
	return "agent_definitions"
}
