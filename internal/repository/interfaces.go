package repository

import "github.com/zhaoxiaoyang741/HomeStock/internal/model"

// Interfaces only; current GORM structs in this package satisfy them.

type CategoryRepo interface {
    Create(category *model.Category) error
    Get(id, tenantID string) (*model.Category, error)
    List(tenantID string) ([]model.Category, error)
    Update(category *model.Category) error
    Delete(id, tenantID string) error
}

type MaterialRepo interface {
    Create(material *model.Material) error
    Get(id, tenantID string) (*model.Material, error)
    FindByNaturalKey(tenantID, name, spec string) (*model.Material, error)
    Update(material *model.Material) error
    List(filter MaterialFilter) ([]MaterialSummary, error)
    GetDetail(id, tenantID string) (*MaterialDetail, error)
}

type StockLotRepo interface {
    Create(lot *model.StockLot) error
    Get(id, tenantID string) (*model.StockLot, error)
    Update(lot *model.StockLot) error
    List(filter StockLotFilter) ([]model.StockLot, error)
    ListConsumableByMaterial(materialID, tenantID string) ([]model.StockLot, error)
}

type StockMovementRepo interface {
    Create(m *model.StockMovement) error
    List(f StockMovementFilter) ([]model.StockMovement, error)
}

type AuditLogRepo interface {
    Create(log *model.AuditLog) error
    List(f AuditLogFilter) (*AuditLogPage, error)
}
