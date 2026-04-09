package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

// CategoryHandler serves category CRUD HTTP endpoints.
type CategoryHandler struct {
	repo      *repository.CategoryRepository
	auditRepo *repository.AuditLogRepository
}

// NewCategoryHandler creates a category handler.
func NewCategoryHandler(repo *repository.CategoryRepository, auditRepo *repository.AuditLogRepository) *CategoryHandler {
	return &CategoryHandler{repo: repo, auditRepo: auditRepo}
}

// RegisterRoutes mounts category endpoints under /api/v1.
func (h *CategoryHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.POST("/categories", h.Create)
	api.GET("/categories", h.List)
	api.GET("/categories/:id", h.Get)
	api.PUT("/categories/:id", h.Update)
	api.DELETE("/categories/:id", h.Delete)
}

type createCategoryRequest struct {
	Name string `json:"name" binding:"required"`
}

type updateCategoryRequest struct {
	Name *string `json:"name"`
}

// Create handles POST /categories.
func (h *CategoryHandler) Create(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category := &model.Category{
		TenantID: tenantIDFromRequest(c),
		Name:     strings.TrimSpace(req.Name),
	}
	if category.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
		return
	}

	if err := h.repo.Create(category); err != nil {
		handleCategoryRepositoryError(c, err, "create category failed")
		return
	}

	actor := actorFromRequest(c)
	recordAuditLog(h.auditRepo, actor, "create", "category", category.ID, category.Name,
		marshalChanges(nil, category))

	c.JSON(http.StatusCreated, category)
}

// List handles GET /categories.
func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.repo.List(tenantIDFromRequest(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list categories failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
		"total":      len(categories),
	})
}

// Get handles GET /categories/:id.
func (h *CategoryHandler) Get(c *gin.Context) {
	category, err := h.repo.Get(strings.TrimSpace(c.Param("id")), tenantIDFromRequest(c))
	if err != nil {
		handleCategoryRepositoryError(c, err, "get category failed")
		return
	}

	c.JSON(http.StatusOK, category)
}

// Update handles PUT /categories/:id.
func (h *CategoryHandler) Update(c *gin.Context) {
	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.repo.Get(strings.TrimSpace(c.Param("id")), tenantIDFromRequest(c))
	if err != nil {
		handleCategoryRepositoryError(c, err, "get category failed")
		return
	}

	beforeName := category.Name

	if req.Name != nil {
		category.Name = strings.TrimSpace(*req.Name)
		if category.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
			return
		}
	}

	if err := h.repo.Update(category); err != nil {
		handleCategoryRepositoryError(c, err, "update category failed")
		return
	}

	actor := actorFromRequest(c)
	recordAuditLog(h.auditRepo, actor, "update", "category", category.ID, category.Name,
		marshalChanges(map[string]string{"name": beforeName}, map[string]string{"name": category.Name}))

	c.JSON(http.StatusOK, category)
}

// Delete handles DELETE /categories/:id.
func (h *CategoryHandler) Delete(c *gin.Context) {
	actor := actorFromRequest(c)
	id := strings.TrimSpace(c.Param("id"))
	tenantID := tenantIDFromRequest(c)

	category, err := h.repo.Get(id, tenantID)
	if err != nil {
		handleCategoryRepositoryError(c, err, "delete category failed")
		return
	}

	if err := h.repo.Delete(id, tenantID); err != nil {
		handleCategoryRepositoryError(c, err, "delete category failed")
		return
	}

	recordAuditLog(h.auditRepo, actor, "delete", "category", category.ID, category.Name,
		marshalChanges(category, nil))

	c.Status(http.StatusNoContent)
}

func handleCategoryRepositoryError(c *gin.Context, err error, fallbackMessage string) {
	switch {
	case repository.IsNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
	case err == repository.ErrCategoryInUse:
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case err == repository.ErrCategoryNameExists:
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallbackMessage})
	}
}
