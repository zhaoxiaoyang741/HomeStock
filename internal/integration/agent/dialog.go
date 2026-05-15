package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

var opIDCounter int64

// DialogState represents the current state of a conversation session.
type DialogState int

const (
	StateIdle DialogState = iota
	StateClarifying
	StateConfirming
)

const (
	sessionIdleTimeout = 5 * time.Minute
)

// DialogSession tracks the state machine for a single chat conversation.
type DialogSession struct {
	ChatID         string
	State          DialogState
	PendingOp      *PendingOperation
	LastActivityAt time.Time
	mu             sync.Mutex
	CreatedAt      time.Time
}

// PendingOperation holds partial action data while waiting for user clarification.
type PendingOperation struct {
	ID              string
	Actions         []ExtractedAction
	FilledSlots     map[string][]Slot // key: "actionIdx.fieldName"
	MissingReqs     []MissingRequirement
	LastAsked       string
	AskCount        int
	PendingConfirm  *PendingConfirmItem // non-nil when awaiting name disambiguation
}

// PendingConfirmItem holds the state for a name disambiguation question.
type PendingConfirmItem struct {
	ActionIdx  int
	ItemIdx    int
	Name       string
	Candidates []ResolveResult
}

// Slot holds a resolved value for a field, tracking its provenance.
type Slot struct {
	Field    string // "quantity" | "unit" | "location" | "material_id"
	Value    any
	Required bool
	Source   string // "extracted" | "inferred" | "user_clarified"
}

// MissingRequirement describes a field that needs user input.
type MissingRequirement struct {
	ActionIdx int
	ItemIdx   int    // -1 means action-level, 0+ means item-level
	Field     string // "name" | "quantity" | "unit" | "location"
	Question  string // natural language question to ask the user
}

// getOrCreateSession returns existing session or creates a new one.
func (l *AgentLoop) getOrCreateSession(chatID string) *DialogSession {
	l.sessionMu.Lock()
	defer l.sessionMu.Unlock()

	session, exists := l.sessions[chatID]
	if !exists {
		session = &DialogSession{
			ChatID:         chatID,
			State:          StateIdle,
			LastActivityAt: time.Now(),
			CreatedAt:      time.Now(),
		}
		l.sessions[chatID] = session
		return session
	}

	// Timeout check — if expired, reset to Idle
	if session.State != StateIdle && time.Since(session.LastActivityAt) > sessionIdleTimeout {
		session.State = StateIdle
		session.PendingOp = nil
	}
	session.LastActivityAt = time.Now()
	return session
}

// cleanupSession removes all data for a chat session.
// Lock order: sessionMu -> historyMu -> opHistoryMu
func (l *AgentLoop) cleanupSession(chatID string) {
	l.sessionMu.Lock()
	delete(l.sessions, chatID)
	l.sessionMu.Unlock()

	l.historyMu.Lock()
	delete(l.histories, chatID)
	l.historyMu.Unlock()

	l.opHistoryMu.Lock()
	delete(l.opHistory, chatID)
	l.opHistoryMu.Unlock()
}

// mergeToPending merges newly extracted info into an existing pending operation.
// Uses four strategies in priority order: position match, type+name match, name match, fallback.
func (l *AgentLoop) mergeToPending(pending *PendingOperation, result *NluResult) {
	if pending == nil || result == nil {
		return
	}

	// If result is a clarify intent, use its content to fill slots
	for _, action := range result.Actions {
		for _, item := range action.Items {
			// Strategy 1: position match — match by action/field position
			if matched := l.mergeByPosition(pending, item, result.RawText); matched {
				continue
			}
			// Strategy 2: type+name match — match by action type + item name
			if matched := l.mergeByName(pending, item, action.Type); matched {
				continue
			}
		}
	}
}

// mergeByPosition tries to match a clarified item to the missing requirement that was just asked.
func (l *AgentLoop) mergeByPosition(pending *PendingOperation, item ExtractedItem, rawText string) bool {
	if len(pending.MissingReqs) == 0 {
		return false
	}
	lastReq := pending.MissingReqs[0]
	return l.fillSlot(pending, lastReq.ActionIdx, lastReq.ItemIdx, lastReq.Field, item)
}

// mergeByName tries to match a clarified item to actions by action type + name.
func (l *AgentLoop) mergeByName(pending *PendingOperation, item ExtractedItem, actionType string) bool {
	for i, action := range pending.Actions {
		if action.Type != actionType {
			continue
		}
		for j, existing := range action.Items {
			if existing.Name != "" && item.Name != "" &&
				strings.Contains(strings.ToLower(existing.Name), strings.ToLower(item.Name)) {
				// Fill matching fields from the new item
				if item.Quantity != nil {
					l.setSlot(pending, i, j, "quantity", *item.Quantity)
				}
				if item.Unit != "" {
					l.setSlot(pending, i, j, "unit", item.Unit)
				}
				if item.Location != "" {
					l.setSlot(pending, i, j, "location", item.Location)
				}
				return true
			}
		}
	}
	return false
}

// fillSlot fills a specific slot in the pending operation and removes the matching MissingRequirement.
func (l *AgentLoop) fillSlot(pending *PendingOperation, actionIdx, itemIdx int, field string, item ExtractedItem) bool {
	var value any
	switch field {
	case "quantity":
		if item.Quantity != nil {
			value = *item.Quantity
		}
	case "unit":
		value = item.Unit
	case "location":
		value = item.Location
	case "name":
		value = item.Name
	}
	if value == nil {
		return false
	}
	l.setSlot(pending, actionIdx, itemIdx, field, value)

	// Remove this requirement from missing list
	for i, req := range pending.MissingReqs {
		if req.ActionIdx == actionIdx && req.ItemIdx == itemIdx && req.Field == field {
			pending.MissingReqs = append(pending.MissingReqs[:i], pending.MissingReqs[i+1:]...)
			break
		}
	}
	return true
}

// setSlot stores a resolved value in the pending operation's slot map.
func (l *AgentLoop) setSlot(pending *PendingOperation, actionIdx, itemIdx int, field string, value any) {
	if pending.FilledSlots == nil {
		pending.FilledSlots = make(map[string][]Slot)
	}
	key := fmt.Sprintf("%d.%d.%s", actionIdx, itemIdx, field)
	pending.FilledSlots[key] = append(pending.FilledSlots[key], Slot{
		Field:  field,
		Value:  value,
		Source: "user_clarified",
	})
}

// newPendingOpID generates a unique ID for a pending operation.
func newPendingOpID() string {
	opIDCounter++
	return fmt.Sprintf("op_%d_%d", time.Now().UnixNano(), opIDCounter)
}
