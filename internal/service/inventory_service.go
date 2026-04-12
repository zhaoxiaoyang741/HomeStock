package service

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
)

type InventoryService struct {
	db        *gorm.DB
	materials repository.MaterialRepo
	lots      repository.StockLotRepo
	moves     repository.StockMovementRepo
	audit     repository.AuditLogRepo
}

func NewInventoryService(db *gorm.DB, materials repository.MaterialRepo, lots repository.StockLotRepo, moves repository.StockMovementRepo, audit repository.AuditLogRepo) *InventoryService {
	return &InventoryService{db: db, materials: materials, lots: lots, moves: moves, audit: audit}
}

type InboundInput struct {
	TenantID    string
	Name        string
	Spec        string
	CategoryID  string
	MaterialID  string
	Quantity    float64
	Unit        string
	Location    string
	ExpireAt    *time.Time
	PurchasedAt *time.Time
	Notes       string
}

type ConsumeResult struct {
	LotID             string      `json:"lot_id"`
	ConsumedQuantity  float64     `json:"consumed_quantity"`
	RemainingQuantity float64     `json:"remaining_quantity"`
	Location          string      `json:"location"`
	ExpireAt          interface{} `json:"expire_at"`
}

func (s *InventoryService) Inbound(ctx context.Context, actor Actor, in InboundInput) (*model.StockLot, error) {
	var createdLot *model.StockLot
	err := database.WithTx(ctx, s.db, func(tx *gorm.DB) error {
		mRepo := gormrepo.NewMaterialRepository(tx)
		lotRepo := gormrepo.NewStockLotRepository(tx)
		moveRepo := gormrepo.NewStockMovementRepository(tx)
		auditRepo := gormrepo.NewAuditLogRepository(tx)

		material, err := s.resolveInboundMaterial(mRepo, in)
		if err != nil { return err }

		unit := strings.TrimSpace(in.Unit)
		if unit == "" { unit = material.DefaultUnit }
		if material.DefaultUnit != "" && unit != material.DefaultUnit { return repository.ErrMaterialUnitMismatch }

		lot := &model.StockLot{
			TenantID:       in.TenantID,
			MaterialID:     material.ID,
			QuantityOnHand: in.Quantity,
			Unit:           unit,
			Location:       strings.TrimSpace(in.Location),
			ExpireAt:       in.ExpireAt,
			PurchasedAt:    in.PurchasedAt,
			ReceivedAt:     time.Now(),
			Notes:          strings.TrimSpace(in.Notes),
			Status:         "active",
		}
		if err := lotRepo.Create(lot); err != nil { return err }
		if err := moveRepo.Create(&model.StockMovement{TenantID: in.TenantID, MaterialID: material.ID, LotID: lot.ID, MovementType: "inbound", QuantityDelta: in.Quantity, Unit: unit, Reason: "inbound", Channel: actor.Channel, UserName: actor.UserName, UserID: actor.UserID, Remark: strings.TrimSpace(in.Notes)}); err != nil { return err }
		var err2 error
		createdLot, err2 = lotRepo.Get(lot.ID, in.TenantID)
		if err2 != nil { return err2 }
		_ = auditRepo.Create(&model.AuditLog{TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID, Channel: actor.Channel, Action: "create", EntityType: "stock_lot", EntityID: lot.ID, EntityName: material.Name})
		return nil
	})
	if err != nil { return nil, err }
	return createdLot, nil
}

func (s *InventoryService) Consume(ctx context.Context, actor Actor, materialID, tenantID string, quantity float64, reason string) ([]ConsumeResult, error) {
	results := []ConsumeResult{}
	err := database.WithTx(ctx, s.db, func(tx *gorm.DB) error {
		materialRepo := gormrepo.NewMaterialRepository(tx)
		lotRepo := gormrepo.NewStockLotRepository(tx)
		moveRepo := gormrepo.NewStockMovementRepository(tx)
		auditRepo := gormrepo.NewAuditLogRepository(tx)

		material, err := materialRepo.Get(materialID, tenantID)
		if err != nil { return err }
		lots, err := lotRepo.ListConsumableByMaterial(materialID, tenantID)
		if err != nil { return err }

		remaining := quantity
		total := 0.0
		for _, l := range lots { total += l.QuantityOnHand }
		if total < quantity { return repository.ErrInsufficientStock }

		for _, l := range lots {
			if remaining <= 0 { break }
			take := l.QuantityOnHand
			if take > remaining { take = remaining }
			l.QuantityOnHand -= take
			if l.QuantityOnHand <= 0 { l.QuantityOnHand = 0; l.Status = "depleted" } else { l.Status = "active" }
			if err := lotRepo.Update(&l); err != nil { return err }
			if err := moveRepo.Create(&model.StockMovement{TenantID: tenantID, MaterialID: material.ID, LotID: l.ID, MovementType: "consume", QuantityDelta: -take, Unit: l.Unit, Reason: strings.TrimSpace(reason), Channel: actor.Channel, UserName: actor.UserName, UserID: actor.UserID, Remark: strings.TrimSpace(reason)}); err != nil { return err }
			results = append(results, ConsumeResult{LotID: l.ID, ConsumedQuantity: take, RemainingQuantity: l.QuantityOnHand, Location: l.Location, ExpireAt: l.ExpireAt})
			remaining -= take
		}
		_ = auditRepo.Create(&model.AuditLog{TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID, Channel: actor.Channel, Action: "update", EntityType: "material", EntityID: material.ID, EntityName: material.Name})
		return nil
	})
	if err != nil { return nil, err }
	return results, nil
}

