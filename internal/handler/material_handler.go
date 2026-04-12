package handler

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
    "gorm.io/gorm"

    "github.com/zhaoxiaoyang741/HomeStock/internal/model"
    "github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type MaterialHandler struct {
    db        *gorm.DB
    repo      *repository.MaterialRepository
    lotRepo   *repository.StockLotRepository
    moveRepo  *repository.StockMovementRepository
    auditRepo *repository.AuditLogRepository
}

func NewMaterialHandler(
    db *gorm.DB,
    repo *repository.MaterialRepository,
    lotRepo *repository.StockLotRepository,
    moveRepo *repository.StockMovementRepository,
    auditRepo *repository.AuditLogRepository,
) *MaterialHandler {
    return &MaterialHandler{
        db:        db,
        repo:      repo,
        lotRepo:   lotRepo,
        moveRepo:  moveRepo,
        auditRepo: auditRepo,
    }
}

func (h *MaterialHandler) RegisterRoutes(api *gin.RouterGroup) {
    api.GET("/materials", h.List)
    api.POST("/materials", h.Create)
    api.GET("/materials/:id", h.Get)
    api.POST("/materials/:id/consume", h.Consume)
}

type createMaterialRequest struct {
    Name        string `json:"name" binding:"required"`
    Spec        string `json:"spec"`
    CategoryID  string `json:"category_id"`
    DefaultUnit string `json:"default_unit"`
}

type consumeMaterialRequest struct {
    Quantity float64 `json:"quantity"`
    Reason   string  `json:"reason"`
}

type consumeLotResult struct {
    LotID             string      `json:"lot_id"`
    ConsumedQuantity  float64     `json:"consumed_quantity"`
    RemainingQuantity float64     `json:"remaining_quantity"`
    Location          string      `json:"location"`
    ExpireAt          interface{} `json:"expire_at"`
}

func (h *MaterialHandler) Create(c *gin.Context) {
    var req createMaterialRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httpresp.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    material := &model.Material{
        TenantID:    tenantIDFromRequest(c),
        Name:        strings.TrimSpace(req.Name),
        Spec:        strings.TrimSpace(req.Spec),
        CategoryID:  strings.TrimSpace(req.CategoryID),
        DefaultUnit: strings.TrimSpace(req.DefaultUnit),
    }
    if material.Name == "" {
        httpresp.Error(c, http.StatusBadRequest, "name cannot be empty")
        return
    }
    if err := h.repo.Create(material); err != nil {
        handleMaterialRepositoryError(c, err, "create material failed")
        return
    }

    created, err := h.repo.Get(material.ID, material.TenantID)
    if err != nil {
        handleMaterialRepositoryError(c, err, "get material failed")
        return
    }

    recordAuditLog(h.auditRepo, actorFromRequest(c), "create", "material", created.ID, created.Name, marshalChanges(nil, created))
    httpresp.Created(c, created)
}

func (h *MaterialHandler) List(c *gin.Context) {
    summaries, err := h.repo.List(repository.MaterialFilter{
        TenantID:   tenantIDFromRequest(c),
        CategoryID: strings.TrimSpace(c.Query("category_id")),
        Keyword:    strings.TrimSpace(c.Query("keyword")),
    })
    if err != nil {
        handleMaterialRepositoryError(c, err, "list materials failed")
        return
    }
    httpresp.List(c, summaries, len(summaries), 0, 0)
}

func (h *MaterialHandler) Get(c *gin.Context) {
    detail, err := h.repo.GetDetail(strings.TrimSpace(c.Param("id")), tenantIDFromRequest(c))
    if err != nil {
        handleMaterialRepositoryError(c, err, "get material failed")
        return
    }
    httpresp.OK(c, detail)
}

