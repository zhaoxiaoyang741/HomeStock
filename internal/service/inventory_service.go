package service

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type InventoryService struct {
	uow repository.UnitOfWork
}

func NewInventoryService(uow repository.UnitOfWork) *InventoryService {
	return &InventoryService{uow: uow}
}

type InboundInput struct {
	TenantID        string
	Name            string
	Spec            string
	CategoryID      string
	MaterialID      string
	Quantity        float64
	Unit            string  // 用户传入的单位（可能是采购单位，也可能是库存单位）
	Price           float64 // 单价
	ShoppingLocation string // 购买地点
	Location        string
	ExpireAt        *time.Time
	PurchasedAt     *time.Time
	Notes           string
}

type ConsumeResult struct {
	LotID             string      `json:"lot_id"`
	ConsumedQuantity  float64     `json:"consumed_quantity"`
	RemainingQuantity float64     `json:"remaining_quantity"`
	Location          string      `json:"location"`
	ExpireAt          interface{} `json:"expire_at"`
}

// resolveStockUnit converts user-provided quantity to the material's stock unit.
// Returns (stockQuantity, stockUnit).
// Example: unit="箱", factor=12, quantity=1 → (12, "瓶")
func resolveStockUnit(unit string, quantity float64, material *model.Material) (float64, string) {
	inputUnit := strings.TrimSpace(unit)
	if inputUnit == "" {
		// No unit specified — use stock unit, quantity as-is
		return quantity, material.StockUnit
	}

	// If user's unit matches the stock unit, no conversion needed
	if inputUnit == material.StockUnit {
		return quantity, material.StockUnit
	}

	// If user's unit matches purchase unit, apply conversion factor
	if inputUnit == material.PurchaseUnit && material.PurchaseUnit != "" && material.UnitFactor > 0 {
		stockQty := quantity * material.UnitFactor
		return stockQty, material.StockUnit
	}

	// Unknown unit — keep as-is, assume it's already in stock unit
	return quantity, inputUnit
}

// calcDefaultExpireAt calculates the default expiry date based on material settings.
// Returns nil if no expiry should be set (material sets -1 = never expires).
func calcDefaultExpireAt(material *model.Material) *time.Time {
	if material.DefaultBestBeforeDays == -1 {
		return nil // never expires
	}
	if material.DefaultBestBeforeDays > 0 {
		t := time.Now().AddDate(0, 0, material.DefaultBestBeforeDays)
		return &t
	}
	return nil // 0 means no default
}

