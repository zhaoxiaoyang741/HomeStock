package llm

import "context"

// LLMProvider is the abstraction for a large language model backend.
type LLMProvider interface {
	// Chat sends a conversation to the LLM and returns the response.
	// tools may be nil if no tool definitions are needed.
	// model selects which model to use; empty means use the default.
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string) (*LLMResponse, error)

	// GetDefaultModel returns the default model identifier for this provider.
	GetDefaultModel() string
}
