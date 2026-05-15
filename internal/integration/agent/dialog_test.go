package agent

import (
	"testing"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/llm"
)

func TestGetOrCreateSession_NewAndExisting(t *testing.T) {
	// Setup minimal AgentLoop
	l := &AgentLoop{
		sessions: make(map[string]*DialogSession),
	}

	s1 := l.getOrCreateSession("chat_1")
	if s1 == nil {
		t.Fatal("expected non-nil session")
	}
	if s1.State != StateIdle {
		t.Fatalf("expected StateIdle, got %v", s1.State)
	}

	s2 := l.getOrCreateSession("chat_1")
	if s2 != s1 {
		t.Fatal("expected same session for same chatID")
	}

	s3 := l.getOrCreateSession("chat_2")
	if s3 == s2 {
		t.Fatal("expected different session for different chatID")
	}
}

func TestSessionTimeout_StateReset(t *testing.T) {
	l := &AgentLoop{
		sessions: make(map[string]*DialogSession),
	}

	session := l.getOrCreateSession("chat_1")
	session.State = StateClarifying
	session.PendingOp = &PendingOperation{ID: "op_1"}
	session.LastActivityAt = time.Now().Add(-10 * time.Minute) // expired

	// getOrCreateSession should reset expired session
	s2 := l.getOrCreateSession("chat_1")
	if s2.State != StateIdle {
		t.Fatalf("expected StateIdle after timeout, got %v", s2.State)
	}
	if s2.PendingOp != nil {
		t.Fatal("expected PendingOp to be cleared after timeout")
	}
}

func TestSessionTimeout_ActiveNotReset(t *testing.T) {
	l := &AgentLoop{
		sessions: make(map[string]*DialogSession),
	}

	session := l.getOrCreateSession("chat_1")
	session.State = StateClarifying
	session.LastActivityAt = time.Now() // still active

	s2 := l.getOrCreateSession("chat_1")
	if s2.State != StateClarifying {
		t.Fatalf("expected StateClarifying (not reset), got %v", s2.State)
	}
}

func TestCleanupSession_RemovesAll(t *testing.T) {
	l := &AgentLoop{
		sessions:  make(map[string]*DialogSession),
		histories: make(map[string][]llm.Message),
		opHistory: make(map[string][]OperationRecord),
	}

	// Create session data
	l.getOrCreateSession("chat_1")
	l.histories["chat_1"] = []llm.Message{} // dummy
	l.opHistory["chat_1"] = []OperationRecord{}

	l.cleanupSession("chat_1")

	if _, exists := l.sessions["chat_1"]; exists {
		t.Fatal("expected session to be removed")
	}
	if _, exists := l.histories["chat_1"]; exists {
		t.Fatal("expected history to be removed")
	}
	if _, exists := l.opHistory["chat_1"]; exists {
		t.Fatal("expected opHistory to be removed")
	}
}

func TestMergeToPending_NilResult(t *testing.T) {
	// Should not panic
	l := &AgentLoop{}
	pending := &PendingOperation{Actions: []ExtractedAction{}}
	l.mergeToPending(pending, nil) // nil result
	l.mergeToPending(nil, &NluResult{}) // nil pending
}

func TestMergeToPending_FillsQuantityByName(t *testing.T) {
	l := &AgentLoop{}
	qty := 3.0
	pending := &PendingOperation{
		Actions: []ExtractedAction{
			{
				Type: "inbound",
				Items: []ExtractedItem{
					{Name: "牛奶", Quantity: nil, Unit: "瓶"},
				},
			},
		},
		MissingReqs: []MissingRequirement{
			{ActionIdx: 0, ItemIdx: 0, Field: "quantity", Question: "几瓶牛奶？"},
		},
	}

	result := &NluResult{
		Intent:  "clarify",
		RawText: "3瓶",
		Actions: []ExtractedAction{
			{
				Type: "clarify",
				Items: []ExtractedItem{
					{Name: "牛奶", Quantity: &qty, Unit: "瓶"},
				},
			},
		},
	}

	l.mergeToPending(pending, result)

	// Verify slot was filled
	key := "0.0.quantity"
	slots, ok := pending.FilledSlots[key]
	if !ok {
		t.Fatal("expected slot to be filled")
	}
	if slots[0].Value != 3.0 {
		t.Fatalf("expected 3.0, got %v", slots[0].Value)
	}
}

func TestNewPendingOpID_Unique(t *testing.T) {
	id1 := newPendingOpID()
	id2 := newPendingOpID()
	if id1 == id2 {
		t.Fatal("expected unique IDs")
	}
}
