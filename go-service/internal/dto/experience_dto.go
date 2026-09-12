package dto

import (
	"time"

	"github.com/google/uuid"

	"auraoa/go-service/internal/model"
)

// ExperienceQuery 定义体验反馈列表筛选条件。
type ExperienceQuery struct {
	Keyword  string `form:"keyword"`
	Feedback string `form:"feedback" binding:"omitempty,oneof=like dislike comments"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ExperiencePage 是统一的服务端分页响应。
type ExperiencePage[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// AuditExperienceItem 展示有互动的审核批次。
type AuditExperienceItem struct {
	ID           uuid.UUID `json:"id"`
	ProcessID    string    `json:"process_id"`
	Title        string    `json:"title"`
	ProcessType  string    `json:"process_type"`
	LikeCount    int64     `json:"like_count"`
	DislikeCount int64     `json:"dislike_count"`
	CommentCount int64     `json:"comment_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AgentExperienceItem 展示被评价的助手消息及所属会话。
type AgentExperienceItem struct {
	ID              uuid.UUID `json:"id"`
	SessionID       uuid.UUID `json:"session_id"`
	Title           string    `json:"title"`
	AgentName       string    `json:"agent_name"`
	Username        string    `json:"username"`
	Content         string    `json:"content"`
	Feedback        string    `json:"feedback"`
	FeedbackComment string    `json:"feedback_comment"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// AuditInteractionResponse 返回分页反馈，评价数量按消息条数统计。
type AuditInteractionResponse struct {
	ExperiencePage[model.AuditComment]
	LikeCount    int64 `json:"like_count"`
	DislikeCount int64 `json:"dislike_count"`
	CanInteract  bool  `json:"can_interact"`
}

// AuditCommentRequest 新增评论，服务层会去除首尾空白并检查字数。
type AuditCommentRequest struct {
	Content  string  `json:"content" binding:"required"`
	Feedback *string `json:"feedback" binding:"omitempty,oneof=like dislike"`
}
