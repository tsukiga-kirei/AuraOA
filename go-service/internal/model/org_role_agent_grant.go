package model

import (
	"time"

	"github.com/google/uuid"
)

// OrgRoleAgentGrant 保留历史角色智能体授权表映射，已不参与权限计算或角色读写。
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

// OrgRoleToolGrant 保留历史角色工具授权表映射，已不参与权限计算或角色读写。
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
