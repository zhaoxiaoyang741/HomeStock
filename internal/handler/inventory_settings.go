package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

// InventoryConfig holds all runtime-editable inventory settings.
type InventoryConfig struct {
	DefaultLocation        string  `json:"default_location"`          // 入库默认存放位置
	DefaultExpiryDays      int     `json:"default_expiry_days"`       // 默认保质期天数（-1=永不过期）
	DefaultQuantity        float64 `json:"default_quantity"`          // 未指定数量时的默认值
	DueSoonDays            int     `json:"due_soon_days"`             // 即将过期提醒天数
	TrackPrice             bool    `json:"track_price"`               // 启用价格追踪
	TrackOpened            bool    `json:"track_opened"`              // 启用开封追踪
	AutoAddShoppingList    bool    `json:"auto_add_shopping_list"`    // 消耗后自动补货
	NluDefaultQuantity     float64 `json:"nlu_default_quantity"`      // NLU 默认数量
	NluAutoSelectThreshold float64 `json:"nlu_auto_select_threshold"` // NLU 名称自动选择阈值 (0-1)
	NluAutoSelectLead      float64 `json:"nlu_auto_select_lead"`      // NLU 名称自动选择领先差 (0-1)
}

const inventoryConfigKey = "inventory_config"

// DefaultInventoryConfig returns sensible defaults.
func DefaultInventoryConfig() InventoryConfig {
	return InventoryConfig{
		DefaultLocation:        "",
		DefaultExpiryDays:      0,
		DefaultQuantity:        1,
		DueSoonDays:            7,
		TrackPrice:             false,
		TrackOpened:            false,
		AutoAddShoppingList:    false,
		NluDefaultQuantity:     1,
		NluAutoSelectThreshold: 0.85,
		NluAutoSelectLead:      0.15,
	}
}

// InventorySettingHandler provides API endpoints for inventory configuration.
type InventorySettingHandler struct {
	repo repository.SystemSettingRepo
}

// NewInventorySettingHandler creates an InventorySettingHandler.
func NewInventorySettingHandler(repo repository.SystemSettingRepo) *InventorySettingHandler {
	return &InventorySettingHandler{repo: repo}
}

// RegisterRoutes registers the inventory config endpoints.
//   GET  /settings/inventory  – returns current inventory config
//   PUT  /settings/inventory  – updates inventory config
func (h *InventorySettingHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/settings/inventory", h.getConfig)
	rg.PUT("/settings/inventory", h.updateConfig)
}

func (h *InventorySettingHandler) getConfig(c *gin.Context) {
	cfg := h.loadConfig()
	httpresp.OK(c, cfg)
}

func (h *InventorySettingHandler) updateConfig(c *gin.Context) {
	var req InventoryConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Validate
	if req.NluAutoSelectThreshold < 0 || req.NluAutoSelectThreshold > 1 {
		httpresp.Error(c, http.StatusBadRequest, "nlu_auto_select_threshold must be between 0 and 1")
		return
	}
	if req.NluAutoSelectLead < 0 || req.NluAutoSelectLead > 1 {
		httpresp.Error(c, http.StatusBadRequest, "nlu_auto_select_lead must be between 0 and 1")
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "marshal config failed")
		return
	}

	if err := h.repo.Upsert(&model.SystemSetting{
		Key:     inventoryConfigKey,
		Payload: string(payload),
	}); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "save config failed")
		return
	}

	httpresp.OK(c, req)
}

// LoadConfig reads the current config from the database and returns it.
// If no stored config exists, returns defaults.
func (h *InventorySettingHandler) loadConfig() InventoryConfig {
	cfg := DefaultInventoryConfig()
	setting, err := h.repo.GetByKey(inventoryConfigKey)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal([]byte(setting.Payload), &cfg)
	return cfg
}

// GetInventoryConfig is a convenience function to load inventory config from a repo.
func GetInventoryConfig(repo repository.SystemSettingRepo) InventoryConfig {
	return (&InventorySettingHandler{repo: repo}).loadConfig()
}
