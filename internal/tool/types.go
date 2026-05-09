package tool

import (
	"context"

	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

// ToolHandler is the signature for a callable tool function.
type ToolHandler func(ctx context.Context, actor service.Actor, args map[string]any) (string, error)

// ToolResult is the outcome of executing a tool.
type ToolResult struct {
	ToolName  string
	Content   string // text result to feed back to the LLM
	Error     string // error message, empty if success
}
