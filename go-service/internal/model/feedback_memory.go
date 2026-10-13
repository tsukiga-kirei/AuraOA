package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// FeedbackMemory 按租户、业务模块和流程类型维护独立的反馈记忆。
type FeedbackMemory struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID           uuid.UUID  `gorm:"type:uuid" json:"-"`
	Module             string     `json:"module"`
	ProcessType        string     `json:"process_type"`
	Revision           int        `json:"revision"`
	LastVersion        int        `json:"last_version"`
	PublishedVersionID *uuid.UUID `gorm:"type:uuid" json:"published_version_id"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// FeedbackMemoryEntry 保留逐条素材；导入评论的原文快照不受原评论后续修改影响。
type FeedbackMemoryEntry struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID       uuid.UUID      `gorm:"type:uuid" json:"-"`
	MemoryID       uuid.UUID      `gorm:"type:uuid" json:"memory_id"`
	SourceType     string         `json:"source_type"`
	SourceID       *uuid.UUID     `gorm:"type:uuid" json:"source_id"`
	Content        string         `json:"content"`
	Condition      string         `json:"condition"`
	Enabled        bool           `json:"enabled"`
	SourceSnapshot datatypes.JSON `gorm:"type:jsonb" json:"source_snapshot"`
	CreatedBy      string         `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// FeedbackMemoryItem 将适用条件、记忆内容及其来源逐条关联。
type FeedbackMemoryItem struct {
	Condition  string   `json:"condition"`
	Content    string   `json:"content"`
	SourceRefs []string `json:"source_refs"`
}

// FeedbackMemoryVersion 保存生成时点、完整输入、输出与发布时点，正文生成后不可变。
type FeedbackMemoryVersion struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID      uuid.UUID      `gorm:"type:uuid" json:"-"`
	MemoryID      uuid.UUID      `gorm:"type:uuid" json:"memory_id"`
	VersionNo     int            `json:"version_no"`
	BaseVersionID *uuid.UUID     `gorm:"type:uuid" json:"base_version_id"`
	InputRevision int            `json:"input_revision"`
	Status        string         `json:"status"`
	Sources       datatypes.JSON `gorm:"type:jsonb" json:"sources"`
	Items         datatypes.JSON `gorm:"type:jsonb" json:"items"`
	Warnings      datatypes.JSON `gorm:"type:jsonb" json:"warnings"`
	SystemPrompt  string         `json:"system_prompt"`
	UserPrompt    string         `json:"user_prompt"`
	PromptText    string         `json:"prompt_text"`
	CreatedBy     string         `json:"created_by"`
	CreatedAt     time.Time      `json:"created_at"`
	PublishedAt   *time.Time     `json:"published_at"`
}
