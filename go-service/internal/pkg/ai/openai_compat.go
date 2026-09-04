package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"auraoa/go-service/internal/model"
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
			Timeout: 30 * time.Minute,
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
	Model              string                 `json:"model"`
	Messages           []openAIMessage        `json:"messages"`
	Temperature        float64                `json:"temperature"`
	MaxTokens          int                    `json:"max_tokens,omitempty"`
	Stream             bool                   `json:"stream,omitempty"`
	StreamOptions      *openAIStreamOptions   `json:"stream_options,omitempty"`
	ChatTemplateKwargs map[string]interface{} `json:"chat_template_kwargs,omitempty"`
	Tools              []ToolDefinition       `json:"tools,omitempty"`
}

// openAIStreamOptions OpenAI 兼容流式选项。
type openAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// openAIMessage OpenAI 消息格式
type openAIMessage struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	Reasoning        string     `json:"reasoning,omitempty"`
	Name             string     `json:"name,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

type openAIChoice struct {
	Message openAIChoiceMessage `json:"message"`
}

type openAIChoiceMessage struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	Reasoning        string     `json:"reasoning,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// openAIResponse OpenAI 兼容 API 响应体
type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Chat 发送对话请求到 OpenAI 兼容 API。
func (c *OpenAICompatCaller) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()

	var messages []openAIMessage
	if len(req.Messages) > 0 {
		for _, m := range req.Messages {
			messages = append(messages, openAIMessage{
				Role:             m.Role,
				Content:          m.Content,
				ReasoningContent: m.ReasoningContent,
				Name:             m.Name,
				ToolCallID:       m.ToolCallID,
				ToolCalls:        m.ToolCalls,
			})
		}
	} else {
		if req.SystemPrompt != "" {
			messages = append(messages, openAIMessage{Role: "system", Content: req.SystemPrompt})
		}
		messages = append(messages, openAIMessage{Role: "user", Content: req.UserPrompt})
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.3
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.cfg.MaxTokens
	}

	chatTemplateKwargs := map[string]interface{}{}
	if req.EnableThinking {
		chatTemplateKwargs["enable_thinking"] = true
		chatTemplateKwargs["thinking"] = true
	} else {
		chatTemplateKwargs["enable_thinking"] = false
		chatTemplateKwargs["thinking"] = false
	}

	body := openAIRequest{
		Model:              c.cfg.ModelName,
		Messages:           messages,
		Temperature:        temperature,
		MaxTokens:          maxTokens,
		Stream:             req.StreamChunkFunc != nil || req.StreamReasoningChunkFunc != nil,
		ChatTemplateKwargs: chatTemplateKwargs,
		Tools:              req.Tools,
	}
	if body.Stream {
		// 在支持 OpenAI stream_options 的实现中，要求在最终 chunk 返回 usage。
		body.StreamOptions = &openAIStreamOptions{IncludeUsage: true}
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

	if body.Stream {
		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("[%s] 返回错误 (状态码 %d): %s", c.cfg.Provider, resp.StatusCode, string(respBody))
		}

		reader := bufio.NewReader(resp.Body)
		var fullContent strings.Builder
		var fullReasoning strings.Builder
		tokenUsage := TokenUsage{}
		usageReceived := false
		toolCallsMap := make(map[int]*ToolCall)

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("读取流失败: %w", err)
			}
			line = bytes.TrimSpace(line)
			if !bytes.HasPrefix(line, []byte("data: ")) {
				continue
			}
			data := bytes.TrimPrefix(line, []byte("data: "))
			if string(data) == "[DONE]" {
				break
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content          string `json:"content"`
						Reasoning        string `json:"reasoning"`         // vLLM 新版
						ReasoningContent string `json:"reasoning_content"` // DeepSeek / 百炼 / vLLM 旧版
						ToolCalls        []struct {
							Index    int    `json:"index"`
							ID       string `json:"id"`
							Type     string `json:"type"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"delta"`
				} `json:"choices"`
				Usage *struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal(data, &chunk); err == nil {
				if chunk.Usage != nil {
					tokenUsage = TokenUsage{
						InputTokens:  chunk.Usage.PromptTokens,
						OutputTokens: chunk.Usage.CompletionTokens,
						TotalTokens:  chunk.Usage.TotalTokens,
					}
					usageReceived = true
				}
				if len(chunk.Choices) > 0 {
					delta := chunk.Choices[0].Delta
					deltaReasoning := delta.Reasoning
					if deltaReasoning == "" {
						deltaReasoning = delta.ReasoningContent
					}
					if deltaReasoning != "" {
						fullReasoning.WriteString(deltaReasoning)
						if req.StreamReasoningChunkFunc != nil {
							req.StreamReasoningChunkFunc(deltaReasoning)
						}
					}
					if delta.Content != "" {
						fullContent.WriteString(delta.Content)
						if req.StreamChunkFunc != nil {
							req.StreamChunkFunc(delta.Content)
						}
					}
					if len(delta.ToolCalls) > 0 {
						for _, tc := range delta.ToolCalls {
							existing, ok := toolCallsMap[tc.Index]
							if !ok {
								existing = &ToolCall{
									ID:   tc.ID,
									Type: tc.Type,
									Function: ToolCallFunction{
										Name: tc.Function.Name,
									},
								}
								if existing.Type == "" {
									existing.Type = "function"
								}
								toolCallsMap[tc.Index] = existing
							}
							if tc.ID != "" {
								existing.ID = tc.ID
							}
							if tc.Function.Name != "" {
								existing.Function.Name = tc.Function.Name
							}
							if tc.Function.Arguments != "" {
								existing.Function.Arguments += tc.Function.Arguments
							}
						}
					}
				}
			}
		}
		outContent := fullContent.String()
		outReasoning := fullReasoning.String()
		if !usageReceived {
			// 兼容不返回 usage 的服务：不做估算，保持空值（0）。
			tokenUsage = TokenUsage{}
		}

		var toolCalls []ToolCall
		for i := 0; i < len(toolCallsMap); i++ {
			if tc, ok := toolCallsMap[i]; ok {
				toolCalls = append(toolCalls, *tc)
			}
		}
		if len(toolCalls) == 0 {
			toolCalls = detectFallbackToolCalls(outContent)
		}

		return &ChatResponse{
			Content:          outContent,
			ReasoningContent: outReasoning,
			ToolCalls:        toolCalls,
			TokenUsage:       tokenUsage,
			ModelID:          c.cfg.ModelName,
			DurationMs:       time.Since(startTime).Milliseconds(),
		}, nil
	}

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
	reasoning := ""
	var toolCalls []ToolCall
	if len(oaiResp.Choices) > 0 {
		content = oaiResp.Choices[0].Message.Content
		reasoning = oaiResp.Choices[0].Message.Reasoning
		if reasoning == "" {
			reasoning = oaiResp.Choices[0].Message.ReasoningContent
		}
		toolCalls = oaiResp.Choices[0].Message.ToolCalls
	}
	if len(toolCalls) == 0 {
		toolCalls = detectFallbackToolCalls(content)
	}

	return &ChatResponse{
		Content:          content,
		ReasoningContent: reasoning,
		ToolCalls:        toolCalls,
		TokenUsage: TokenUsage{
			InputTokens:  oaiResp.Usage.PromptTokens,
			OutputTokens: oaiResp.Usage.CompletionTokens,
			TotalTokens:  oaiResp.Usage.TotalTokens,
		},
		ModelID:    c.cfg.ModelName,
		DurationMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// detectFallbackToolCalls 针对无法原生调用 OpenAI tool_calls 的模型，检测 Markdown 中的 JSON 结构化工具调用
func detectFallbackToolCalls(content string) []ToolCall {
	if !strings.Contains(content, "tool") && !strings.Contains(content, "function") && !strings.Contains(content, "call") {
		return nil
	}
	// 匹配 ```json ... ``` 块
	startIdx := strings.Index(content, "```json")
	if startIdx == -1 {
		startIdx = strings.Index(content, "```")
	}
	if startIdx == -1 {
		return nil
	}
	block := content[startIdx:]
	if strings.HasPrefix(block, "```json") {
		block = strings.TrimPrefix(block, "```json")
	} else {
		block = strings.TrimPrefix(block, "```")
	}
	endIdx := strings.Index(block, "```")
	if endIdx != -1 {
		block = block[:endIdx]
	}
	block = strings.TrimSpace(block)

	var payload struct {
		Name       string                 `json:"name"`
		Tool       string                 `json:"tool"`
		Action     string                 `json:"action"`
		Arguments  map[string]interface{} `json:"arguments"`
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := json.Unmarshal([]byte(block), &payload); err == nil {
		toolName := payload.Name
		if toolName == "" {
			toolName = payload.Tool
		}
		if toolName == "" {
			toolName = payload.Action
		}
		if toolName != "" {
			args := payload.Arguments
			if args == nil {
				args = payload.Parameters
			}
			argsJSON, _ := json.Marshal(args)
			return []ToolCall{
				{
					ID:   "call_" + fmt.Sprintf("%d", time.Now().UnixNano()),
					Type: "function",
					Function: ToolCallFunction{
						Name:      toolName,
						Arguments: string(argsJSON),
					},
				},
			}
		}
	}
	return nil
}
