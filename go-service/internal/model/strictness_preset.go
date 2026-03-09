package model

import (
	"time"

	"github.com/google/uuid"
)

// StrictnessPreset 审核尺度预设，每个租户维护 strict/standard/loose 三条记录。
type StrictnessPreset struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID              uuid.UUID `gorm:"type:uuid;not null" json:"tenant_id"`
	Strictness            string    `gorm:"size:20;not null" json:"strictness"` // strict | standard | loose
	ReasoningInstruction  string    `gorm:"type:text;not null;default:''" json:"reasoning_instruction"`
	ExtractionInstruction string    `gorm:"type:text;not null;default:''" json:"extraction_instruction"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (StrictnessPreset) TableName() string { return "strictness_presets" }
