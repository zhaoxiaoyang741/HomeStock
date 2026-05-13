package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

const defaultOllamaTimeout = 120 * time.Second

// ollamaReq maps to the Ollama /api/chat request body.
type ollamaReq struct {
	Model    string          `json:"model"`
	Messages []ollamaMsg     `json:"messages"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
}

// ollamaMsg is a single message in Ollama's chat format.
type ollamaMsg struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

// ollamaToolCall mirrors Ollama's tool call format (arguments is a map, not a string).
type ollamaToolCall struct {
	ID       string              `json:"id,omitempty"`
	Type     string              `json:"type"`
	Function ollamaFunctionCall  `json:"function"`
}

type ollamaFunctionCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ollamaResp maps the Ollama /api/chat response body.
type ollamaResp struct {
	Model      string       `json:"model"`
	CreatedAt  string       `json:"created_at"`
	Message    ollamaMsg    `json:"message"`
	DoneReason string       `json:"done_reason"`
	Done       bool         `json:"done"`
	Error      string       `json:"error,omitempty"`

	// Usage-like stats
	PromptEvalCount int `json:"prompt_eval_count"`
	EvalCount       int `json:"eval_count"`
}

// OllamaProvider implements LLMProvider for Ollama's local API.
type OllamaProvider struct {
	model   string
	apiBase string
	client  *http.Client
}

// NewOllamaProvider creates an Ollama provider from a model config.
func NewOllamaProvider(cfg config.ModelConfig) *OllamaProvider {
	base := strings.TrimRight(cfg.APIBase, "/")
	if base == "" {
		base = "http://localhost:11434"
	}

	return &OllamaProvider{
		model:   cfg.Model,
		apiBase: base,
		client: &http.Client{
			Timeout: defaultOllamaTimeout,
		},
	}
}

// GetDefaultModel returns the configured model identifier.
func (p *OllamaProvider) GetDefaultModel() string {
	return p.model
}

// Chat sends a chat completion request to Ollama and returns the parsed response.
func (p *OllamaProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string) (*LLMResponse, error) {
	if model == "" {
		model = p.model
	}

	reqBody := ollamaReq{
		Model:    model,
		Messages: marshalOllamaMessages(messages),
		Stream:   false,
	}

	if len(tools) > 0 {
		reqBody.Tools = tools
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiBase+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama: read response: %w", err)
	}

	var apiResp ollamaResp
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		return nil, fmt.Errorf("ollama: parse response: %w (body: %s)", err, truncate(string(raw), 200))
	}

	if apiResp.Error != "" {
		return nil, fmt.Errorf("ollama: API error: %s", apiResp.Error)
	}

	result := &LLMResponse{
		Content: apiResp.Message.Content,
	}

	// Convert Ollama tool_calls to the standard ToolCall format
	if len(apiResp.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]ToolCall, 0, len(apiResp.Message.ToolCalls))
		for _, otc := range apiResp.Message.ToolCalls {
			// Ollama may omit tool call IDs — generate one if missing
			id := otc.ID
			if id == "" {
				id = "ollama_" + uuid.NewString()[:8]
			}
			// Marshal arguments map to JSON string for compatibility
			argsJSON := ""
			if len(otc.Function.Arguments) > 0 {
				argsBytes, err := json.Marshal(otc.Function.Arguments)
				if err == nil {
					argsJSON = string(argsBytes)
				}
			}
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:   id,
				Type: otc.Type,
				Function: FunctionCall{
					Name:      otc.Function.Name,
					Arguments: argsJSON,
				},
			})
		}
	}

	switch apiResp.DoneReason {
	case "stop":
		result.FinishReason = "stop"
	case "tool_calls":
		result.FinishReason = "tool_calls"
	default:
		result.FinishReason = apiResp.DoneReason
	}

	if apiResp.PromptEvalCount > 0 || apiResp.EvalCount > 0 {
		result.Usage = UsageInfo{
			PromptTokens:     apiResp.PromptEvalCount,
			CompletionTokens: apiResp.EvalCount,
			TotalTokens:      apiResp.PromptEvalCount + apiResp.EvalCount,
		}
	}

	return result, nil
}

// marshalOllamaMessages converts internal Messages to Ollama's message format.
func marshalOllamaMessages(msgs []Message) []ollamaMsg {
	out := make([]ollamaMsg, 0, len(msgs))
	for _, m := range msgs {
		om := ollamaMsg{
			Role:    m.Role,
			Content: m.Content,
		}

		// Convert standard ToolCalls to Ollama format
		if len(m.ToolCalls) > 0 {
			om.ToolCalls = make([]ollamaToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				otc := ollamaToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: ollamaFunctionCall{
						Name: tc.Function.Name,
					},
				}
				// Unmarshal arguments JSON string to map
				if tc.Function.Arguments != "" {
					var argsMap map[string]any
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &argsMap); err == nil {
						otc.Function.Arguments = argsMap
					}
				}
				om.ToolCalls = append(om.ToolCalls, otc)
			}
		}

		out = append(out, om)
	}
	return out
}
