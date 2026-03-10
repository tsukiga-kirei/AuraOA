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

// OpenAICompatCaller 通用 OpenAI 兼容 API 调用器。
// 适用于所有支持 OpenAI Chat Completions 格式的 provider：
//   - 本地: xinference, ollama, vllm
//   - 云端: aliyun_bailian, deepseek, zhipu, openai, azure_openai
type OpenAICompatCaller struct {
	cfg    *model.AIModelConfig
	client *http.Client
}

// NewOpenAICompatCaller 创建通用 OpenAI 兼容调用器实例。
func NewOpenAICompatCaller(cfg *model.AIModelConfig) (*OpenAICompatCaller, error) {
	return &OpenAICompatCaller{
		cfg: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// TestConnection 测试模型连接是否可用。
func (c *OpenAICompatCaller) TestConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/models", c.cfg.Endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("[%s] 连接失败: %w", c.cfg.Provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("[%s] API Key 无效", c.cfg.Provider)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[%s] 返回状态码: %d", c.cfg.Provider, resp.StatusCode)
	}
	return nil
}

// openAIRequest OpenAI 兼容 API 请求体
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

// openAIMessage OpenAI 消息格式
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse OpenAI 兼容 API 响应体
type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Chat 发送对话请求到 OpenAI 兼容 API。
func (c *OpenAICompatCaller) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
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

	url := fmt.Sprintf("%s/chat/completions", c.cfg.Endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("[%s] 调用失败: %w", c.cfg.Provider, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[%s] 返回错误 (状态码 %d): %s", c.cfg.Provider, resp.StatusCode, string(respBody))
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
