package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AIModelConfig 系统级 AI 模型配置。
type AIModelConfig struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Provider         string         `gorm:"size:100;not null" json:"provider"`
	ProviderLabel    string         `gorm:"size:100;default:''" json:"provider_label"`
	ModelName        string         `gorm:"size:100;not null" json:"model_name"`
	DisplayName      string         `gorm:"size:200;not null" json:"display_name"`
	DeployType       string         `gorm:"size:20;not null;default:local" json:"deploy_type"`
	Endpoint         string         `gorm:"size:500;not null;default:''" json:"endpoint"`
	APIKey           string         `gorm:"column:api_key;size:500;default:''" json:"-"` // 不输出到JSON
	APIKeyConfigured bool           `gorm:"not null;default:false" json:"api_key_configured"`
	MaxTokens        int            `gorm:"not null;default:8192" json:"max_tokens"`
	ContextWindow    int            `gorm:"not null;default:131072" json:"context_window"`
	CostPer1kTokens  float64        `gorm:"column:cost_per_1k_tokens;type:decimal(10,6);default:0" json:"cost_per_1k_tokens"`
	Status           string         `gorm:"size:20;not null;default:offline" json:"status"`
	Enabled          bool           `gorm:"not null;default:true" json:"enabled"`
	Description      string         `gorm:"type:text;default:''" json:"description"`
	Capabilities     datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"capabilities"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

func (AIModelConfig) TableName() string { return "ai_model_configs" }
