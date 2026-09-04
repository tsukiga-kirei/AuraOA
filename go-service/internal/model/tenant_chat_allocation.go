package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// TenantChatAllocation 系统管理员分配给租户的对话与智能体配额
type TenantChatAllocation struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID          uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"tenant_id"`
	AgentCodes        datatypes.JSON `gorm:"type:jsonb;not null;default:'[\"oa_query\",\"oa_assist\"]'" json:"agent_codes"`
	ToolCodes         datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"tool_codes"`
	SkillCodes        datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"skill_codes"`
	AllowCustomSkills bool           `gorm:"not null;default:false" json:"allow_custom_skills"`
	AllowTenantMCP    bool           `gorm:"not null;default:false" json:"allow_tenant_mcp"`
	MaxMCPServers     int            `gorm:"not null;default:0" json:"max_mcp_servers"`
	MCPTemplateIDs    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"mcp_template_ids"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

func (TenantChatAllocation) TableName() string {
	return "tenant_chat_allocations"
}
