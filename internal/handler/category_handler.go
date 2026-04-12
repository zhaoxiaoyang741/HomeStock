package handler

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"

    httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
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

type createCategoryRequest struct { Name string `json:"name" binding:"required"` }

type updateCategoryRequest struct { Name *string `json:"name"` }

func (h *CategoryHandler) Create(c *gin.Context) {
    var req createCategoryRequest
    if err := c.ShouldBindJSON(&req); err != nil { httpresp.Error(c, http.StatusBadRequest, err.Error()); return }
    category := &model.Category{ TenantID: tenantIDFromRequest(c), Name: strings.TrimSpace(req.Name) }
    if category.Name == "" { httpresp.Error(c, http.StatusBadRequest, "name cannot be empty"); return }
    if err := h.repo.Create(category); err != nil { handleCategoryRepositoryError(c, err, "create category failed"); return }
    actor := actorFromRequest(c)
    recordAuditLog(h.auditRepo, actor, "create", "category", category.ID, category.Name, marshalChanges(nil, category))
    httpresp.Created(c, category)
}

func (h *CategoryHandler) List(c *gin.Context) {
    categories, err := h.repo.List(tenantIDFromRequest(c))
    if err != nil { httpresp.Error(c, http.StatusInternalServerError, "list categories failed"); return }
    httpresp.List(c, categories, len(categories), 0, 0)
}

func (h *CategoryHandler) Get(c *gin.Context) {
    category, err := h.repo.Get(strings.TrimSpace(c.Param("id")), tenantIDFromRequest(c))
    if err != nil { handleCategoryRepositoryError(c, err, "get category failed"); return }
    httpresp.OK(c, category)
}

func (h *CategoryHandler) Update(c *gin.Context) {
    var req updateCategoryRequest
    if err := c.ShouldBindJSON(&req); err != nil { httpresp.Error(c, http.StatusBadRequest, err.Error()); return }
    category, err := h.repo.Get(strings.TrimSpace(c.Param("id")), tenantIDFromRequest(c))
    if err != nil { handleCategoryRepositoryError(c, err, "get category failed"); return }
    beforeName := category.Name
    if req.Name != nil { category.Name = strings.TrimSpace(*req.Name); if category.Name == "" { httpresp.Error(c, http.StatusBadRequest, "name cannot be empty"); return } }
    if err := h.repo.Update(category); err != nil { handleCategoryRepositoryError(c, err, "update category failed"); return }
    actor := actorFromRequest(c)
    recordAuditLog(h.auditRepo, actor, "update", "category", category.ID, category.Name, marshalChanges(map[string]string{"name": beforeName}, map[string]string{"name": category.Name}))
    httpresp.OK(c, category)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
    actor := actorFromRequest(c)
    id := strings.TrimSpace(c.Param("id"))
    tenantID := tenantIDFromRequest(c)
    category, err := h.repo.Get(id, tenantID)
    if err != nil { handleCategoryRepositoryError(c, err, "delete category failed"); return }
    if err := h.repo.Delete(id, tenantID); err != nil { handleCategoryRepositoryError(c, err, "delete category failed"); return }
    recordAuditLog(h.auditRepo, actor, "delete", "category", category.ID, category.Name, marshalChanges(category, nil))
    httpresp.NoContent(c)
}

func handleCategoryRepositoryError(c *gin.Context, err error, fallbackMessage string) {
    switch {
    case repository.IsNotFound(err):
        httpresp.Error(c, http.StatusNotFound, "category not found")
    case err == repository.ErrCategoryInUse:
        httpresp.Error(c, http.StatusConflict, err.Error())
    case err == repository.ErrCategoryNameExists:
        httpresp.Error(c, http.StatusConflict, err.Error())
    default:
        httpresp.Error(c, http.StatusInternalServerError, fallbackMessage)
    }
}
