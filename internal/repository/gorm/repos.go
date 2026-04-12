package gormrepo

import (
	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

func NewCategoryRepository(db *gorm.DB) *repository.CategoryRepository { return repository.NewCategoryRepository(db) }
func NewMaterialRepository(db *gorm.DB) *repository.MaterialRepository { return repository.NewMaterialRepository(db) }
func NewStockLotRepository(db *gorm.DB) *repository.StockLotRepository { return repository.NewStockLotRepository(db) }
func NewStockMovementRepository(db *gorm.DB) *repository.StockMovementRepository { return repository.NewStockMovementRepository(db) }
func NewAuditLogRepository(db *gorm.DB) *repository.AuditLogRepository { return repository.NewAuditLogRepository(db) }
