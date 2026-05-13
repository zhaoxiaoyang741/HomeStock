package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

// CategoryHandler serves category CRUD HTTP endpoints.
type CategoryHandler struct{ svc *service.CategoryService }

// NewCategoryHandler creates a category handler.
func NewCategoryHandler(svc *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
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

func (h *CategoryHandler) Create(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpresp.Error(c, http.StatusBadRequest, "name cannot be empty")
		return
	}
	created, err := h.svc.Create(c.Request.Context(), svcActorFromRequest(c), name)
	if err != nil {
		handleCategoryRepositoryError(c, err, "create category failed")
		return
	}
	httpresp.Created(c, created)
}

func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.svc.List(c.Request.Context(), tenantIDFromRequest(c))
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "list categories failed")
		return
	}
	httpresp.List(c, categories, len(categories), 0, 0)
}

func (h *CategoryHandler) Get(c *gin.Context) {
	category, err := h.svc.Get(c.Request.Context(), strings.TrimSpace(c.Param("id")), tenantIDFromRequest(c))
	if err != nil {
		handleCategoryRepositoryError(c, err, "get category failed")
		return
	}
	httpresp.OK(c, category)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if n == "" {
			httpresp.Error(c, http.StatusBadRequest, "name cannot be empty")
			return
		}
	}
	updated, err := h.svc.Update(c.Request.Context(), svcActorFromRequest(c), strings.TrimSpace(c.Param("id")), tenantIDFromRequest(c), req.Name)
	if err != nil {
		handleCategoryRepositoryError(c, err, "update category failed")
		return
	}
	httpresp.OK(c, updated)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if err := h.svc.Delete(c.Request.Context(), svcActorFromRequest(c), id, tenantIDFromRequest(c)); err != nil {
		handleCategoryRepositoryError(c, err, "delete category failed")
		return
	}
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
