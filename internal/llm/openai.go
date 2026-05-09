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

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// Default timeout for LLM API requests.
const defaultTimeout = 60 * time.Second

// openAIReq maps to the OpenAI chat/completions request body.
type openAIReq struct {
	Model       string          `json:"model"`
	Messages    []openAIMsg     `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  any             `json:"tool_choice,omitempty"`
}

// openAIMsg is a single message in the OpenAI chat format.
type openAIMsg struct {
	Role         string     `json:"role"`
	Content      string     `json:"content,omitempty"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID   string     `json:"tool_call_id,omitempty"`
	Name         string     `json:"name,omitempty"`
}

// openAIResp maps the relevant parts of the OpenAI chat/completions response.
type openAIResp struct {
	Choices []struct {
		Message       openAIMsg `json:"message"`
		FinishReason  string    `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// OpenAIProvider implements LLMProvider for OpenAI-compatible APIs.
type OpenAIProvider struct {
	model    string
	apiKey   string
	apiBase  string
	client   *http.Client
}

// NewOpenAIProvider creates an OpenAI-compatible provider from a model config.
func NewOpenAIProvider(cfg config.ModelConfig) *OpenAIProvider {
	base := strings.TrimRight(cfg.APIBase, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	// Remove /v1/chat/completions suffix if someone put the full path in config
	base = strings.TrimSuffix(base, "/chat/completions")
	base = strings.TrimSuffix(base, "/v1")
	base += "/v1"

	return &OpenAIProvider{
		model:   cfg.Model,
		apiKey:  cfg.APIKey,
		apiBase: base,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// GetDefaultModel returns the configured model identifier.
func (p *OpenAIProvider) GetDefaultModel() string {
	return p.model
}

// Chat sends a chat completion request and returns the parsed response.
func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string) (*LLMResponse, error) {
	if model == "" {
		model = p.model
	}

	reqBody := openAIReq{
		Model:    model,
		Messages: marshalMessages(messages),
	}

	// Attach tools only when present — some providers error on an empty tools array.
	if len(tools) > 0 {
		reqBody.Tools = tools
		reqBody.ToolChoice = "auto"
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiBase+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llm: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("llm: read response: %w", err)
	}

	var apiResp openAIResp
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		return nil, fmt.Errorf("llm: parse response: %w (body: %s)", err, truncate(string(raw), 200))
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("llm: API error [%s] %s (code=%s)", apiResp.Error.Type, apiResp.Error.Message, apiResp.Error.Code)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("llm: empty choices (body: %s)", truncate(string(raw), 200))
	}

	choice := apiResp.Choices[0]

	result := &LLMResponse{
		Content:      choice.Message.Content,
		ToolCalls:    choice.Message.ToolCalls,
		FinishReason: choice.FinishReason,
	}

	if apiResp.Usage != nil {
		result.Usage = UsageInfo{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		}
	}

	return result, nil
}

// marshalMessages converts internal Messages to OpenAI API message objects.
func marshalMessages(msgs []Message) []openAIMsg {
	out := make([]openAIMsg, 0, len(msgs))
	for _, m := range msgs {
		om := openAIMsg{
			Role:       m.Role,
			Content:    m.Content,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
			Name:       m.Name,
		}
		// OpenAI requires content to be present for user messages.
		// For assistant messages with tool_calls, content can be empty.
		if len(om.ToolCalls) > 0 && om.Content == "" {
			om.Content = ""
		}
		out = append(out, om)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
