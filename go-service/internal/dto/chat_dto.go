package dto

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// EffectiveAgentDTO 用户可见的有效智能体
type EffectiveAgentDTO struct {
	ID             uuid.UUID           `json:"id"`
	AgentCode      string              `json:"agent_code"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	IsSystem       bool                `json:"is_system"`
	QuickQuestions []QuickQuestionItem `json:"quick_questions"`
	ToolCodes      []string            `json:"tool_codes"`
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	AgentCode string  `json:"agent_code" binding:"required"`
	Title     string  `json:"title"`
	ProcessID *string `json:"process_id"`
	Source    string  `json:"source"` // standalone | embed
}

// UpdateSessionRequest 更新会话请求（如重命名、置顶）
type UpdateSessionRequest struct {
	Title  *string `json:"title"`
	Pinned *bool   `json:"pinned"`
}

// ChatSessionItemDTO 会话列表项
type ChatSessionItemDTO struct {
	ID        uuid.UUID `json:"id"`
	AgentID   uuid.UUID `json:"agent_id"`
	AgentCode string    `json:"agent_code"`
	AgentName string    `json:"agent_name,omitempty"`
	Title     string    `json:"title"`
	Source    string    `json:"source"`
	ProcessID *string   `json:"process_id"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatSessionListResponse 会话列表分页响应
type ChatSessionListResponse struct {
	Items    []ChatSessionItemDTO `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

// ChatMessageDTO 消息详情
type ChatMessageDTO struct {
	ID               uuid.UUID      `json:"id"`
	SessionID        uuid.UUID      `json:"session_id"`
	Role             string         `json:"role"`
	Content          string         `json:"content"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	Status           string         `json:"status"`
	ToolCalls        datatypes.JSON `json:"tool_calls"`
	TokenUsage       datatypes.JSON `json:"token_usage,omitempty"`
	Feedback         *string        `json:"feedback,omitempty"`
	FeedbackAt       *time.Time     `json:"feedback_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

// UpdateFeedbackRequest 更新消息点赞/点踩评价请求
type UpdateFeedbackRequest struct {
	Feedback *string `json:"feedback"` // like | dislike | nil
}

// ChatSessionDetailResponse 会话详情响应（含历史消息）
type ChatSessionDetailResponse struct {
	Session  ChatSessionItemDTO `json:"session"`
	Messages []ChatMessageDTO   `json:"messages"`
}

// SendMessageStreamRequest 发送消息流式请求
type SendMessageStreamRequest struct {
	Content string `json:"content" binding:"required"`
}

// SSEEvent SSE 流事件标准结构
type SSEEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}
