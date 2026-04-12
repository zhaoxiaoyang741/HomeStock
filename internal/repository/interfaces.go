package repository

import (
	"context"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

// Repository interfaces used by services and handlers.

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

// Query filters and public DTOs.

type MaterialFilter struct {
	TenantID   string
	CategoryID string
	Keyword    string
}

type MaterialSummary struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	Name            string          `json:"name"`
	Spec            string          `json:"spec"`
	CategoryID      string          `json:"category_id"`
	Category        *model.Category `json:"category"`
	DefaultUnit     string          `json:"default_unit"`
	Status          string          `json:"status"`
	TotalQuantity   float64         `json:"total_quantity"`
	LotCount        int64           `json:"lot_count"`
	NearestExpireAt *time.Time      `json:"nearest_expire_at"`
	Locations       []string        `json:"locations"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type MaterialDetail struct {
	MaterialSummary
}

type StockLotFilter struct {
	TenantID     string
	MaterialID   string
	CategoryID   string
	Location     string
	Status       string
	Keyword      string
	ExpiringSoon bool
}

type StockMovementFilter struct {
	TenantID     string
	MaterialID   string
	LotID        string
	MovementType string
	StartDate    time.Time
	EndDate      time.Time
}

type AuditLogFilter struct {
	TenantID string
	Action   string
	Channel  string
	UserName string
	StartDate time.Time
	EndDate   time.Time
	Page      int
	PageSize  int
}

type AuditLogPage struct {
	Logs     []model.AuditLog `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// Repos groups repository interfaces for a storage backend.
type Repos interface {
	Categories() CategoryRepo
	Materials() MaterialRepo
	StockLots() StockLotRepo
	StockMovements() StockMovementRepo
	AuditLogs() AuditLogRepo
}


// UnitOfWork provides access to repositories and transactional execution.
type UnitOfWork interface {
	Repos() Repos
	WithTx(ctx context.Context, fn func(r Repos) error) error
}



