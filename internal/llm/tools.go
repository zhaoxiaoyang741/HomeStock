package llm

// ToolDefinition is the OpenAI-compatible function calling tool definition.
type ToolDefinition struct {
	Type     string                 `json:"type"` // "function"
	Function ToolFunctionDefinition `json:"function"`
}

// ToolFunctionDefinition describes a callable function to the LLM.
type ToolFunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema
}