func (s *InventoryService) Adjust(ctx context.Context, actor Actor, lotID, tenantID string, targetQuantity float64, reason, remark string) (*model.StockLot, error) {
	var updated *model.StockLot
	err := database.WithTx(ctx, s.db, func(tx *gorm.DB) error {
		lotRepo := gormrepo.NewStockLotRepository(tx)
		moveRepo := gormrepo.NewStockMovementRepository(tx)
		auditRepo := gormrepo.NewAuditLogRepository(tx)

		lot, err := lotRepo.Get(lotID, tenantID)
		if err != nil { return err }
		delta := targetQuantity - lot.QuantityOnHand
		lot.QuantityOnHand = targetQuantity
		if targetQuantity == 0 { lot.Status = "depleted" } else { lot.Status = "active" }
		if err := lotRepo.Update(lot); err != nil { return err }
		if delta != 0 {
			if err := moveRepo.Create(&model.StockMovement{TenantID: tenantID, MaterialID: lot.MaterialID, LotID: lot.ID, MovementType: "adjustment", QuantityDelta: delta, Unit: lot.Unit, Reason: strings.TrimSpace(reason), Channel: actor.Channel, UserName: actor.UserName, UserID: actor.UserID, Remark: strings.TrimSpace(remark)}); err != nil { return err }
		}
		_ = auditRepo.Create(&model.AuditLog{TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID, Channel: actor.Channel, Action: "update", EntityType: "stock_lot", EntityID: lot.ID, EntityName: lot.Material.Name})
		updated, err = lotRepo.Get(lot.ID, tenantID)
		return err
	})
	if err != nil { return nil, err }
	return updated, nil
}

func (s *InventoryService) ListLots(ctx context.Context, f repository.StockLotFilter) ([]model.StockLot, error) { return s.lots.List(f) }

func (s *InventoryService) resolveInboundMaterial(repo repository.MaterialRepo, in InboundInput) (*model.Material, error) {
	if id := strings.TrimSpace(in.MaterialID); id != "" { return repo.Get(id, in.TenantID) }
	name := strings.TrimSpace(in.Name)
	spec := strings.TrimSpace(in.Spec)
	material, err := repo.FindByNaturalKey(in.TenantID, name, spec)
	if err == nil { return material, nil }
	if !repository.IsNotFound(err) { return nil, err }
	material = &model.Material{TenantID: in.TenantID, Name: name, Spec: spec, CategoryID: strings.TrimSpace(in.CategoryID), DefaultUnit: strings.TrimSpace(in.Unit)}
	if err := repo.Create(material); err != nil { return nil, err }
	return repo.Get(material.ID, in.TenantID)
}

// UpdateLotInput and UpdateLot implement stock lot info update logic without quantity changes.
type UpdateLotInput struct {
	ExpireAt    *time.Time
	PurchasedAt *time.Time
	Location    *string
	Notes       *string
}

func (s *InventoryService) UpdateLot(ctx context.Context, actor Actor, lotID, tenantID string, in UpdateLotInput) (*model.StockLot, error) {
	lot, err := s.lots.Get(lotID, tenantID)
	if err != nil { return nil, err }
	if in.ExpireAt != nil { lot.ExpireAt = in.ExpireAt }
	if in.PurchasedAt != nil { lot.PurchasedAt = in.PurchasedAt }
	if in.Location != nil { lot.Location = strings.TrimSpace(*in.Location) }
	if in.Notes != nil { lot.Notes = strings.TrimSpace(*in.Notes) }
	if err := s.lots.Update(lot); err != nil { return nil, err }
	updated, err := s.lots.Get(lot.ID, lot.TenantID)
	if err == nil { _ = s.audit.Create(&model.AuditLog{TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID, Channel: actor.Channel, Action: "update", EntityType: "stock_lot", EntityID: updated.ID, EntityName: updated.Material.Name}) }
	return updated, err
}

