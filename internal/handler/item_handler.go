package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

// ItemHandler serves item CRUD HTTP endpoints.
type ItemHandler struct {
	repo *repository.ItemRepository
}

// NewItemHandler creates an item handler.
func NewItemHandler(repo *repository.ItemRepository) *ItemHandler {
	return &ItemHandler{repo: repo}
}

// RegisterRoutes mounts item endpoints under /api/v1.
func (h *ItemHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.POST("/items", h.Create)
	api.GET("/items", h.List)
}

type createItemRequest struct {
	Name        string  `json:"name" binding:"required"`
	Category    string  `json:"category"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	Location    string  `json:"location"`
	ExpireAt    string  `json:"expire_at"`
	PurchasedAt string  `json:"purchased_at"`
	Notes       string  `json:"notes"`
}

// Create handles POST /items.
func (h *ItemHandler) Create(c *gin.Context) {
	var req createItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	item := &model.Item{
		TenantID: tenantIDFromRequest(c),
		Name:     strings.TrimSpace(req.Name),
		Category: strings.TrimSpace(req.Category),
		Quantity: req.Quantity,
		Unit:     strings.TrimSpace(req.Unit),
		Location: strings.TrimSpace(req.Location),
		Notes:    strings.TrimSpace(req.Notes),
	}

	if req.PurchasedAt != "" {
		purchasedAt, err := time.Parse(time.RFC3339, req.PurchasedAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid purchased_at, must be RFC3339",
			})
			return
		}
		item.PurchasedAt = purchasedAt
	}

	if req.ExpireAt != "" {
		expireAt, err := time.Parse(time.RFC3339, req.ExpireAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid expire_at, must be RFC3339",
			})
			return
		}
		item.ExpireAt = &expireAt
	}

	if err := h.repo.Create(item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "create item failed",
		})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// List handles GET /items.
func (h *ItemHandler) List(c *gin.Context) {
	items, err := h.repo.List(repository.ItemFilter{
		TenantID: tenantIDFromRequest(c),
		Location: strings.TrimSpace(c.Query("location")),
		Category: strings.TrimSpace(c.Query("category")),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "list items failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": len(items),
	})
}

func tenantIDFromRequest(c *gin.Context) string {
	tenantID := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))
	if tenantID == "" {
		return "default"
	}

	return tenantID
}
