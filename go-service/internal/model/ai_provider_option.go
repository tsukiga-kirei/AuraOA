package model

import (
	"time"

	"github.com/google/uuid"
)

// AIProviderOption AI 服务商选项。
type AIProviderOption struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code       string    `gorm:"uniqueIndex;size:100;not null" json:"code"`
	Label      string    `gorm:"size:100;not null" json:"label"`
	DeployType string    `gorm:"size:50;not null" json:"deploy_type"`
	SortOrder  int       `gorm:"not null;default:0" json:"sort_order"`
	Enabled    bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AIProviderOption) TableName() string { return "ai_provider_options" }
