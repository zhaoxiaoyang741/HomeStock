package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/tool"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/bus"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/llm"
)

// mockProvider implements llm.LLMProvider for testing.
type mockProvider struct {
	mu        sync.Mutex
	responses []*llm.LLMResponse // each call returns the next response
	index     int
}

func (m *mockProvider) addResponse(resp *llm.LLMResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = append(m.responses, resp)
}

func (m *mockProvider) Chat(_ context.Context, msgs []llm.Message, _ []llm.ToolDefinition, _ string) (*llm.LLMResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.index >= len(m.responses) {
		panic("mockProvider: no more responses")
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

func (m *mockProvider) GetDefaultModel() string { return "mock-model" }

func TestAgentLoop_TextResponse(t *testing.T) {
	mb := bus.NewMessageBus(8)
	provider := &mockProvider{}
	disp := tool.NewDispatcher()

	provider.addResponse(&llm.LLMResponse{
		Content:      "你好！有什么可以帮你的？",
		FinishReason: "stop",
		Usage:        llm.UsageInfo{PromptTokens: 10, CompletionTokens: 5},
	})

	loop := NewAgentLoop(mb, provider, disp, "你是 HomeStock 助手。")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loop.Start(ctx)

	// Send an inbound message
	mb.PublishInbound(ctx, bus.InboundMessage{
		Channel:    "feishu",
		ChatID:     "chat_1",
		SenderID:   "user_1",
		SenderName: "TestUser",
		Text:       "你好",
	})

	// Read the outbound
	out := <-mb.OutboundChan()
	if out.ChatID != "chat_1" {
		t.Fatalf("expected chat_1, got %q", out.ChatID)
	}
	if out.Text != "你好！有什么可以帮你的？" {
		t.Fatalf("unexpected response: %q", out.Text)
	}
}

func TestAgentLoop_ToolCallThenText(t *testing.T) {
	mb := bus.NewMessageBus(8)
	provider := &mockProvider{}
	disp := tool.NewDispatcher()

	disp.Register("query_inventory", func(_ context.Context, _ service.Actor, args map[string]any) (string, error) {
		keyword, _ := args["keyword"].(string)
		return `当前库存有 5 件商品 "` + keyword + `"。`, nil
	})

	// First response: tool call
	provider.addResponse(&llm.LLMResponse{
		ToolCalls: []llm.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: llm.FunctionCall{
					Name:      "query_inventory",
					Arguments: `{"keyword": "牛奶"}`,
				},
			},
		},
		FinishReason: "tool_calls",
		Usage:        llm.UsageInfo{PromptTokens: 20, CompletionTokens: 10},
	})

	// Second response: text
	provider.addResponse(&llm.LLMResponse{
		Content:      "目前库存有 5 件牛奶商品。",
		FinishReason: "stop",
		Usage:        llm.UsageInfo{PromptTokens: 30, CompletionTokens: 5},
	})

	loop := NewAgentLoop(mb, provider, disp, "你是 HomeStock 助手。")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loop.Start(ctx)

	mb.PublishInbound(ctx, bus.InboundMessage{
		Channel:    "feishu",
		ChatID:     "chat_2",
		SenderID:   "user_1",
		SenderName: "TestUser",
		Text:       "查询牛奶库存",
	})

	out := <-mb.OutboundChan()
	if out.ChatID != "chat_2" {
		t.Fatalf("expected chat_2, got %q", out.ChatID)
	}
	if out.Text != "目前库存有 5 件牛奶商品。" {
		t.Fatalf("unexpected response: %q", out.Text)
	}
}

