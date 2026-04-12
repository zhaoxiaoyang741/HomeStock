package handler

import (
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"

    "github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type StockMovementHandler struct {
    repo repository.StockMovementRepo
}

func NewStockMovementHandler(repo repository.StockMovementRepo) *StockMovementHandler {
    return &StockMovementHandler{repo: repo}
}

func (h *StockMovementHandler) RegisterRoutes(api *gin.RouterGroup) { api.GET("/stock-movements", h.List) }

func (h *StockMovementHandler) List(c *gin.Context) {
    filter := repository.StockMovementFilter{
        TenantID:     tenantIDFromRequest(c),
        MaterialID:   strings.TrimSpace(c.Query("material_id")),
        LotID:        strings.TrimSpace(c.Query("lot_id")),
        MovementType: strings.TrimSpace(c.Query("movement_type")),
    }
    if startStr := strings.TrimSpace(c.Query("start_date")); startStr != "" {
        if t, err := time.Parse("2006-01-02", startStr); err == nil { filter.StartDate = t } else if t, err := time.Parse(time.RFC3339, startStr); err == nil { filter.StartDate = t }
    }
    if endStr := strings.TrimSpace(c.Query("end_date")); endStr != "" {
        if t, err := time.Parse("2006-01-02", endStr); err == nil { filter.EndDate = t.Add(24*time.Hour - time.Second) } else if t, err := time.Parse(time.RFC3339, endStr); err == nil { filter.EndDate = t }
    }
    movements, err := h.repo.List(filter)
    if err != nil { httpresp.Error(c, http.StatusInternalServerError, "list stock movements failed"); return }
    httpresp.List(c, movements, len(movements), 0, 0)
}

