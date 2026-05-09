package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/llm"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// modelListResponse is the response body for GET /models.
type modelListResponse struct {
	Models      []modelItemResponse `json:"models"`
	ActiveModel string              `json:"active_model"`
}

// modelItemResponse is a single model entry in the list response.
type modelItemResponse struct {
	ModelName string `json:"model_name"`
	Model     string `json:"model"`
	Provider  string `json:"provider"`
	Enabled   bool   `json:"enabled"`
	APIKey    string `json:"api_key"`
	APIBase   string `json:"api_base"`
}

// ModelHandler serves runtime model configuration endpoints.
type ModelHandler struct {
	configPath string
	activeName string
	swapFn     func(name string, cfg config.ModelConfig) error
}

// NewModelHandler creates a ModelHandler.
func NewModelHandler(configPath string) *ModelHandler {
	return &ModelHandler{
		configPath: configPath,
	}
}

// SetSwapFn registers the callback for hot-swapping the LLM provider.
func (h *ModelHandler) SetSwapFn(fn func(string, config.ModelConfig) error) {
	h.swapFn = fn
}

// SetActiveName records which model was swapped in at startup.
func (h *ModelHandler) SetActiveName(name string) {
	h.activeName = name
}

// RegisterRoutes mounts model config endpoints under the API group.
func (h *ModelHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/models", h.ListModels)
	api.PATCH("/models", h.UpdateModel)
	api.POST("/models/swap", h.SwapModel)
}

// ListModels returns all model configs with masked API keys and the active model name.
func (h *ModelHandler) ListModels(c *gin.Context) {
	cfg := config.Get()

	items := make([]modelItemResponse, 0, len(cfg.ModelList))
	for _, m := range cfg.ModelList {
		items = append(items, modelItemResponse{
			ModelName: m.ModelName,
			Model:     m.Model,
			Provider:  m.Provider,
			Enabled:   m.Enabled,
			APIKey:    maskAPIKey(m.APIKey),
			APIBase:   m.APIBase,
		})
	}

	httpresp.OK(c, modelListResponse{
		Models:      items,
		ActiveModel: h.activeName,
	})
}

// UpdateModel handles partial updates to a model config by model_name.
// Request: { model_name: string, enabled?: bool, model?: string, provider?: string, api_key?: string, api_base?: string }
// Empty/omitted string fields preserve existing values.
func (h *ModelHandler) UpdateModel(c *gin.Context) {
	var req struct {
		ModelName string  `json:"model_name"`
		Enabled   *bool   `json:"enabled,omitempty"`
		Model     *string `json:"model,omitempty"`
		Provider  *string `json:"provider,omitempty"`
		APIKey    *string `json:"api_key,omitempty"`
		APIBase   *string `json:"api_base,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.ModelName == "" {
		httpresp.Error(c, http.StatusBadRequest, "model_name is required")
		return
	}

	var found bool
	if err := config.Save(h.configPath, func(cfg *config.Config) {
		for i := range cfg.ModelList {
			if cfg.ModelList[i].ModelName == req.ModelName {
				m := &cfg.ModelList[i]
				if req.Enabled != nil {
					m.Enabled = *req.Enabled
				}
				if req.Model != nil && *req.Model != "" {
					m.Model = *req.Model
				}
				if req.Provider != nil && *req.Provider != "" {
					m.Provider = *req.Provider
				}
				if req.APIKey != nil && *req.APIKey != "" {
					m.APIKey = *req.APIKey
				}
				if req.APIBase != nil {
					m.APIBase = *req.APIBase
				}
				found = true
				return
			}
		}
	}); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "save config failed: "+err.Error())
		return
	}

	if !found {
		httpresp.Error(c, http.StatusNotFound, "model "+req.ModelName+" not found")
		return
	}

	httpresp.OK(c, gin.H{"message": "config updated"})
}

// SwapModel creates a new LLM provider from the named model config
// and hot-swaps it into the running AgentLoop.
// Request: { model_name: string }
func (h *ModelHandler) SwapModel(c *gin.Context) {
	var req struct {
		ModelName string `json:"model_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.ModelName == "" {
		httpresp.Error(c, http.StatusBadRequest, "model_name is required")
		return
	}

	cfg := config.Get()
	var found *config.ModelConfig
	for i := range cfg.ModelList {
		if cfg.ModelList[i].ModelName == req.ModelName {
			m := cfg.ModelList[i]
			found = &m
			break
		}
	}
	if found == nil {
		httpresp.Error(c, http.StatusNotFound, "model "+req.ModelName+" not found")
		return
	}

	// Validate the provider can be created before swapping.
	provider, err := llm.NewProvider(*found)
	if err != nil {
		httpresp.Error(c, http.StatusBadRequest, "invalid model config: "+err.Error())
		return
	}

	if h.swapFn == nil {
		httpresp.Error(c, http.StatusInternalServerError, "swap function not registered")
		return
	}

	if err := h.swapFn(req.ModelName, *found); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "swap failed: "+err.Error())
		return
	}

	// Keep the provider reference to prevent GC; the AgentLoop now owns it.
	_ = provider

	h.activeName = req.ModelName
	httpresp.OK(c, gin.H{"message": "switched to " + req.ModelName})
}

// maskAPIKey masks all but the first 4 and last 4 characters of an API key.
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	runes := []rune(key)
	if len(runes) <= 8 {
		return "********"
	}
	return string(runes[:4]) + "********" + string(runes[len(runes)-4:])
}