func TestAgentLoop_HistoryPreserved(t *testing.T) {
	mb := bus.NewMessageBus(8)
	provider := &mockProvider{}
	disp := tool.NewDispatcher()

	provider.addResponse(&llm.LLMResponse{
		Content:      "回复1",
		FinishReason: "stop",
	})
	provider.addResponse(&llm.LLMResponse{
		Content:      "回复2",
		FinishReason: "stop",
	})

	loop := NewAgentLoop(mb, provider, disp, "你是 HomeStock 助手。")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loop.Start(ctx)

	// First message
	mb.PublishInbound(ctx, bus.InboundMessage{
		Channel: "feishu", ChatID: "chat_3", SenderID: "u1", Text: "消息1",
	})
	out1 := <-mb.OutboundChan()
	if out1.Text != "回复1" {
		t.Fatalf("expected '回复1', got %q", out1.Text)
	}

	// Second message
	mb.PublishInbound(ctx, bus.InboundMessage{
		Channel: "feishu", ChatID: "chat_3", SenderID: "u1", Text: "消息2",
	})
	out2 := <-mb.OutboundChan()
	if out2.Text != "回复2" {
		t.Fatalf("expected '回复2', got %q", out2.Text)
	}

	// Verify history has accumulated messages (user + assistant from both turns)
	loop.historyMu.RLock()
	histLen := len(loop.histories["chat_3"])
	loop.historyMu.RUnlock()

	if histLen < 4 {
		t.Fatalf("expected at least 4 history entries, got %d", histLen)
	}

	// Verify the first user message is still in history
	loop.historyMu.RLock()
	defer loop.historyMu.RUnlock()
	if loop.histories["chat_3"][0].Role != "user" || loop.histories["chat_3"][0].Content != "消息1" {
		t.Fatalf("expected first history entry to be '消息1', got %+v", loop.histories["chat_3"][0])
	}
}

func TestAgentLoop_MediaNotSupported(t *testing.T) {
	mb := bus.NewMessageBus(8)
	provider := &mockProvider{}
	disp := tool.NewDispatcher()

	provider.addResponse(&llm.LLMResponse{
		Content:      "暂不支持语音消息。",
		FinishReason: "stop",
	})

	loop := NewAgentLoop(mb, provider, disp, "你是 HomeStock 助手。")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loop.Start(ctx)

	mb.PublishInbound(ctx, bus.InboundMessage{
		Channel: "feishu", ChatID: "chat_4", SenderID: "u1",
		Text: "语音内容", MediaType: "voice", FileKey: "file_1",
	})

	out := <-mb.OutboundChan()
	// The LLM gets a "[voice消息暂不支持，请发送文字消息]" message
	if out.Text != "暂不支持语音消息。" {
		t.Fatalf("unexpected: %q", out.Text)
	}
}

func TestAgentLoop_ToolExecutionFailure(t *testing.T) {
	mb := bus.NewMessageBus(8)
	provider := &mockProvider{}
	disp := tool.NewDispatcher()

	disp.Register("fail_tool", func(_ context.Context, _ service.Actor, _ map[string]any) (string, error) {
		return "", nil // but the tool dispatcher will return error
	})

	// Actually, let's register a tool that returns an error
	disp.Register("error_tool", func(_ context.Context, _ service.Actor, _ map[string]any) (string, error) {
		return "", nil
	})
	// Override with a failing one
	disp.Register("fail_tool_direct", func(_ context.Context, _ service.Actor, _ map[string]any) (string, error) {
		return "", nil
	})

	// First response: tool call that doesn't exist
	provider.addResponse(&llm.LLMResponse{
		ToolCalls: []llm.ToolCall{
			{
				ID:   "call_err",
				Type: "function",
				Function: llm.FunctionCall{
					Name:      "nonexistent_tool",
					Arguments: `{}`,
				},
			},
		},
		FinishReason: "tool_calls",
	})

	// Second response: after seeing the error
	provider.addResponse(&llm.LLMResponse{
		Content:      "抱歉，我没有找到这个工具。",
		FinishReason: "stop",
	})

	loop := NewAgentLoop(mb, provider, disp, "你是 HomeStock 助手。")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loop.Start(ctx)

	mb.PublishInbound(ctx, bus.InboundMessage{
		Channel: "feishu", ChatID: "chat_5", SenderID: "u1", Text: "测试",
	})

	out := <-mb.OutboundChan()
	if out.Text != "抱歉，我没有找到这个工具。" {
		t.Fatalf("unexpected: %q", out.Text)
	}
}

