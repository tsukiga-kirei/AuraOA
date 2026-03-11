package model

import (
	"time"

	"github.com/google/uuid"
)

// SystemPromptTemplate 系统提示词模板，全局预置，创建流程时用于初始化 ai_config。
type SystemPromptTemplate struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PromptKey   string     `gorm:"size:100;not null;uniqueIndex" json:"prompt_key"`
	PromptType  string     `gorm:"size:20;not null" json:"prompt_type"`  // system | user
	Phase       string     `gorm:"size:20;not null" json:"phase"`        // reasoning | extraction
	Strictness  *string    `gorm:"size:20" json:"strictness"`            // NULL for system prompts; strict|standard|loose for user prompts
	Content     string     `gorm:"type:text;not null;default:''" json:"content"`
	Description string     `gorm:"size:500;default:''" json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (SystemPromptTemplate) TableName() string { return "system_prompt_templates" }