func (h *MaterialHandler) Consume(c *gin.Context) {
    var req consumeMaterialRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httpresp.Error(c, http.StatusBadRequest, err.Error())
        return
    }
    if req.Quantity <= 0 {
        httpresp.Error(c, http.StatusBadRequest, "quantity must be greater than 0")
        return
    }

    tenantID := tenantIDFromRequest(c)
    materialID := strings.TrimSpace(c.Param("id"))
    actor := actorFromRequest(c)
    var (
        material *model.Material
        results  []consumeLotResult
    )

    err := h.db.Transaction(func(tx *gorm.DB) error {
        materialRepo := repository.NewMaterialRepository(tx)
        lotRepo := repository.NewStockLotRepository(tx)
        moveRepo := repository.NewStockMovementRepository(tx)
        auditRepo := repository.NewAuditLogRepository(tx)

        gotMaterial, err := materialRepo.Get(materialID, tenantID)
        if err != nil {
            return err
        }
        material = gotMaterial

        lots, err := lotRepo.ListConsumableByMaterial(materialID, tenantID)
        if err != nil {
            return err
        }

        remaining := req.Quantity
        totalAvailable := 0.0
        for _, lot := range lots {
            totalAvailable += lot.QuantityOnHand
        }
        if totalAvailable < req.Quantity {
            return repository.ErrInsufficientStock
        }

        for _, lot := range lots {
            if remaining <= 0 {
                break
            }
            take := lot.QuantityOnHand
            if take > remaining {
                take = remaining
            }
            lot.QuantityOnHand -= take
            if lot.QuantityOnHand <= 0 {
                lot.QuantityOnHand = 0
                lot.Status = "depleted"
            } else {
                lot.Status = "active"
            }
            if err := lotRepo.Update(&lot); err != nil {
                return err
            }
            if err := moveRepo.Create(&model.StockMovement{
                TenantID:      tenantID,
                MaterialID:    material.ID,
                LotID:         lot.ID,
                MovementType:  "consume",
                QuantityDelta: -take,
                Unit:          lot.Unit,
                Reason:        strings.TrimSpace(req.Reason),
                Channel:       actor.Channel,
                UserName:      actor.UserName,
                UserID:        actor.UserID,
                Remark:        strings.TrimSpace(req.Reason),
            }); err != nil {
                return err
            }
            results = append(results, consumeLotResult{
                LotID:             lot.ID,
                ConsumedQuantity:  take,
                RemainingQuantity: lot.QuantityOnHand,
                Location:          lot.Location,
                ExpireAt:          lot.ExpireAt,
            })
            remaining -= take
        }

        recordAuditLog(auditRepo, actor, "update", "material", material.ID, material.Name, marshalChanges(nil, gin.H{
            "action":   "consume",
            "quantity": req.Quantity,
            "reason":   strings.TrimSpace(req.Reason),
            "lots":     results,
        }))
        return nil
    })
    if err != nil {
        handleMaterialRepositoryError(c, err, "consume material failed")
        return
    }

    detail, err := h.repo.GetDetail(materialID, tenantID)
    if err != nil {
        handleMaterialRepositoryError(c, err, "get material failed")
        return
    }

    httpresp.OK(c, gin.H{
        "material":           detail,
        "requested_quantity": req.Quantity,
        "unit":               material.DefaultUnit,
        "consumed_lots":      results,
    })
}

func handleMaterialRepositoryError(c *gin.Context, err error, fallbackMessage string) {
    switch {
    case repository.IsNotFound(err):
        httpresp.Error(c, http.StatusNotFound, "material not found")
    case err == repository.ErrInvalidCategoryID:
        httpresp.Error(c, http.StatusBadRequest, err.Error())
    case err == repository.ErrMaterialUnitMismatch:
        httpresp.Error(c, http.StatusBadRequest, err.Error())
    case err == repository.ErrInsufficientStock:
        httpresp.Error(c, http.StatusBadRequest, err.Error())
    default:
        httpresp.Error(c, http.StatusInternalServerError, fallbackMessage)
    }
}
