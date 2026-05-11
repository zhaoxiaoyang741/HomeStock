package llm

import (
	"fmt"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// NewProvider creates an LLMProvider based on the model configuration.
// Supported provider types: "openai" (default), "ollama", and "deepseek".
func NewProvider(cfg config.ModelConfig) (LLMProvider, error) {
	switch cfg.Provider {
	case "", "openai":
		return NewOpenAIProvider(cfg), nil
	case "ollama":
		return NewOllamaProvider(cfg), nil
	case "deepseek":
		cfg2 := cfg
		if cfg2.APIBase == "" {
			cfg2.APIBase = "https://api.deepseek.com"
		}
		return NewOpenAIProvider(cfg2), nil
	default:
		return nil, fmt.Errorf("llm: unsupported provider %q", cfg.Provider)
	}
}
