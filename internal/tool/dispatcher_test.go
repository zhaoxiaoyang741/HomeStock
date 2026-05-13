package tool

import (
	"context"
	"testing"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/llm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

func TestRegisterAndExecute(t *testing.T) {
	d := NewDispatcher()
	d.Register("hello", func(_ context.Context, _ service.Actor, args map[string]any) (string, error) {
		name, _ := args["name"].(string)
		return "hello " + name, nil
	})

	result, err := d.Execute(context.Background(), service.Actor{}, "hello", map[string]any{"name": "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", result)
	}
}

func TestExecuteUnknownTool(t *testing.T) {
	d := NewDispatcher()
	_, err := d.Execute(context.Background(), service.Actor{}, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()

	d := NewDispatcher()
	d.Register("foo", func(_ context.Context, _ service.Actor, _ map[string]any) (string, error) {
		return "ok", nil
	})
	d.Register("foo", func(_ context.Context, _ service.Actor, _ map[string]any) (string, error) {
		return "ok", nil
	})
}

func TestToolHandlerReturnsError(t *testing.T) {
	d := NewDispatcher()
	d.Register("fail", func(_ context.Context, _ service.Actor, _ map[string]any) (string, error) {
		return "", errMock
	})

	_, err := d.Execute(context.Background(), service.Actor{}, "fail", nil)
	if err == nil {
		t.Fatal("expected error from handler")
	}
}

func TestConcurrencySafety(t *testing.T) {
	d := NewDispatcher()
	d.Register("ping", func(_ context.Context, _ service.Actor, _ map[string]any) (string, error) {
		return "pong", nil
	})

	done := make(chan struct{}, 20)
	for range 20 {
		go func() {
			_, _ = d.Execute(context.Background(), service.Actor{}, "ping", nil)
			done <- struct{}{}
		}()
	}
	for range 20 {
		<-done
	}
}

func TestSetDefinitions(t *testing.T) {
	d := NewDispatcher()
	defs := []llm.ToolDefinition{
		{Type: "function", Function: llm.ToolFunctionDefinition{Name: "tool_a"}},
		{Type: "function", Function: llm.ToolFunctionDefinition{Name: "tool_b"}},
	}
	d.SetDefinitions(defs)

	got := d.Definitions()
	if len(got) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(got))
	}
	if got[0].Function.Name != "tool_a" {
		t.Fatalf("expected first definition name 'tool_a', got %q", got[0].Function.Name)
	}
}

func TestEmptyDefinitions(t *testing.T) {
	d := NewDispatcher()
	defs := d.Definitions()
	if len(defs) != 0 {
		t.Fatalf("expected empty definitions, got %d", len(defs))
	}
}

func TestReceiverArgument(t *testing.T) {
	d := NewDispatcher()
	d.Register("echo", func(_ context.Context, actor service.Actor, _ map[string]any) (string, error) {
		return actor.TenantID, nil
	})

	actor := service.Actor{TenantID: "tenant-42"}
	result, err := d.Execute(context.Background(), actor, "echo", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "tenant-42" {
		t.Fatalf("expected 'tenant-42', got %q", result)
	}
}

var errMock = &mockError{}

type mockError struct{}

func (e *mockError) Error() string { return "mock error" }
