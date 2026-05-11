package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/llm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/internal/tool"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

const (
	defaultMaxHistory = 10
	maxToolCallDepth  = 10
)

// AgentLoop is the core message processing orchestrator.
//
// It consumes InboundMessage from the MessageBus, constructs conversations,
// calls the LLM, executes tool calls, and publishes OutboundMessage responses.
type AgentLoop struct {
	bus        *MessageBus
	provider   llm.LLMProvider
	providerMu sync.RWMutex
	dispatcher *tool.Dispatcher

	systemPrompt string

	histories  map[string][]llm.Message
	historyMu  sync.RWMutex
	maxHistory int

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewAgentLoop creates an AgentLoop.
func NewAgentLoop(bus *MessageBus, provider llm.LLMProvider, dispatcher *tool.Dispatcher, systemPrompt string) *AgentLoop {
	return &AgentLoop{
		bus:          bus,
		provider:     provider,
		dispatcher:   dispatcher,
		systemPrompt: systemPrompt,
		histories:    make(map[string][]llm.Message),
		maxHistory:   defaultMaxHistory,
	}
}

// Start begins the agent loop in a background goroutine.
func (l *AgentLoop) Start(ctx context.Context) {
	l.ctx, l.cancel = context.WithCancel(ctx)
	l.wg.Add(1)
	go l.run()
}

// Stop gracefully stops the agent loop.
func (l *AgentLoop) Stop() {
	if l.cancel != nil {
		l.cancel()
	}
	l.wg.Wait()
}

// SwapProvider atomically replaces the LLM provider at runtime.
func (l *AgentLoop) SwapProvider(provider llm.LLMProvider) {
	l.providerMu.Lock()
	defer l.providerMu.Unlock()
	l.provider = provider
}

func (l *AgentLoop) run() {
	defer l.wg.Done()
	logger.InfoCF("agent", "agent loop started", nil)

	for {
		select {
		case msg := <-l.bus.InboundChan():
			l.processMessage(msg)
		case <-l.ctx.Done():
			logger.InfoCF("agent", "agent loop stopped", nil)
			return
		case <-l.bus.Done():
			return
		}
	}
}

func (l *AgentLoop) processMessage(msg InboundMessage) {
	logger.InfoCF("agent", "processing message", map[string]any{
		"channel": msg.Channel,
		"chat_id": msg.ChatID,
		"sender":  msg.SenderName,
	})

	actor := service.Actor{
		Channel:  msg.Channel,
		UserName: msg.SenderName,
		UserID:   msg.SenderID,
		TenantID: "default",
	}

	userContent := msg.Text
	if msg.MediaType != "" && msg.MediaType != "text" {
		userContent = fmt.Sprintf("[%s消息暂不支持，请发送文字消息]", msg.MediaType)
	}

	l.appendHistory(msg.ChatID, llm.Message{Role: "user", Content: userContent})

	// Build conversation — system prompt + stored history (which now includes the user message)
	messages := l.buildConversation(msg.ChatID)
	tools := l.dispatcher.Definitions()

	response, err := l.chatWithTools(messages, tools, 0, actor)
	if err != nil {
		logger.ErrorCF("agent", "chat failed", map[string]any{
			"channel": msg.ChatID,
			"chat_id": msg.ChatID,
			"error":   err.Error(),
		})
		l.reply(msg, "服务暂时不可用，请稍后重试。")
		return
	}

	l.reply(msg, response)

	// Update final assistant response in history
	l.appendHistory(msg.ChatID, llm.Message{Role: "assistant", Content: response})

	l.trimHistory(msg.ChatID)
}

// buildConversation constructs the message list for the LLM call.
func (l *AgentLoop) buildConversation(chatID string) []llm.Message {
	msgs := []llm.Message{
		{Role: "system", Content: l.systemPrompt},
	}
	l.historyMu.RLock()
	msgs = append(msgs, l.histories[chatID]...)
	l.historyMu.RUnlock()
	return msgs
}

// chatWithTools sends messages to the LLM and handles tool call loops.
func (l *AgentLoop) chatWithTools(messages []llm.Message, tools []llm.ToolDefinition, depth int, actor service.Actor) (string, error) {
	if depth > maxToolCallDepth {
		return "抱歉，操作步骤过多，请简化您的请求。", nil
	}

	l.providerMu.RLock()
	resp, err := l.provider.Chat(l.ctx, messages, tools, "")
	l.providerMu.RUnlock()
	if err != nil {
		return "", fmt.Errorf("llm chat: %w", err)
	}

	// Store assistant response (may contain tool_calls) into messages for the next iteration
	messages = append(messages, llm.Message{
		Role:              "assistant",
		Content:           resp.Content,
		ReasoningContent:  resp.ReasoningContent,
		ToolCalls:         resp.ToolCalls,
	})

	// Text response — done
	if len(resp.ToolCalls) == 0 {
		return resp.Content, nil
	}

	// Execute tool calls

	for _, tc := range resp.ToolCalls {
		var args map[string]any
		if tc.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				args = map[string]any{}
			}
		}

		logger.InfoCF("agent", "executing tool", map[string]any{
			"tool": tc.Function.Name,
			"args": tc.Function.Arguments,
		})

		start := time.Now()
		result, err := l.dispatcher.Execute(l.ctx, actor, tc.Function.Name, args)
		elapsed := time.Since(start)

		content := result
		if err != nil {
			content = fmt.Sprintf("执行出错: %v", err)
			logger.ErrorCF("agent", "tool execution failed", map[string]any{
				"tool":    tc.Function.Name,
				"error":   err.Error(),
				"elapsed": elapsed.String(),
			})
		} else {
			logger.InfoCF("agent", "tool executed", map[string]any{
				"tool":    tc.Function.Name,
				"elapsed": elapsed.String(),
			})
		}

		messages = append(messages, llm.Message{
			Role:       "tool",
			ToolCallID: tc.ID,
			Name:       tc.Function.Name,
			Content:    content,
		})
	}

	// Continue loop so the LLM can produce a follow-up text response
	return l.chatWithTools(messages, tools, depth+1, actor)
}

func (l *AgentLoop) reply(msg InboundMessage, text string) {
	if text == "" {
		return
	}
	out := OutboundMessage{
		Channel: msg.Channel,
		ChatID:  msg.ChatID,
		Text:    text,
	}
	if err := l.bus.PublishOutbound(l.ctx, out); err != nil {
		logger.ErrorCF("agent", "publish outbound failed", map[string]any{
			"channel": msg.Channel,
			"chat_id": msg.ChatID,
			"error":   err.Error(),
		})
	}
}

func (l *AgentLoop) appendHistory(chatID string, msg llm.Message) {
	l.historyMu.Lock()
	defer l.historyMu.Unlock()
	l.histories[chatID] = append(l.histories[chatID], msg)
}

// trimHistory keeps only the last N user+assistant turns per chat_id.
func (l *AgentLoop) trimHistory(chatID string) {
	l.historyMu.Lock()
	defer l.historyMu.Unlock()

	hist := l.histories[chatID]
	if len(hist) <= l.maxHistory*4 { // each turn ≈ 4 messages (user + asst + tools + asst)
		return
	}

	turnCount := 0
	cut := 0
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Role == "user" || hist[i].Role == "assistant" {
			turnCount++
		}
		// keep the system prompt prefix if it's there
		if turnCount > l.maxHistory {
			cut = i + 1
			break
		}
	}
	if cut > 0 && cut < len(hist) {
		l.histories[chatID] = hist[cut:]
	}
}