func TestAgentLoop_LLMError(t *testing.T) {
	mb := bus.NewMessageBus(8)
	provider := &mockProvider{}
	disp := tool.NewDispatcher()

	loop := NewAgentLoop(mb, provider, disp, "你是 HomeStock 助手。")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loop.Start(ctx)

	provider.addResponse(&llm.LLMResponse{
		Content:      "正常回复",
		FinishReason: "stop",
	})

	mb.PublishInbound(ctx, bus.InboundMessage{
		Channel: "feishu", ChatID: "chat_6", SenderID: "u1", Text: "hello",
	})

	out := <-mb.OutboundChan()
	if out.Text != "正常回复" {
		t.Fatalf("expected normal reply, got %q", out.Text)
	}
}

// TestAgentLoop_MaxDepth tests that the tool call depth limit works.
func TestAgentLoop_MaxDepth(t *testing.T) {
	mb := bus.NewMessageBus(8)
	provider := &mockProvider{}
	disp := tool.NewDispatcher()

	disp.Register("chain_tool", func(_ context.Context, _ service.Actor, _ map[string]any) (string, error) {
		return "工具执行成功", nil
	})

	// Add many tool call responses to exceed maxToolCallDepth
	for i := 0; i < maxToolCallDepth+2; i++ {
		provider.addResponse(&llm.LLMResponse{
			ToolCalls: []llm.ToolCall{
				{
					ID:   "call_chain",
					Type: "function",
					Function: llm.FunctionCall{
						Name:      "chain_tool",
						Arguments: `{}`,
					},
				},
			},
			FinishReason: "tool_calls",
		})
	}

	loop := NewAgentLoop(mb, provider, disp, "你是 HomeStock 助手。")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loop.Start(ctx)

	mb.PublishInbound(ctx, bus.InboundMessage{
		Channel: "feishu", ChatID: "chat_7", SenderID: "u1", Text: "do chain",
	})

	out := <-mb.OutboundChan()
	if out.Text != "抱歉，操作步骤过多，请简化您的请求。" {
		t.Fatalf("expected depth limit message, got %q", out.Text)
	}
}

func TestMessageBus_PublishInboundBlocked(t *testing.T) {
	mb := bus.NewMessageBus(1) // small buffer — 1 slot
	ctx := context.Background()

	// Fill the buffer
	err := mb.PublishInbound(ctx, bus.InboundMessage{Text: "first"})
	if err != nil {
		t.Fatalf("first send should succeed: %v", err)
	}

	// Buffer is full. Cancel context and try again — the cancelled
	// context's Done() channel should be selected before the blocked send.
	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	err = mb.PublishInbound(ctx2, bus.InboundMessage{Text: "second"})
	if err == nil {
		t.Fatal("expected error on full buffer with cancelled context")
	}
}

func TestMessageBus_Close(t *testing.T) {
	mb := bus.NewMessageBus(8)
	mb.Close()

	ctx := context.Background()
	err := mb.PublishInbound(ctx, bus.InboundMessage{Text: "test"})
	if err == nil {
		t.Fatal("expected error after close")
	}

	// Also verify outbound returns error after close
	err = mb.PublishOutbound(ctx, bus.OutboundMessage{Text: "test"})
	if err == nil {
		t.Fatal("expected error after close on outbound")
	}

	// Double close should not panic
	mb.Close()
}

func TestJSONUnmarshalArgs(t *testing.T) {
	// Verify that JSON unmarshaling of tool arguments works
	raw := `{"keyword": "牛奶", "quantity": 5}`
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if args["keyword"] != "牛奶" {
		t.Fatalf("expected '牛奶', got %v", args["keyword"])
	}
	// JSON numbers become float64 in Go
	if qty, ok := args["quantity"].(float64); !ok || qty != 5.0 {
		t.Fatalf("expected 5.0, got %v", args["quantity"])
	}
}

// Ensure tool definitions can be set and retrieved from the dispatcher
func TestToolDispatcher_Definitions(t *testing.T) {
	disp := tool.NewDispatcher()
	defs := []llm.ToolDefinition{
		{
			Type: "function",
			Function: llm.ToolFunctionDefinition{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
	}
	disp.SetDefinitions(defs)
	got := disp.Definitions()
	if len(got) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(got))
	}
	if got[0].Function.Name != "test_tool" {
		t.Fatalf("expected 'test_tool', got %q", got[0].Function.Name)
	}
}
