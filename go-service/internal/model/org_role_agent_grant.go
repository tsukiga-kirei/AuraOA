package model

import (
	"time"

	"github.com/google/uuid"
)

// OrgRoleAgentGrant 组织角色授予的智能体
type OrgRoleAgentGrant struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	RoleID    uuid.UUID `gorm:"type:uuid;not null;index" json:"role_id"`
	AgentCode string    `gorm:"size:64;not null" json:"agent_code"`
	CreatedAt time.Time `json:"created_at"`
}

func (OrgRoleAgentGrant) TableName() string {
	return "org_role_agent_grants"
}

// OrgRoleToolGrant 组织角色授予的工具（含系统工具、mcp、skill脚本）
type OrgRoleToolGrant struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	RoleID    uuid.UUID `gorm:"type:uuid;not null;index" json:"role_id"`
	ToolCode  string    `gorm:"size:128;not null" json:"tool_code"`
	CreatedAt time.Time `json:"created_at"`
}

func (OrgRoleToolGrant) TableName() string {
	return "org_role_tool_grants"
}
