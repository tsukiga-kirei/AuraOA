package model

import (
	"time"

	"github.com/google/uuid"
)

// AgentSkill 智能体指令包 (SKILL.md)
type AgentSkill struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    *uuid.UUID `gorm:"type:uuid;index" json:"tenant_id,omitempty"` // NULL 表示系统内置
	SkillCode   string     `gorm:"size:64;not null" json:"skill_code"`
	Name        string     `gorm:"size:128;not null" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	Enabled     bool       `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (AgentSkill) TableName() string {
	return "agent_skills"
}
