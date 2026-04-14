package model

import (
	"time"

	"github.com/google/uuid"
)

// AIDeployTypeOption AI 部署类型选项。
type AIDeployTypeOption struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code      string    `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Label     string    `gorm:"size:100;not null" json:"label"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

func (AIDeployTypeOption) TableName() string { return "ai_deploy_type_options" }
