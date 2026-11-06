package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AgentDefinition 表示智能体定义（平台种子或租户自定义）
type AgentDefinition struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID         *uuid.UUID     `gorm:"type:uuid;index" json:"tenant_id,omitempty"`
	AgentCode        string         `gorm:"size:64;not null;index" json:"agent_code"`
	Name             string         `gorm:"size:128;not null" json:"name"`
	Description      string         `gorm:"type:text" json:"description"`
	SystemPrompt     string         `gorm:"type:text;not null;default:''" json:"system_prompt"`
	Enabled          bool           `gorm:"not null;default:true" json:"enabled"`
	IsSystem         bool           `gorm:"not null;default:false" json:"is_system"`
	ToolHistoryTurns int            `gorm:"not null" json:"tool_history_turns"`
	MaxLoopSteps     int            `gorm:"not null;default:8" json:"max_loop_steps"`
	QuickQuestions   datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"quick_questions"`
	AccessControl    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"access_control"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`

	// 关联工具绑定
	ToolBindings []AgentToolBinding `gorm:"foreignKey:AgentID" json:"tool_bindings,omitempty"`
}

func (AgentDefinition) TableName() string {
	return "agent_definitions"
}

// EffectiveToolHistoryTurns 保留显式零轮数，仅对非法负值使用历史默认配置。
func (a AgentDefinition) EffectiveToolHistoryTurns() int {
	if a.ToolHistoryTurns < 0 {
		return 3
	}
	return a.ToolHistoryTurns
}

// EffectiveMaxLoopSteps 返回单轮对话允许的模型步数。未配置时用 8，超出 1-20 时收拢到边界。
func (a AgentDefinition) EffectiveMaxLoopSteps() int {
	steps := a.MaxLoopSteps
	if steps <= 0 {
		return 8
	}
	if steps > 20 {
		return 20
	}
	return steps
}
