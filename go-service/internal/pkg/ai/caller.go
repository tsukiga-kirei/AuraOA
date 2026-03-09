package ai

import (
	"context"

	"oa-smart-audit/go-service/internal/model"
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
	SystemPrompt string             `json:"system_prompt"`
	UserPrompt   string             `json:"user_prompt"`
	ModelConfig  *model.AIModelConfig `json:"-"`
	Temperature  float64            `json:"temperature"`
	MaxTokens    int                `json:"max_tokens"`
}

// ChatResponse AI 对话响应
type ChatResponse struct {
	Content    string     `json:"content"`
	TokenUsage TokenUsage `json:"token_usage"`
	ModelID    string     `json:"model_id"`
	DurationMs int64     `json:"duration_ms"`
}

// TokenUsage Token 消耗统计
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
