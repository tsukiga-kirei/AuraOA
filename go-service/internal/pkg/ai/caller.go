package ai

import (
	"context"

	"github.com/google/uuid"

	"auraoa/go-service/internal/model"
)

// AIModelCaller 定义 AI 模型调用接口。
// 不同部署类型（本地 Xinference / 云端阿里百炼）各自实现。
type AIModelCaller interface {
	// TestConnection 测试模型连接是否可用
	TestConnection(ctx context.Context) error

	// Chat 发送对话请求，返回模型响应和 Token 消耗
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

// ChatRequest AI 对话请求
type ChatRequest struct {
	SystemPrompt             string               `json:"system_prompt"`
	UserPrompt               string               `json:"user_prompt"`
	ModelConfig              *model.AIModelConfig `json:"-"`
	Temperature              float64              `json:"temperature"`
	MaxTokens                int                  `json:"max_tokens"`
	EnableThinking           bool                 `json:"enable_thinking"`
	SkipQuotaCheck           bool                 `json:"skip_quota_check"`
	RequestType              string               `json:"request_type"`
	CallType                 string               `json:"call_type"` // reasoning | structured
	ProcessID                string               `json:"process_id"`
	ProcessTitle             string               `json:"process_title"`
	BusinessLogID            *uuid.UUID           `json:"business_log_id"`
	Messages                 []ChatMessage        `json:"messages,omitempty"`
	Tools                    []ToolDefinition     `json:"tools,omitempty"`
	StreamChunkFunc          func(string)         `json:"-"`
	StreamReasoningChunkFunc func(string)         `json:"-"`
}

// ChatMessage 表示多轮对话中的消息
type ChatMessage struct {
	Role             string     `json:"role"` // system, user, assistant, tool
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	Name             string     `json:"name,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// ToolDefinition 定义传给大模型的工具函数格式
type ToolDefinition struct {
	Type     string       `json:"type"` // "function"
	Function FunctionSpec `json:"function"`
}

// FunctionSpec 工具函数详情
type FunctionSpec struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ToolCall 模型发出的工具调用
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction 工具调用的函数名与入参 JSON 字符串
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatResponse AI 对话响应
type ChatResponse struct {
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	TokenUsage       TokenUsage `json:"token_usage"`
	ModelID          string     `json:"model_id"`
	DurationMs       int64      `json:"duration_ms"`
}

// TokenUsage Token 消耗统计
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
