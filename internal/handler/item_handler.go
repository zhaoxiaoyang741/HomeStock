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
	repo      *repository.ItemRepository
	auditRepo *repository.AuditLogRepository
}

// NewItemHandler creates an item handler.
func NewItemHandler(repo *repository.ItemRepository, auditRepo *repository.AuditLogRepository) *ItemHandler {
	return &ItemHandler{repo: repo, auditRepo: auditRepo}
}

// RegisterRoutes mounts item endpoints under /api/v1.
func (h *ItemHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.POST("/items", h.Create)
	api.GET("/items", h.List)
	api.GET("/items/similar", h.FindSimilar) // must be before /:id
	api.GET("/items/:id", h.Get)
	api.PUT("/items/:id", h.Update)
	api.DELETE("/items/:id", h.Delete)
}

type createItemRequest struct {
	Name        string  `json:"name" binding:"required"`
	CategoryID  string  `json:"category_id"`
	Spec        string  `json:"spec"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	Location    string  `json:"location"`
	ExpireAt    string  `json:"expire_at"`
	PurchasedAt string  `json:"purchased_at"`
	Notes       string  `json:"notes"`
}

type updateItemRequest struct {
	Name        *string  `json:"name"`
	CategoryID  *string  `json:"category_id"`
	Spec        *string  `json:"spec"`
	Quantity    *float64 `json:"quantity"`
	Unit        *string  `json:"unit"`
	Location    *string  `json:"location"`
	ExpireAt    *string  `json:"expire_at"`
	PurchasedAt *string  `json:"purchased_at"`
	Notes       *string  `json:"notes"`
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
		TenantID:   tenantIDFromRequest(c),
		Name:       strings.TrimSpace(req.Name),
		CategoryID: strings.TrimSpace(req.CategoryID),
		Spec:       strings.TrimSpace(req.Spec),
		Quantity:   req.Quantity,
		Unit:       strings.TrimSpace(req.Unit),
		Location:   strings.TrimSpace(req.Location),
		Notes:      strings.TrimSpace(req.Notes),
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
		handleItemRepositoryError(c, err, "create item failed")
		return
	}

	createdItem, err := h.repo.Get(item.ID, item.TenantID)
	if err != nil {
		handleItemRepositoryError(c, err, "get item failed")
		return
	}

	actor := actorFromRequest(c)
	recordAuditLog(h.auditRepo, actor, "create", "item", createdItem.ID, createdItem.Name,
		marshalChanges(nil, createdItem))

	c.JSON(http.StatusCreated, createdItem)
}

// List handles GET /items.
func (h *ItemHandler) List(c *gin.Context) {
	items, err := h.repo.List(repository.ItemFilter{
		TenantID:   tenantIDFromRequest(c),
		Location:   strings.TrimSpace(c.Query("location")),
		CategoryID: strings.TrimSpace(c.Query("category_id")),
	})
	if err != nil {
		handleItemRepositoryError(c, err, "list items failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": len(items),
	})
}

// Get handles GET /items/:id.
func (h *ItemHandler) Get(c *gin.Context) {
	item, err := h.repo.Get(strings.TrimSpace(c.Param("id")), tenantIDFromRequest(c))
	if err != nil {
		handleItemRepositoryError(c, err, "get item failed")
		return
	}

	c.JSON(http.StatusOK, item)
}

// Update handles PUT /items/:id.
func (h *ItemHandler) Update(c *gin.Context) {
	var req updateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	item, err := h.repo.Get(strings.TrimSpace(c.Param("id")), tenantIDFromRequest(c))
	if err != nil {
		handleItemRepositoryError(c, err, "get item failed")
		return
	}

	// Snapshot before mutation for audit log.
	beforeItem := *item
	if item.ExpireAt != nil {
		t := *item.ExpireAt
		beforeItem.ExpireAt = &t
	}

	if req.Name != nil {
		item.Name = strings.TrimSpace(*req.Name)
		if item.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "name cannot be empty",
			})
			return
		}
	}
	if req.CategoryID != nil {
		item.CategoryID = strings.TrimSpace(*req.CategoryID)
	}
	if req.Quantity != nil {
		item.Quantity = *req.Quantity
	}
	if req.Unit != nil {
		item.Unit = strings.TrimSpace(*req.Unit)
	}
	if req.Spec != nil {
		item.Spec = strings.TrimSpace(*req.Spec)
	}
	if req.Location != nil {
		item.Location = strings.TrimSpace(*req.Location)
	}
	if req.Notes != nil {
		item.Notes = strings.TrimSpace(*req.Notes)
	}
	if req.PurchasedAt != nil {
		if strings.TrimSpace(*req.PurchasedAt) == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid purchased_at, must be RFC3339",
			})
			return
		}

		purchasedAt, err := time.Parse(time.RFC3339, *req.PurchasedAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid purchased_at, must be RFC3339",
			})
			return
		}
		item.PurchasedAt = purchasedAt
	}
	if req.ExpireAt != nil {
		if strings.TrimSpace(*req.ExpireAt) == "" {
			item.ExpireAt = nil
		} else {
			expireAt, err := time.Parse(time.RFC3339, *req.ExpireAt)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid expire_at, must be RFC3339",
				})
				return
			}
			item.ExpireAt = &expireAt
		}
	}

	if err := h.repo.Update(item); err != nil {
		handleItemRepositoryError(c, err, "update item failed")
		return
	}

	item, err = h.repo.Get(item.ID, item.TenantID)
	if err != nil {
		handleItemRepositoryError(c, err, "get item failed")
		return
	}

	actor := actorFromRequest(c)
	recordAuditLog(h.auditRepo, actor, "update", "item", item.ID, item.Name,
		marshalChanges(beforeItem, item))

	c.JSON(http.StatusOK, item)
}

// Delete handles DELETE /items/:id.
func (h *ItemHandler) Delete(c *gin.Context) {
	actor := actorFromRequest(c)
	id := strings.TrimSpace(c.Param("id"))
	tenantID := tenantIDFromRequest(c)

	item, err := h.repo.Get(id, tenantID)
	if err != nil {
		handleItemRepositoryError(c, err, "delete item failed")
		return
	}

	if err := h.repo.Delete(id, tenantID); err != nil {
		handleItemRepositoryError(c, err, "delete item failed")
		return
	}

	recordAuditLog(h.auditRepo, actor, "delete", "item", item.ID, item.Name,
		marshalChanges(item, nil))

	c.Status(http.StatusNoContent)
}

// FindSimilar handles GET /items/similar?name=X&spec=Y.
func (h *ItemHandler) FindSimilar(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	items, err := h.repo.FindSimilar(repository.ItemFilter{
		TenantID: tenantIDFromRequest(c),
		Name:     name,
		Spec:     strings.TrimSpace(c.Query("spec")),
	})
	if err != nil {
		handleItemRepositoryError(c, err, "find similar items failed")
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

func handleItemRepositoryError(c *gin.Context, err error, fallbackMessage string) {
	switch {
	case repository.IsNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
	case err == repository.ErrInvalidCategoryID:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallbackMessage})
	}
}
