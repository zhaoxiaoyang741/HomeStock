package llm

// Message represents a single message in the LLM conversation.
type Message struct {
	Role             string     // "system", "user", "assistant", "tool"
	Content          string     // text content
	ReasoningContent string     // DeepSeek reasoning model — must be echoed back
	ToolCalls        []ToolCall // assistant message tool calls
	ToolCallID       string     // tool message — associates with a tool call
	Name             string     // tool message — the function name that was called
}

// ToolCall represents a function call requested by the assistant.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall is the function name and arguments from a tool call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// LLMResponse is the parsed response from an LLM provider.
type LLMResponse struct {
	Content          string    // text content (empty if tool_calls)
	ReasoningContent string    // DeepSeek reasoning model — must be echoed back
	ToolCalls        []ToolCall
	FinishReason     string // "stop", "tool_calls", "length"
	Usage            UsageInfo
}

// UsageInfo contains token usage statistics.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
