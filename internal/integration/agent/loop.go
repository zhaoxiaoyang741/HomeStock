package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/tool"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/bus"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/llm"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

const (
	defaultMaxHistory = 10
	maxToolCallDepth  = 10
)

// OperationRecord records a successfully executed tool call for undo support.
type OperationRecord struct {
	ToolName string
	ArgsJSON string // original JSON arguments
	Action   string // "inbound" or "consume"
}

// AgentLoop is the core message processing orchestrator.
//
// It consumes bus.InboundMessage from the bus.MessageBus, constructs conversations,
// calls the LLM, executes tool calls, and publishes bus.OutboundMessage responses.
type AgentLoop struct {
	bus        *bus.MessageBus
	provider   llm.LLMProvider
	providerMu sync.RWMutex
	dispatcher *tool.Dispatcher

	systemPrompt string

	histories  map[string][]llm.Message
	historyMu  sync.RWMutex
	maxHistory int

	// Undo support (Phase 1a)
	opHistory   map[string][]OperationRecord // stack per chatID
	opHistoryMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewAgentLoop creates an AgentLoop.
func NewAgentLoop(bus *bus.MessageBus, provider llm.LLMProvider, dispatcher *tool.Dispatcher, systemPrompt string) *AgentLoop {
	return &AgentLoop{
		bus:          bus,
		provider:     provider,
		dispatcher:   dispatcher,
		systemPrompt: systemPrompt,
		histories:    make(map[string][]llm.Message),
		maxHistory:   defaultMaxHistory,
		opHistory:    make(map[string][]OperationRecord),
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

func (l *AgentLoop) processMessage(msg bus.InboundMessage) {
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

	// Undo command — highest priority
	if isUndoCommand(userContent) {
		undone := l.undoLastOperation(msg.ChatID)
		if undone {
			l.reply(msg, "已撤回上一条操作。")
		} else {
			l.reply(msg, "没有可撤回的操作。")
		}
		return
	}

	// Normal flow — append user message and chat with tools
	l.appendHistory(msg.ChatID, llm.Message{Role: "user", Content: userContent})

	messages := l.buildConversation(msg.ChatID)
	tools := l.dispatcher.Definitions()

	response, err := l.chatWithTools(messages, tools, 0, actor, msg.ChatID, msg)
	if err != nil {
		logger.ErrorCF("agent", "chat failed", map[string]any{
			"chat_id": msg.ChatID,
			"error":   err.Error(),
		})
		l.reply(msg, "服务暂时不可用，请稍后重试。")
		return
	}

	l.reply(msg, response)

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
func (l *AgentLoop) chatWithTools(messages []llm.Message, tools []llm.ToolDefinition, depth int, actor service.Actor, chatID string, msg bus.InboundMessage) (string, error) {
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
		Role:             "assistant",
		Content:          resp.Content,
		ReasoningContent: resp.ReasoningContent,
		ToolCalls:        resp.ToolCalls,
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
			// Record successful tool execution for undo (Phase 1a)
			l.recordOperation(chatID, OperationRecord{
				ToolName: tc.Function.Name,
				ArgsJSON: tc.Function.Arguments,
				Action:   classifyToolAction(tc.Function.Name),
			})
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
	return l.chatWithTools(messages, tools, depth+1, actor, chatID, msg)
}

// ---------------------------------------------------------------------------
// Undo support
// ---------------------------------------------------------------------------

func (l *AgentLoop) recordOperation(chatID string, op OperationRecord) {
	l.opHistoryMu.Lock()
	defer l.opHistoryMu.Unlock()
	l.opHistory[chatID] = append(l.opHistory[chatID], op)
}

func (l *AgentLoop) popLastOperation(chatID string) (OperationRecord, bool) {
	l.opHistoryMu.Lock()
	defer l.opHistoryMu.Unlock()
	stack := l.opHistory[chatID]
	if len(stack) == 0 {
		return OperationRecord{}, false
	}
	last := stack[len(stack)-1]
	l.opHistory[chatID] = stack[:len(stack)-1]
	return last, true
}

// undoLastOperation reverses the last recorded tool operation for a chat.
func (l *AgentLoop) undoLastOperation(chatID string) bool {
	op, ok := l.popLastOperation(chatID)
	if !ok {
		return false
	}

	// Parse original args
	var args map[string]any
	if err := json.Unmarshal([]byte(op.ArgsJSON), &args); err != nil {
		logger.WarnCF("agent", "undo: failed to parse original args", map[string]any{
			"chat_id": chatID,
			"tool":    op.ToolName,
			"error":   err.Error(),
		})
		return false
	}

	// Build inverse operation
	ctx := context.Background()
	actor := service.Actor{
		Channel:  "system",
		UserName: "system",
		UserID:   "system",
		TenantID: "default",
	}

	scenarios := l.undoMapping(op.ToolName, args)
	for _, s := range scenarios {
		if s.skip {
			continue
		}
		if _, err := l.dispatcher.Execute(ctx, actor, s.toolName, s.args); err != nil {
			logger.ErrorCF("agent", "undo: inverse operation failed", map[string]any{
				"chat_id": chatID,
				"tool":    s.toolName,
				"error":   err.Error(),
			})
		}
	}
	return true
}

type undoScenario struct {
	toolName string
	args     map[string]any
	skip     bool
}

// undoMapping returns the inverse operations for a given tool call.
func (l *AgentLoop) undoMapping(toolName string, args map[string]any) []undoScenario {
	switch toolName {
	case "inbound_stock":
		// Inverse: consume the same quantity
		qty, _ := args["quantity"].(float64)
		name, _ := args["name"].(string)
		if qty <= 0 || name == "" {
			return []undoScenario{{skip: true}}
		}
		// We need the material ID. Try to look it up by name.
		materialID := l.resolveMaterialID(name)
		if materialID == "" {
			return []undoScenario{{skip: true}}
		}
		return []undoScenario{{
			toolName: "consume_material",
			args: map[string]any{
				"material_id": materialID,
				"quantity":    qty,
				"reason":      "撤回入库",
			},
		}}

	case "consume_material":
		// Inverse: inbound the same quantity
		qty, _ := args["quantity"].(float64)
		materialID, _ := args["material_id"].(string)
		if qty <= 0 || materialID == "" {
			return []undoScenario{{skip: true}}
		}
		return []undoScenario{{
			toolName: "inbound_stock",
			args: map[string]any{
				"material_id": materialID,
				"quantity":    qty,
				"notes":       "撤回消耗",
			},
		}}

	default:
		// query_inventory, update_stock_lot, health check — no undo
		return []undoScenario{{skip: true}}
	}
}

// resolveMaterialID looks up a material ID by name. Returns empty if not found.
func (l *AgentLoop) resolveMaterialID(name string) string {
	ctx := context.Background()
	actor := service.Actor{
		Channel:  "system",
		UserName: "system",
		UserID:   "system",
		TenantID: "default",
	}
	result, err := l.dispatcher.Execute(ctx, actor, "query_inventory", map[string]any{
		"keyword":         name,
		"show_zero_stock": true,
	})
	if err != nil || result == "" {
		return ""
	}
	return extractMaterialID(result)
}

// ---------------------------------------------------------------------------
// Reply and history helpers
// ---------------------------------------------------------------------------

func (l *AgentLoop) reply(msg bus.InboundMessage, text string) {
	if text == "" {
		return
	}
	out := bus.OutboundMessage{
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

func (l *AgentLoop) trimHistory(chatID string) {
	l.historyMu.Lock()
	defer l.historyMu.Unlock()

	hist := l.histories[chatID]
	if len(hist) <= l.maxHistory*4 {
		return
	}

	turnCount := 0
	cut := 0
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Role == "user" || hist[i].Role == "assistant" {
			turnCount++
		}
		if turnCount > l.maxHistory {
			cut = i + 1
			break
		}
	}
	if cut > 0 && cut < len(hist) {
		l.histories[chatID] = hist[cut:]
	}
}

// ---------------------------------------------------------------------------
// Material ID extraction
// ---------------------------------------------------------------------------

// extractMaterialID parses a material_id from tool result text.
func extractMaterialID(result string) string {
	prefix := "material_id:"
	if idx := strings.Index(result, prefix); idx >= 0 {
		rest := result[idx+len(prefix):]
		rest = strings.TrimSpace(rest)
		if end := strings.IndexAny(rest, " \n\t,"); end >= 0 {
			return rest[:end]
		}
		return rest
	}
	return ""
}

// ---------------------------------------------------------------------------
// Command classification helpers
// ---------------------------------------------------------------------------

// classifyToolAction maps a tool name to a high-level action for undo.
func classifyToolAction(toolName string) string {
	switch toolName {
	case "inbound_stock":
		return "inbound"
	case "consume_material":
		return "consume"
	default:
		return "other"
	}
}

func isUndoCommand(text string) bool {
	t := strings.TrimSpace(text)
	return t == "撤回" || t == "撤销" || strings.HasPrefix(t, "撤回上") || strings.HasPrefix(t, "撤销上")
}
