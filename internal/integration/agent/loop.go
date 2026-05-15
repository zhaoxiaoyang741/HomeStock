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
	nlu        *NluEngine // NLU engine for structured extraction (nil = fallback to LLM+tool)

	systemPrompt string

	histories  map[string][]llm.Message
	historyMu  sync.RWMutex
	maxHistory int

	sessions   map[string]*DialogSession
	sessionMu  sync.Mutex
	nameResolver NameResolver

	// Undo support (Phase 1a)
	opHistory   map[string][]OperationRecord // stack per chatID
	opHistoryMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewAgentLoop creates an AgentLoop.
func NewAgentLoop(bus *bus.MessageBus, provider llm.LLMProvider, dispatcher *tool.Dispatcher, systemPrompt string, nlu *NluEngine) *AgentLoop {
	l := &AgentLoop{
		bus:          bus,
		provider:     provider,
		dispatcher:   dispatcher,
		nlu:          nlu,
		systemPrompt: systemPrompt,
		histories:    make(map[string][]llm.Message),
		maxHistory:   defaultMaxHistory,
		sessions:     make(map[string]*DialogSession),
		opHistory:    make(map[string][]OperationRecord),
	}
	if nlu != nil {
		l.nameResolver = DefaultNameResolver(nlu.materialSvc)
	}
	return l
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

	// Undo command — highest priority, bypass dialog state
	if isUndoCommand(userContent) {
		undone := l.undoLastOperation(msg.ChatID)
		if undone {
			l.reply(msg, "已撤回上一条操作。")
		} else {
			l.reply(msg, "没有可撤回的操作。")
		}
		return
	}

	// No NLU engine? fallback to standard flow (backward compat)
	if l.nlu == nil {
		l.fallbackToStandardFlow(msg, actor, msg.ChatID)
		return
	}

	// Get or create session with timeout check
	session := l.getOrCreateSession(msg.ChatID)

	switch session.State {
	case StateIdle:
		l.handleIdleState(session, msg, actor)
	case StateClarifying:
		l.handleClarifyingState(session, msg, actor)
	case StateConfirming:
		l.handleConfirmingState(session, msg, actor)
	}
}

// ---------------------------------------------------------------------------
// Dialog state handlers
// ---------------------------------------------------------------------------

// handleIdleState processes a message when the session is Idle.
// NLU extract -> validate -> resolve names -> execute / clarify / confirm.
func (l *AgentLoop) handleIdleState(session *DialogSession, msg bus.InboundMessage, actor service.Actor) {
	text := l.nlu.SanitizeInput(msg.Text)
	if text == "" {
		l.fallbackToStandardFlow(msg, actor, msg.ChatID)
		return
	}

	recentCtx := l.buildRecentContext(msg.ChatID)
	result := l.nluCall(l.ctx, text, recentCtx, actor, msg.ChatID)
	if result == nil {
		l.fallbackToStandardFlow(msg, actor, msg.ChatID)
		return
	}

	// Append user message for context continuity
	l.appendHistory(msg.ChatID, llm.Message{Role: "user", Content: msg.Text})

	switch result.Intent {
	case "chitchat", "reject":
		l.fallbackToStandardFlow(msg, actor, msg.ChatID)
		return
	case "execute":
		if len(result.Actions) == 0 {
			l.fallbackToStandardFlow(msg, actor, msg.ChatID)
			return
		}

		// Validate required fields
		missing := l.validateActions(result.Actions)
		if len(missing) > 0 {
			session.State = StateClarifying
			session.PendingOp = &PendingOperation{
				ID:          newPendingOpID(),
				Actions:     result.Actions,
				MissingReqs: missing,
				FilledSlots: make(map[string][]Slot),
			}
			l.reply(msg, missing[0].Question)
			return
		}

		// Resolve names — check for ambiguity
		confirmItem := l.resolveItemNames(result.Actions, actor)
		if confirmItem != nil {
			session.State = StateConfirming
			session.PendingOp = &PendingOperation{
				ID:             newPendingOpID(),
				Actions:        result.Actions,
				FilledSlots:    make(map[string][]Slot),
				PendingConfirm: confirmItem,
			}
			confirmMsg := buildConfirmMessage(confirmItem.Name, confirmItem.Candidates)
			l.reply(msg, confirmMsg)
			return
		}

		// Execute directly
		reply := l.executeActions(result.Actions, msg, actor)
		l.reply(msg, reply)
		l.appendHistory(msg.ChatID, llm.Message{Role: "assistant", Content: reply})
		l.trimHistory(msg.ChatID)

		// Extract learnings for user memory
		learnings := l.extractMemoryFromActions(result.Actions)
		for _, lrn := range learnings {
			if err := l.nlu.AppendMemory(msg.ChatID, lrn); err != nil {
				logger.WarnCF("agent", "failed to save user memory", map[string]any{"error": err.Error()})
			}
		}

	case "clarify":
		l.reply(msg, "请描述得更清楚一些，您想做什么操作？")
	}
}