func (s *InventoryService) Inbound(ctx context.Context, actor Actor, in InboundInput) ([]*model.StockLot, error) {
	var createdLots []*model.StockLot
	err := s.uow.WithTx(ctx, func(r repository.Repos) error {
		mRepo := r.Materials()
		lotRepo := r.StockLots()
		moveRepo := r.StockMovements()
		auditRepo := r.AuditLogs()

		material, err := s.resolveInboundMaterial(mRepo, in)
		if err != nil {
			return err
		}

		// ——— Unit conversion ———
		stockQty, stockUnit := resolveStockUnit(in.Unit, in.Quantity, material)
		if stockQty <= 0 {
			stockQty = in.Quantity
		}
		if stockUnit == "" {
			stockUnit = material.DefaultUnit
			if stockUnit == "" {
				stockUnit = "个"
			}
		}

		// ——— Default best-before ———
		expireAt := in.ExpireAt
		if expireAt == nil {
			expireAt = calcDefaultExpireAt(material)
		}

		// ——— Price ———
		price := in.Price
		if price < 0 {
			price = 0
		}

		// ——— TrackByUnit: create one lot per unit ———
		if material.TrackByUnit {
			wholeUnits := int(math.Floor(stockQty))
			if wholeUnits < 1 {
				wholeUnits = 1
			}
			for i := 0; i < wholeUnits; i++ {
				lot := &model.StockLot{
					TenantID:         in.TenantID,
					MaterialID:       material.ID,
					QuantityOnHand:   1,
					Unit:             stockUnit,
					Price:            price,
					ShoppingLocation: strings.TrimSpace(in.ShoppingLocation),
					Location:         strings.TrimSpace(in.Location),
					ExpireAt:         expireAt,
					PurchasedAt:      in.PurchasedAt,
					ReceivedAt:       time.Now(),
					Notes:            strings.TrimSpace(in.Notes),
					Status:           "active",
				}
				if err := lotRepo.Create(lot); err != nil {
					return err
				}
				if err := moveRepo.Create(&model.StockMovement{
					TenantID: in.TenantID, MaterialID: material.ID, LotID: lot.ID,
					MovementType: "inbound", QuantityDelta: 1, Unit: stockUnit,
					Price: price, Reason: "入库", Channel: actor.Channel,
					UserName: actor.UserName, UserID: actor.UserID,
					Remark: strings.TrimSpace(in.Notes),
				}); err != nil {
					return err
				}
				createdLots = append(createdLots, lot)
			}
		} else {
			// ——— Normal: one lot with total quantity ———
			lot := &model.StockLot{
				TenantID:         in.TenantID,
				MaterialID:       material.ID,
				QuantityOnHand:   stockQty,
				Unit:             stockUnit,
				Price:            price,
				ShoppingLocation: strings.TrimSpace(in.ShoppingLocation),
				Location:         strings.TrimSpace(in.Location),
				ExpireAt:         expireAt,
				PurchasedAt:      in.PurchasedAt,
				ReceivedAt:       time.Now(),
				Notes:            strings.TrimSpace(in.Notes),
				Status:           "active",
			}
			if err := lotRepo.Create(lot); err != nil {
				return err
			}
			if err := moveRepo.Create(&model.StockMovement{
				TenantID: in.TenantID, MaterialID: material.ID, LotID: lot.ID,
				MovementType: "inbound", QuantityDelta: stockQty, Unit: stockUnit,
				Price: price, Reason: "入库", Channel: actor.Channel,
				UserName: actor.UserName, UserID: actor.UserID,
				Remark: strings.TrimSpace(in.Notes),
			}); err != nil {
				return err
			}
			createdLots = append(createdLots, lot)
		}

		_ = auditRepo.Create(&model.AuditLog{
			TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID,
			Channel: actor.Channel, Action: "create", EntityType: "stock_lot",
			EntityID: material.ID, EntityName: material.Name,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Reload created lots to get full material associations
	loaded := make([]*model.StockLot, 0, len(createdLots))
	for _, lot := range createdLots {
		if l, e := s.uow.Repos().StockLots().Get(lot.ID, lot.TenantID); e == nil {
			loaded = append(loaded, l)
		}
	}
	return loaded, nil
}

func (s *InventoryService) Consume(ctx context.Context, actor Actor, materialID, tenantID string, quantity float64, reason string) ([]ConsumeResult, error) {
	results := []ConsumeResult{}
	err := s.uow.WithTx(ctx, func(r repository.Repos) error {
		materialRepo := r.Materials()
		lotRepo := r.StockLots()
		moveRepo := r.StockMovements()
		auditRepo := r.AuditLogs()

		material, err := materialRepo.Get(materialID, tenantID)
		if err != nil {
			return err
		}
		lots, err := lotRepo.ListConsumableByMaterial(materialID, tenantID)
		if err != nil {
			return err
		}

		remaining := quantity
		total := 0.0
		for _, l := range lots {
			total += l.QuantityOnHand
		}
		if total < quantity {
			return repository.ErrInsufficientStock
		}

		consumeReason := strings.TrimSpace(reason)
		if consumeReason == "" {
			consumeReason = "消耗"
		}
		for _, l := range lots {
			if remaining <= 0 {
				break
			}
			take := l.QuantityOnHand
			if take > remaining {
				take = remaining
			}
			l.QuantityOnHand -= take
			if l.QuantityOnHand <= 0 {
				l.QuantityOnHand = 0
				l.Status = "depleted"
			} else {
				l.Status = "active"
			}
			if err := lotRepo.Update(&l); err != nil {
				return err
			}
			if err := moveRepo.Create(&model.StockMovement{
				TenantID: tenantID, MaterialID: material.ID, LotID: l.ID,
				MovementType: "consume", QuantityDelta: -take, Unit: l.Unit,
				Price: l.Price, Reason: consumeReason, Channel: actor.Channel,
				UserName: actor.UserName, UserID: actor.UserID, Remark: consumeReason,
			}); err != nil {
				return err
			}
			results = append(results, ConsumeResult{
				LotID:             l.ID,
				ConsumedQuantity:  take,
				RemainingQuantity: l.QuantityOnHand,
				Location:          l.Location,
				ExpireAt:          l.ExpireAt,
			})
			remaining -= take
		}
		_ = auditRepo.Create(&model.AuditLog{
			TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID,
			Channel: actor.Channel, Action: "update", EntityType: "material",
			EntityID: material.ID, EntityName: material.Name,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *InventoryService) Adjust(ctx context.Context, actor Actor, lotID, tenantID string, targetQuantity float64, reason, remark string) (*model.StockLot, error) {
	var updated *model.StockLot
	err := s.uow.WithTx(ctx, func(r repository.Repos) error {
		lotRepo := r.StockLots()
		moveRepo := r.StockMovements()
		auditRepo := r.AuditLogs()

		lot, err := lotRepo.Get(lotID, tenantID)
		if err != nil {
			return err
		}
		delta := targetQuantity - lot.QuantityOnHand
		lot.QuantityOnHand = targetQuantity
		if targetQuantity == 0 {
			lot.Status = "depleted"
		} else {
			lot.Status = "active"
		}
		if err := lotRepo.Update(lot); err != nil {
			return err
		}
		adjustReason := strings.TrimSpace(reason)
		if adjustReason == "" {
			adjustReason = "调整"
		}
		if delta != 0 {
			if err := moveRepo.Create(&model.StockMovement{
				TenantID: tenantID, MaterialID: lot.MaterialID, LotID: lot.ID,
				MovementType: "adjustment", QuantityDelta: delta, Unit: lot.Unit,
				Reason: adjustReason, Channel: actor.Channel,
				UserName: actor.UserName, UserID: actor.UserID, Remark: strings.TrimSpace(remark),
			}); err != nil {
				return err
			}
		}
		_ = auditRepo.Create(&model.AuditLog{
			TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID,
			Channel: actor.Channel, Action: "update", EntityType: "stock_lot",
			EntityID: lot.ID, EntityName: lot.Material.Name,
		})
		var err2 error
		updated, err2 = lotRepo.Get(lot.ID, lot.TenantID)
		return err2
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// OpenProduct marks a stock lot as opened, optionally recalculating the best-before date.
func (s *InventoryService) OpenProduct(ctx context.Context, actor Actor, lotID, tenantID string, newBestBeforeDays *int) (*model.StockLot, error) {
	var updated *model.StockLot
	err := s.uow.WithTx(ctx, func(r repository.Repos) error {
		lotRepo := r.StockLots()
		moveRepo := r.StockMovements()

		lot, err := lotRepo.Get(lotID, tenantID)
		if err != nil {
			return err
		}
		if lot.IsOpen {
			return nil // already opened
		}
		now := time.Now()
		lot.IsOpen = true
		lot.OpenedAt = &now

		// Recalculate best-before if a value is provided
		if newBestBeforeDays != nil && *newBestBeforeDays >= 0 {
			newExpiry := now.AddDate(0, 0, *newBestBeforeDays)
			lot.ExpireAt = &newExpiry
		}

		if err := lotRepo.Update(lot); err != nil {
			return err
		}
		if err := moveRepo.Create(&model.StockMovement{
			TenantID: tenantID, MaterialID: lot.MaterialID, LotID: lot.ID,
			MovementType: "product-opened", QuantityDelta: 0, Unit: lot.Unit,
			Reason: "开封", Channel: actor.Channel,
			UserName: actor.UserName, UserID: actor.UserID,
		}); err != nil {
			return err
		}
		var err2 error
		updated, err2 = lotRepo.Get(lot.ID, lot.TenantID)
		return err2
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// TransferLot moves a stock lot from one location to another.
func (s *InventoryService) TransferLot(ctx context.Context, actor Actor, lotID, tenantID, targetLocation string) (*model.StockLot, error) {
	var updated *model.StockLot
	err := s.uow.WithTx(ctx, func(r repository.Repos) error {
		lotRepo := r.StockLots()
		moveRepo := r.StockMovements()

		lot, err := lotRepo.Get(lotID, tenantID)
		if err != nil {
			return err
		}
		if lot.Status != "active" || lot.QuantityOnHand <= 0 {
			return repository.ErrInsufficientStock
		}
		sourceLocation := lot.Location
		lot.Location = strings.TrimSpace(targetLocation)
		if err := lotRepo.Update(lot); err != nil {
			return err
		}

		// Record paired transfer movements
		if err := moveRepo.Create(&model.StockMovement{
			TenantID: tenantID, MaterialID: lot.MaterialID, LotID: lot.ID,
			MovementType: "transfer_from", QuantityDelta: -lot.QuantityOnHand,
			Unit: lot.Unit, Reason: "从" + sourceLocation + "转移到" + targetLocation,
			Channel: actor.Channel, UserName: actor.UserName, UserID: actor.UserID,
		}); err != nil {
			return err
		}
		if err := moveRepo.Create(&model.StockMovement{
			TenantID: tenantID, MaterialID: lot.MaterialID, LotID: lot.ID,
			MovementType: "transfer_to", QuantityDelta: lot.QuantityOnHand,
			Unit: lot.Unit, Reason: "从" + sourceLocation + "转移到" + targetLocation,
			Channel: actor.Channel, UserName: actor.UserName, UserID: actor.UserID,
		}); err != nil {
			return err
		}
		var err2 error
		updated, err2 = lotRepo.Get(lot.ID, lot.TenantID)
		return err2
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *InventoryService) ListLots(ctx context.Context, f repository.StockLotFilter) ([]model.StockLot, error) {
	return s.uow.Repos().StockLots().List(f)
}

func (s *InventoryService) resolveInboundMaterial(repo repository.MaterialRepo, in InboundInput) (*model.Material, error) {
	if id := strings.TrimSpace(in.MaterialID); id != "" {
		return repo.Get(id, in.TenantID)
	}
	name := strings.TrimSpace(in.Name)
	spec := strings.TrimSpace(in.Spec)
	material, err := repo.FindByNaturalKey(in.TenantID, name, spec)
	if err == nil {
		return material, nil
	}
	if !repository.IsNotFound(err) {
		return nil, err
	}
	material = &model.Material{
		TenantID:    in.TenantID,
		Name:        name,
		Spec:        spec,
		CategoryID:  strings.TrimSpace(in.CategoryID),
		DefaultUnit: strings.TrimSpace(in.Unit),
	}
	// Also propagate purchase/stock unit if provided during first-time creation
	if pu := strings.TrimSpace(in.Unit); pu != "" {
		material.PurchaseUnit = pu
	}
	if err := repo.Create(material); err != nil {
		return nil, err
	}
	return repo.Get(material.ID, in.TenantID)
}

func (s *InventoryService) VoidLot(ctx context.Context, actor Actor, lotID, tenantID string) (*model.StockLot, error) {
	var updated *model.StockLot
	err := s.uow.WithTx(ctx, func(r repository.Repos) error {
		lotRepo := r.StockLots()
		moveRepo := r.StockMovements()
		auditRepo := r.AuditLogs()

		lot, err := lotRepo.Get(lotID, tenantID)
		if err != nil {
			return err
		}
		delta := -lot.QuantityOnHand
		lot.QuantityOnHand = 0
		lot.Status = "void"
		if err := lotRepo.Update(lot); err != nil {
			return err
		}
		if delta != 0 {
			if err := moveRepo.Create(&model.StockMovement{
				TenantID: tenantID, MaterialID: lot.MaterialID, LotID: lot.ID,
				MovementType: "void", QuantityDelta: delta, Unit: lot.Unit,
				Reason: "作废", Channel: actor.Channel,
				UserName: actor.UserName, UserID: actor.UserID,
			}); err != nil {
				return err
			}
		}
		_ = auditRepo.Create(&model.AuditLog{
			TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID,
			Channel: actor.Channel, Action: "delete", EntityType: "stock_lot",
			EntityID: lot.ID, EntityName: lot.Material.Name,
		})
		var err2 error
		updated, err2 = lotRepo.Get(lot.ID, lot.TenantID)
		return err2
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// UpdateLotInput and UpdateLot implement stock lot info update logic without quantity changes.
type UpdateLotInput struct {
	ExpireAt        *time.Time
	PurchasedAt     *time.Time
	Location        *string
	Notes           *string
	ShoppingLocation *string // 新增：购买地点
}

func (s *InventoryService) UpdateLot(ctx context.Context, actor Actor, lotID, tenantID string, in UpdateLotInput) (*model.StockLot, error) {
	lot, err := s.uow.Repos().StockLots().Get(lotID, tenantID)
	if err != nil {
		return nil, err
	}
	if in.ExpireAt != nil {
		lot.ExpireAt = in.ExpireAt
	}
	if in.PurchasedAt != nil {
		lot.PurchasedAt = in.PurchasedAt
	}
	if in.Location != nil {
		lot.Location = strings.TrimSpace(*in.Location)
	}
	if in.Notes != nil {
		lot.Notes = strings.TrimSpace(*in.Notes)
	}
	if in.ShoppingLocation != nil {
		lot.ShoppingLocation = strings.TrimSpace(*in.ShoppingLocation)
	}
	if err := s.uow.Repos().StockLots().Update(lot); err != nil {
		return nil, err
	}
	updated, err := s.uow.Repos().StockLots().Get(lot.ID, lot.TenantID)
	if err == nil {
		_ = s.uow.Repos().AuditLogs().Create(&model.AuditLog{
			TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID,
			Channel: actor.Channel, Action: "update", EntityType: "stock_lot",
			EntityID: updated.ID, EntityName: updated.Material.Name,
		})
	}
	return updated, err
}
