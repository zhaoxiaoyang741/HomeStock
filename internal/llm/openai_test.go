package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestOpenAIProvider_Chat_TextResponse(t *testing.T) {
	var reqBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&reqBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {"role": "assistant", "content": "Hello, world!"},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(config.ModelConfig{
		Model:   "gpt-4o",
		APIKey:  "sk-test",
		APIBase: srv.URL,
	})

	resp, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hi"},
	}, nil, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello, world!" {
		t.Fatalf("expected 'Hello, world!', got %q", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("expected finish_reason 'stop', got %q", resp.FinishReason)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}

	// Verify the request was formed correctly
	if reqBody["model"] != "gpt-4o" {
		t.Fatalf("expected model 'gpt-4o', got %v", reqBody["model"])
	}
	if _, ok := reqBody["tools"]; ok {
		t.Fatal("tools should not be present when none are provided")
	}
}

func TestOpenAIProvider_Chat_ToolCallResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"role": "assistant",
					"content": null,
					"tool_calls": [{
						"id": "call_123",
						"type": "function",
						"function": {"name": "query_inventory", "arguments": "{\"keyword\": \"milk\"}"}
					}]
				},
				"finish_reason": "tool_calls"
			}],
			"usage": {"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30}
		}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(config.ModelConfig{
		Model:   "gpt-4o",
		APIKey:  "sk-test",
		APIBase: srv.URL,
	})

	resp, err := p.Chat(context.Background(), []Message{
		{Role: "user", Content: "查询库存"},
	}, []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "query_inventory",
				Description: "查询库存",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"keyword": map[string]any{"type": "string"},
					},
				},
			},
		},
	}, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Function.Name != "query_inventory" {
		t.Fatalf("expected tool 'query_inventory', got %q", resp.ToolCalls[0].Function.Name)
	}
	if resp.FinishReason != "tool_calls" {
		t.Fatalf("expected finish_reason 'tool_calls', got %q", resp.FinishReason)
	}
	if resp.Content != "" {
		t.Fatalf("expected empty content for tool call, got %q", resp.Content)
	}
}

func TestOpenAIProvider_Chat_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{
			"error": {
				"message": "Rate limit exceeded",
				"type": "rate_limit_error",
				"code": "rate_limited"
			}
		}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(config.ModelConfig{
		Model:   "gpt-4o",
		APIKey:  "sk-test",
		APIBase: srv.URL,
	})

	_, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "Hi"}}, nil, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "rate_limit_error") {
		t.Fatalf("error should mention rate_limit_error, got: %v", err)
	}
}

func TestOpenAIProvider_Chat_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [], "usage": {}}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(config.ModelConfig{
		Model:   "gpt-4o",
		APIKey:  "sk-test",
		APIBase: srv.URL,
	})

	_, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "Hi"}}, nil, "")
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestOpenAIProvider_DefaultModel(t *testing.T) {
	p := NewOpenAIProvider(config.ModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "sk-test",
	})
	if p.GetDefaultModel() != "gpt-4o-mini" {
		t.Fatalf("expected 'gpt-4o-mini', got %q", p.GetDefaultModel())
	}
}

func TestOpenAIProvider_Chat_WithModelOverride(t *testing.T) {
	var capturedModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		capturedModel, _ = body["model"].(string)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"role": "assistant", "content": "ok"}, "finish_reason": "stop"}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(config.ModelConfig{
		Model:   "gpt-4o",
		APIKey:  "sk-test",
		APIBase: srv.URL,
	})

	p.Chat(context.Background(), []Message{{Role: "user", Content: "Hi"}}, nil, "gpt-4o-mini")
	if capturedModel != "gpt-4o-mini" {
		t.Fatalf("expected 'gpt-4o-mini', got %q", capturedModel)
	}
}

func TestOpenAIProvider_BaseURLNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string // the base part before /chat/completions
	}{
		{"https://api.openai.com/v1", "https://api.openai.com/v1"},
		{"https://api.openai.com/v1/", "https://api.openai.com/v1"},
		{"https://api.openai.com/v1/chat/completions", "https://api.openai.com/v1"},
		{"https://api.deepseek.com", "https://api.deepseek.com/v1"},
		{"", "https://api.openai.com/v1"},
	}

	for _, tt := range tests {
		p := NewOpenAIProvider(config.ModelConfig{
			Model:   "test",
			APIKey:  "sk-test",
			APIBase: tt.input,
		})
		if p.apiBase != tt.expected {
			t.Errorf("NewOpenAIProvider(%q).apiBase = %q, want %q", tt.input, p.apiBase, tt.expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
