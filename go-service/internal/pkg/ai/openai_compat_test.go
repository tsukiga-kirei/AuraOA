package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"auraoa/go-service/internal/model"
)

func TestOpenAICompatCaller_TestConnection_SendsJSONContentType(t *testing.T) {
	var gotContentType, gotAuth, gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	caller, err := NewOpenAICompatCaller(&model.AIModelConfig{
		Provider: "openai",
		Endpoint: server.URL + "/v1",
		APIKey:   "sk-test",
	})
	if err != nil {
		t.Fatalf("create caller failed: %v", err)
	}

	if err := caller.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection failed: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if gotPath != "/v1/models" {
		t.Errorf("expected /v1/models, got %s", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", gotContentType)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("expected Authorization Bearer sk-test, got %q", gotAuth)
	}
}

func TestOpenAICompatCaller_TestConnection_RetriesWithoutContentType(t *testing.T) {
	var contentTypes []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		contentTypes = append(contentTypes, ct)
		if ct == "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	caller, err := NewOpenAICompatCaller(&model.AIModelConfig{
		Provider: "vllm",
		Endpoint: server.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("create caller failed: %v", err)
	}
	if err := caller.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection failed: %v", err)
	}
	if len(contentTypes) != 2 {
		t.Fatalf("expected 2 probes, got %d (%v)", len(contentTypes), contentTypes)
	}
	if contentTypes[0] != "application/json" || contentTypes[1] != "" {
		t.Errorf("expected json then empty Content-Type, got %v", contentTypes)
	}
}

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
		resp.Choices = append(resp.Choices, openAIChoice{
			Message: openAIChoiceMessage{
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
