package tool

import (
	"context"
	"fmt"
	"sync"

	"github.com/zhaoxiaoyang741/HomeStock/internal/llm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

// Dispatcher registers and executes tool handlers by name.
type Dispatcher struct {
	mu          sync.RWMutex
	handlers    map[string]ToolHandler
	definitions []llm.ToolDefinition
}

// NewDispatcher creates an empty Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string]ToolHandler),
	}
}

// Register adds a tool handler under the given name.
// Panics if the name is already registered.
func (d *Dispatcher) Register(name string, handler ToolHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.handlers[name]; exists {
		panic(fmt.Sprintf("tool %q already registered", name))
	}
	d.handlers[name] = handler
}

// Execute runs the named tool with the given arguments and returns the result.
// Returns an error if the tool is not found or the handler fails.
func (d *Dispatcher) Execute(ctx context.Context, actor service.Actor, toolName string, args map[string]any) (string, error) {
	d.mu.RLock()
	handler, ok := d.handlers[toolName]
	d.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("tool %q not found", toolName)
	}

	return handler(ctx, actor, args)
}

// Definitions returns the LLM tool definitions for all registered tools.
// Must be populated by each tool package in its init() or via registration.
func (d *Dispatcher) Definitions() []llm.ToolDefinition {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Subclasses/tools register their own definitions separately.
	// This returns what's been registered. For now, it's a placeholder
	// that will be populated when concrete tools are added.
	return d.definitions
}

// SetDefinitions sets the tool definitions for LLM consumption.
func (d *Dispatcher) SetDefinitions(defs []llm.ToolDefinition) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.definitions = defs
}