// handleClarifyingState processes a message when awaiting field clarification.
// NLU extract -> mergeToPending -> re-validate -> execute / re-ask / reset.
func (l *AgentLoop) handleClarifyingState(session *DialogSession, msg bus.InboundMessage, actor service.Actor) {
	pending := session.PendingOp
	if pending == nil {
		session.State = StateIdle
		l.processMessage(msg)
		return
	}

	text := l.nlu.SanitizeInput(msg.Text)
	if text == "" {
		l.reply(msg, pending.MissingReqs[0].Question)
		return
	}

	recentCtx := l.buildRecentContext(msg.ChatID)
	result := l.nluCall(l.ctx, text, recentCtx, actor, msg.ChatID)
	if result == nil {
		pending.AskCount++
		if pending.AskCount >= 3 {
			l.reply(msg, "抱歉，无法理解您的输入，请稍后重新尝试。")
			session.State = StateIdle
			session.PendingOp = nil
			return
		}
		l.reply(msg, pending.MissingReqs[0].Question)
		return
	}

	// Merge newly extracted info into pending operation
	l.mergeToPending(pending, result)

	// Re-validate — still missing fields?
	if len(pending.MissingReqs) > 0 {
		pending.AskCount++
		if pending.AskCount >= 3 {
			l.reply(msg, "抱歉，无法获取完整信息，请稍后重新尝试。")
			session.State = StateIdle
			session.PendingOp = nil
			return
		}
		l.reply(msg, pending.MissingReqs[0].Question)
		return
	}

	// All fields filled — apply slot values back to action items
	l.applySlots(pending)

	// Resolve names
	confirmItem := l.resolveItemNames(pending.Actions, actor)
	if confirmItem != nil {
		session.State = StateConfirming
		pending.PendingConfirm = confirmItem
		confirmMsg := buildConfirmMessage(confirmItem.Name, confirmItem.Candidates)
		l.reply(msg, confirmMsg)
		return
	}

	// Execute
	l.appendHistory(msg.ChatID, llm.Message{Role: "user", Content: msg.Text})
	reply := l.executeActions(pending.Actions, msg, actor)
	l.reply(msg, reply)
	l.appendHistory(msg.ChatID, llm.Message{Role: "assistant", Content: reply})
	l.trimHistory(msg.ChatID)

	// Extract learnings for user memory
	learnings := l.extractMemoryFromActions(pending.Actions)
	for _, lrn := range learnings {
		if err := l.nlu.AppendMemory(msg.ChatID, lrn); err != nil {
			logger.WarnCF("agent", "failed to save user memory", map[string]any{"error": err.Error()})
		}
	}

	session.State = StateIdle
	session.PendingOp = nil
}

