package llm

import (
	"fmt"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// NewProvider creates an LLMProvider based on the model configuration.
// Supported provider types: "openai" (default) and "ollama".
func NewProvider(cfg config.ModelConfig) (LLMProvider, error) {
	switch cfg.Provider {
	case "", "openai":
		return NewOpenAIProvider(cfg), nil
	case "ollama":
		return NewOllamaProvider(cfg), nil
	default:
		return nil, fmt.Errorf("llm: unsupported provider %q", cfg.Provider)
	}
}
