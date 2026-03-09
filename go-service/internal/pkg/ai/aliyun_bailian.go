package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"oa-smart-audit/go-service/internal/model"
)

// AliyunBailianCaller 阿里云百炼云端模型的调用器。
// 通过 DashScope 兼容 API（OpenAI 格式）调用云端模型。
type AliyunBailianCaller struct {
	cfg    *model.AIModelConfig
	client *http.Client
}

// 阿里云百炼默认 Endpoint
const defaultBailianEndpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1"

// NewAliyunBailianCaller 创建阿里云百炼调用器实例。
func NewAliyunBailianCaller(cfg *model.AIModelConfig) (*AliyunBailianCaller, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("阿里云百炼需要配置 API Key")
	}
	return &AliyunBailianCaller{
		cfg: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// getEndpoint 获取 API Endpoint，优先使用配置值，否则使用默认值。
func (c *AliyunBailianCaller) getEndpoint() string {
	if c.cfg.Endpoint != "" {
		return c.cfg.Endpoint
	}
	return defaultBailianEndpoint
}

// TestConnection 测试阿里云百炼模型连接是否可用。
func (c *AliyunBailianCaller) TestConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/models", c.getEndpoint())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("阿里云百炼连接失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("阿里云百炼 API Key 无效")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("阿里云百炼返回状态码: %d", resp.StatusCode)
	}
	return nil
}

// Chat 发送对话请求到阿里云百炼云端模型。
func (c *AliyunBailianCaller) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()

	messages := []openAIMessage{
		{Role: "system", Content: req.SystemPrompt},
		{Role: "user", Content: req.UserPrompt},
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.3
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.cfg.MaxTokens
	}

	body := openAIRequest{
		Model:       c.cfg.ModelName,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", c.getEndpoint())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("阿里云百炼调用失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("阿里云百炼返回错误 (状态码 %d): %s", resp.StatusCode, string(respBody))
	}

	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	content := ""
	if len(oaiResp.Choices) > 0 {
		content = oaiResp.Choices[0].Message.Content
	}

	return &ChatResponse{
		Content: content,
		TokenUsage: TokenUsage{
			InputTokens:  oaiResp.Usage.PromptTokens,
			OutputTokens: oaiResp.Usage.CompletionTokens,
			TotalTokens:  oaiResp.Usage.TotalTokens,
		},
		ModelID:    c.cfg.ModelName,
		DurationMs: time.Since(startTime).Milliseconds(),
	}, nil
}