// handleConfirmingState processes a message when awaiting name disambiguation.
// Parse A/B/C response -> resolve name -> execute / re-ask.
func (l *AgentLoop) handleConfirmingState(session *DialogSession, msg bus.InboundMessage, actor service.Actor) {
	pending := session.PendingOp
	if pending == nil || pending.PendingConfirm == nil {
		session.State = StateIdle
		l.processMessage(msg)
		return
	}

	text := strings.TrimSpace(msg.Text)
	ci := pending.PendingConfirm

	// Try parsing A/B/C choice first
	selected := parseConfirmChoice(text, ci.Candidates)
	if selected == nil && l.nameResolver != nil {
		// Try as a more specific name
		if candidates, err := l.nameResolver(l.ctx, text, actor.TenantID); err == nil && len(candidates) > 0 {
			selected = &candidates[0]
		}
	}
	if selected == nil {
		pending.AskCount++
		if pending.AskCount >= 3 {
			l.reply(msg, "抱歉，无法识别您选择的物料，请稍后重新尝试。")
			session.State = StateIdle
			session.PendingOp = nil
			return
		}
		l.reply(msg, "请回复 A、B、C 选择物料，或输入更精确的名称。")
		return
	}

	// Store resolved material ID
	if ci.ActionIdx < len(pending.Actions) && ci.ItemIdx < len(pending.Actions[ci.ActionIdx].Items) {
		pending.Actions[ci.ActionIdx].Items[ci.ItemIdx].ResolvedMaterialID = selected.MaterialID
		pending.Actions[ci.ActionIdx].Items[ci.ItemIdx].Name = selected.Name
	}
	pending.PendingConfirm = nil

	// Check for more ambiguous items
	confirmItem := l.resolveItemNames(pending.Actions, actor)
	if confirmItem != nil {
		pending.PendingConfirm = confirmItem
		confirmMsg := buildConfirmMessage(confirmItem.Name, confirmItem.Candidates)
		l.reply(msg, confirmMsg)
		return
	}

	// Execute
	l.appendHistory(msg.ChatID, llm.Message{Role: "user", Content: msg.Text})
	reply := l.executeActions(pending.Actions, msg, actor)
	l.reply(msg, reply)
	l.appendHistory(msg.ChatID, llm.Message{Role: "assistant", Content: reply})
	l.trimHistory(msg.ChatID)

	// Extract learnings for user memory
	learnings := l.extractMemoryFromActions(pending.Actions)
	for _, lrn := range learnings {
		if err := l.nlu.AppendMemory(msg.ChatID, lrn); err != nil {
			logger.WarnCF("agent", "failed to save user memory", map[string]any{"error": err.Error()})
		}
	}

	session.State = StateIdle
	session.PendingOp = nil
}

// ---------------------------------------------------------------------------
// NLU call helper
// ---------------------------------------------------------------------------

// nluCall builds the NLU prompt (with user memory injected), calls the LLM, and parses the result.
func (l *AgentLoop) nluCall(ctx context.Context, text, recentContext string, actor service.Actor, chatID string) *NluResult {
	userMemory := l.nlu.LoadMemory(chatID)
	catalog := l.nlu.PrefetchCatalog(ctx, text, actor.TenantID)
	sysPrompt := l.nlu.BuildNluSystemPrompt(catalog, recentContext, userMemory)

	messages := []llm.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: text},
	}

	l.providerMu.RLock()
	resp, err := l.provider.Chat(ctx, messages, nil, "")
	l.providerMu.RUnlock()
	if err != nil {
		logger.ErrorCF("agent", "NLU call failed", map[string]any{"error": err.Error()})
		return nil
	}

	result, err := l.nlu.ParseResponse(resp.Content)
	if err != nil {
		logger.ErrorCF("agent", "NLU parse failed", map[string]any{
			"error": err.Error(),
			"raw":   resp.Content,
		})
		return nil
	}
	result.RawText = text
	return result
}

// ---------------------------------------------------------------------------
// Fallback to standard LLM+tool flow
// ---------------------------------------------------------------------------

