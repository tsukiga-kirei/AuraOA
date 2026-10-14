package dto

import (
	"time"

	"github.com/google/uuid"

	"auraoa/go-service/internal/model"
)

// FeedbackMemoryScope 指定流程类型及独立的审核/归档场景。
type FeedbackMemoryScope struct {
	Module      string `form:"module" json:"module" binding:"required,oneof=audit archive"`
	ProcessType string `form:"process_type" json:"process_type" binding:"required,max=200"`
}

// FeedbackMemoryEntryRequest 新增/编辑手动素材，或启停已导入评论。
type FeedbackMemoryEntryRequest struct {
	FeedbackMemoryScope
	Content   string `json:"content" binding:"required,max=2000"`
	Condition string `json:"condition" binding:"max=1000"`
	Enabled   bool   `json:"enabled"`
}

// FeedbackMemoryImportRequest 批量选择当前流程的评论。
type FeedbackMemoryImportRequest struct {
	FeedbackMemoryScope
	CommentIDs []uuid.UUID `json:"comment_ids" binding:"required,min=1,max=100"`
}

// FeedbackMemoryComment 包含原始流程及反馈人的上下文。
type FeedbackMemoryComment struct {
	ID         uuid.UUID `json:"id"`
	AuditLogID uuid.UUID `json:"audit_log_id"`
	ProcessID  string    `json:"process_id"`
	Title      string    `json:"title"`
	Username   string    `json:"username"`
	Content    string    `json:"content"`
	Feedback   *string   `json:"feedback"`
	UpdatedAt  time.Time `json:"updated_at"`
	Imported   bool      `json:"imported"`
	Deprecated bool      `json:"deprecated"`
}

// FeedbackMemoryWorkspace 返回当前素材、发布记忆及只读生成规则。
type FeedbackMemoryWorkspace struct {
	Memory             *model.FeedbackMemory        `json:"memory"`
	Entries            []model.FeedbackMemoryEntry  `json:"entries"`
	Published          *model.FeedbackMemoryVersion `json:"published"`
	Baseline           *model.FeedbackMemoryVersion `json:"baseline"`
	DeletedEntries     []model.FeedbackMemoryEntry  `json:"-"`
	SystemPrompt       string                       `json:"system_prompt"`
	UserPromptTemplate string                       `json:"user_prompt_template"`
}

// FeedbackMemoryCommentQuery 在服务端分页前筛选弃用状态，默认隐藏已弃用建议。
type FeedbackMemoryCommentQuery struct {
	ExperienceQuery
	Deprecation string `form:"deprecation" binding:"omitempty,oneof=active deprecated all"`
}

// FeedbackMemoryCommentStateRequest 设置当前记忆库的建议弃用状态。
type FeedbackMemoryCommentStateRequest struct {
	FeedbackMemoryScope
	Deprecated bool `json:"deprecated"`
}

// FeedbackMemoryDraftRequest 整体保存条目，使用编辑修订号防止相互覆盖。
type FeedbackMemoryDraftRequest struct {
	FeedbackMemoryScope
	Items        []model.FeedbackMemoryItem `json:"items" binding:"max=80"`
	EditRevision int                        `json:"edit_revision" binding:"min=0"`
}

// FeedbackMemoryPublishRequest 发布用户已审阅的编辑修订，避免发布他人刚修改的正文。
type FeedbackMemoryPublishRequest struct {
	FeedbackMemoryScope
	EditRevision int `json:"edit_revision" binding:"min=0"`
}
