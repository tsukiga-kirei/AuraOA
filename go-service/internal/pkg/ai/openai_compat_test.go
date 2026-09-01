package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"auraoa/go-service/internal/model"
)

func TestOpenAICompatCaller_NonStreaming_Reasoning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req failed: %v", err)
		}
		if req.ChatTemplateKwargs == nil || req.ChatTemplateKwargs["enable_thinking"] != true {
			t.Fatalf("expected enable_thinking=true, got: %v", req.ChatTemplateKwargs)
		}

		resp := openAIResponse{
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}
		resp.Choices = append(resp.Choices, struct {
			Message struct {
				Role             string `json:"role"`
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content,omitempty"`
				Reasoning        string `json:"reasoning,omitempty"`
			} `json:"message"`
		}{
			Message: struct {
				Role             string `json:"role"`
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content,omitempty"`
				Reasoning        string `json:"reasoning,omitempty"`
			}{
				Role:      "assistant",
				Content:   "这是回答正文",
				Reasoning: "这是思考链路",
			},
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	caller, err := NewOpenAICompatCaller(&model.AIModelConfig{
		Provider:  "vllm",
		Endpoint:  server.URL,
		ModelName: "qwen-qwq",
		MaxTokens: 2048,
	})
	if err != nil {
		t.Fatalf("create caller failed: %v", err)
	}

	chatResp, err := caller.Chat(context.Background(), &ChatRequest{
		SystemPrompt:   "system",
		UserPrompt:     "user",
		EnableThinking: true,
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if chatResp.Content != "这是回答正文" {
		t.Errorf("expected Content '这是回答正文', got: %s", chatResp.Content)
	}
	if chatResp.ReasoningContent != "这是思考链路" {
		t.Errorf("expected ReasoningContent '这是思考链路', got: %s", chatResp.ReasoningContent)
	}
	if chatResp.TokenUsage.TotalTokens != 30 {
		t.Errorf("expected TotalTokens 30, got: %d", chatResp.TokenUsage.TotalTokens)
	}
}

func TestOpenAICompatCaller_Streaming_ReasoningContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		// chunk 1: reasoning
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"深度思考中...\"}}]}\n\n"))
		flusher.Flush()

		// chunk 2: content
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"审核通过\"}}]}\n\n"))
		flusher.Flush()

		// chunk 3: done
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	caller, err := NewOpenAICompatCaller(&model.AIModelConfig{
		Provider:  "deepseek",
		Endpoint:  server.URL,
		ModelName: "deepseek-r1",
	})
	if err != nil {
		t.Fatalf("create caller failed: %v", err)
	}

	var streamedContent string
	var streamedReasoning string
	chatResp, err := caller.Chat(context.Background(), &ChatRequest{
		SystemPrompt:   "system",
		UserPrompt:     "user",
		EnableThinking: true,
		StreamChunkFunc: func(c string) {
			streamedContent += c
		},
		StreamReasoningChunkFunc: func(r string) {
			streamedReasoning += r
		},
	})
	if err != nil {
		t.Fatalf("Chat stream failed: %v", err)
	}

	if chatResp.Content != "审核通过" || streamedContent != "审核通过" {
		t.Errorf("expected content '审核通过', got content=%s, streamed=%s", chatResp.Content, streamedContent)
	}
	if chatResp.ReasoningContent != "深度思考中..." || streamedReasoning != "深度思考中..." {
		t.Errorf("expected reasoning '深度思考中...', got resp=%s, streamed=%s", chatResp.ReasoningContent, streamedReasoning)
	}
}