// fallbackToStandardFlow uses the original LLM+tool path for chitchat or NLU failures.
func (l *AgentLoop) fallbackToStandardFlow(msg bus.InboundMessage, actor service.Actor, chatID string) {
	l.appendHistory(chatID, llm.Message{Role: "user", Content: msg.Text})
	messages := l.buildConversation(chatID)
	tools := l.dispatcher.Definitions()

	response, err := l.chatWithTools(messages, tools, 0, actor, chatID, msg)
	if err != nil {
		logger.ErrorCF("agent", "chat failed", map[string]any{
			"chat_id": chatID,
			"error":   err.Error(),
		})
		l.reply(msg, "服务暂时不可用，请稍后重试。")
		return
	}

	l.reply(msg, response)
	l.appendHistory(chatID, llm.Message{Role: "assistant", Content: response})
	l.trimHistory(chatID)
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// validateActions checks required fields and returns missing requirements.
func (l *AgentLoop) validateActions(actions []ExtractedAction) []MissingRequirement {
	var reqs []MissingRequirement
	for i, action := range actions {
		if action.Type == "query" || action.Type == "undo" {
			continue
		}
		for j, item := range action.Items {
			if item.Name == "" {
				reqs = append(reqs, MissingRequirement{
					ActionIdx: i, ItemIdx: j, Field: "name",
					Question: "请问要操作哪个物品？",
				})
				continue
			}
			if item.Quantity == nil || *item.Quantity <= 0 {
				reqs = append(reqs, MissingRequirement{
					ActionIdx: i, ItemIdx: j, Field: "quantity",
					Question: fmt.Sprintf("「%s」的数量是多少？", item.Name),
				})
			}
		}
	}
	return reqs
}

// ---------------------------------------------------------------------------
// Name resolution
// ---------------------------------------------------------------------------

// resolveItemNames resolves item names to material IDs.
// Uniquely matched items get ResolvedMaterialID set.
// Returns the first ambiguous item needing user confirmation, or nil.
func (l *AgentLoop) resolveItemNames(actions []ExtractedAction, actor service.Actor) *PendingConfirmItem {
	if l.nameResolver == nil {
		return nil
	}
	for i := range actions {
		for j := range actions[i].Items {
			item := &actions[i].Items[j]
			if item.Name == "" || item.ResolvedMaterialID != "" {
				continue
			}
			candidates, err := l.nameResolver(l.ctx, item.Name, actor.TenantID)
			if err != nil || len(candidates) == 0 {
				continue
			}
			if needsConfirmation(candidates) {
				return &PendingConfirmItem{
					ActionIdx:  i,
					ItemIdx:    j,
					Name:       item.Name,
					Candidates: candidates,
				}
			}
			// Unique match — store the material ID
			item.ResolvedMaterialID = candidates[0].MaterialID
		}
	}
	return nil
}

// parseConfirmChoice interprets A/B/C/D/E letter responses for disambiguation.
func parseConfirmChoice(text string, candidates []ResolveResult) *ResolveResult {
	t := strings.TrimSpace(text)
	if t == "" {
		return nil
	}
	labels := map[string]int{"A": 0, "B": 1, "C": 2, "D": 3, "E": 4}
	if idx, ok := labels[strings.ToUpper(t)]; ok && idx < len(candidates) {
		return &candidates[idx]
	}
	return nil
}

// applySlots propagates FilledSlots values back to action items before execution.
func (l *AgentLoop) applySlots(pending *PendingOperation) {
	for key, slots := range pending.FilledSlots {
		if len(slots) == 0 {
			continue
		}
		// key format: "actionIdx.itemIdx.field"
		parts := strings.SplitN(key, ".", 3)
		if len(parts) != 3 {
			continue
		}
		var actionIdx, itemIdx int
		if n, _ := fmt.Sscanf(parts[0], "%d", &actionIdx); n != 1 {
			continue
		}
		if n, _ := fmt.Sscanf(parts[1], "%d", &itemIdx); n != 1 {
			continue
		}
		if actionIdx >= len(pending.Actions) || itemIdx >= len(pending.Actions[actionIdx].Items) {
			continue
		}

		field := parts[2]
		item := &pending.Actions[actionIdx].Items[itemIdx]
		slot := slots[len(slots)-1] // latest value

		switch field {
		case "quantity":
			if v, ok := slot.Value.(float64); ok {
				item.Quantity = &v
			}
		case "unit":
			if v, ok := slot.Value.(string); ok {
				item.Unit = v
			}
		case "location":
			if v, ok := slot.Value.(string); ok {
				item.Location = v
			}
		case "name":
			if v, ok := slot.Value.(string); ok {
				item.Name = v
			}
		case "material_id":
			if v, ok := slot.Value.(string); ok {
				item.ResolvedMaterialID = v
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Action execution
// ---------------------------------------------------------------------------

// executeActions dispatches tool calls for each action item and composes a reply.
func (l *AgentLoop) executeActions(actions []ExtractedAction, msg bus.InboundMessage, actor service.Actor) string {
	var replies []string
	ctx := l.ctx

	for _, action := range actions {
		switch action.Type {
		case "inbound":
			for _, item := range action.Items {
				args := l.buildInboundArgs(item)
				if args == nil {
					replies = append(replies, fmt.Sprintf("「%s」入库失败：缺少必要参数", item.Name))
					continue
				}
				result, err := l.dispatcher.Execute(ctx, actor, "inbound_stock", args)
				if err != nil {
					replies = append(replies, fmt.Sprintf("「%s」入库失败: %v", item.Name, err))
				} else {
					replies = append(replies, result)
					l.recordOperation(msg.ChatID, OperationRecord{
						ToolName: "inbound_stock",
						ArgsJSON: toJSON(args),
						Action:   "inbound",
					})
				}
			}

		case "consume":
			for _, item := range action.Items {
				args := l.buildConsumeArgs(item)
				if args == nil {
					replies = append(replies, fmt.Sprintf("「%s」出库失败：缺少必要参数", item.Name))
					continue
				}
				result, err := l.dispatcher.Execute(ctx, actor, "consume_material", args)
				if err != nil {
					replies = append(replies, fmt.Sprintf("「%s」出库失败: %v", item.Name, err))
				} else {
					replies = append(replies, result)
					l.recordOperation(msg.ChatID, OperationRecord{
						ToolName: "consume_material",
						ArgsJSON: toJSON(args),
						Action:   "consume",
					})
				}
			}

		case "query":
			keyword := ""
			if len(action.Items) > 0 {
				keyword = action.Items[0].Name
			}
			if keyword == "" && len(action.Items) > 0 && action.Items[0].Spec != "" {
				keyword = action.Items[0].Spec
			}
			args := map[string]any{"keyword": keyword, "show_zero_stock": false}
			result, err := l.dispatcher.Execute(ctx, actor, "query_inventory", args)
			if err != nil {
				replies = append(replies, fmt.Sprintf("查询失败: %v", err))
			} else {
				replies = append(replies, result)
			}

		default:
			replies = append(replies, fmt.Sprintf("不支持的操作: %s", action.Type))
		}
	}
	return strings.Join(replies, "\n")
}

// buildInboundArgs constructs arguments for inbound_stock from an extracted item.
// If ResolvedMaterialID is empty, the service layer will auto-create the material by name.
func (l *AgentLoop) buildInboundArgs(item ExtractedItem) map[string]any {
	if item.Quantity == nil || *item.Quantity <= 0 {
		return nil
	}
	args := map[string]any{
		"name":     item.Name,
		"quantity": *item.Quantity,
	}
	if item.ResolvedMaterialID != "" {
		args["material_id"] = item.ResolvedMaterialID
	}
	if item.Unit != "" {
		args["unit"] = item.Unit
	}
	if item.Location != "" {
		args["location"] = item.Location
	}
	if item.ExpireAt != "" {
		args["expire_at"] = item.ExpireAt
	}
	return args
}

// buildConsumeArgs constructs arguments for consume_material from an extracted item.
func (l *AgentLoop) buildConsumeArgs(item ExtractedItem) map[string]any {
	materialID := item.ResolvedMaterialID
	if materialID == "" {
		return nil
	}
	if item.Quantity == nil || *item.Quantity <= 0 {
		return nil
	}
	return map[string]any{
		"material_id": materialID,
		"quantity":    *item.Quantity,
		"reason":      "用户操作",
	}
}

// extractMemoryFromActions generates learnable preference entries from executed actions.
// These are stored in the user's memory file to reduce confirmations in future interactions.
func (l *AgentLoop) extractMemoryFromActions(actions []ExtractedAction) []string {
	var learnings []string
	for _, action := range actions {
		if action.Type != "inbound" {
			continue
		}
		for _, item := range action.Items {
			var parts []string
			if item.Location != "" {
				parts = append(parts, "存放位置: "+item.Location)
			}
			if item.Unit != "" {
				parts = append(parts, "默认单位: "+item.Unit)
			}
			if len(parts) > 0 {
				learnings = append(learnings, item.Name+" -> "+strings.Join(parts, ", "))
			}
		}
	}
	return learnings
}

// ---------------------------------------------------------------------------
// Context helpers
// ---------------------------------------------------------------------------

// buildRecentContext returns the last 2 conversation turns as formatted text.
func (l *AgentLoop) buildRecentContext(chatID string) string {
	l.historyMu.RLock()
	defer l.historyMu.RUnlock()

	hist := l.histories[chatID]
	if len(hist) == 0 {
		return ""
	}

	var b strings.Builder
	start := 0
	if len(hist) > 4 {
		start = len(hist) - 4
	}
	for _, m := range hist[start:] {
		role := "用户"
		if m.Role == "assistant" {
			role = "助手"
		}
		b.WriteString(fmt.Sprintf("%s: %s\n", role, m.Content))
	}
	return b.String()
}

// toJSON marshals a value to a JSON string.
func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
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
