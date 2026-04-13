package gormrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type gormRepos struct{ db *gorm.DB }

func (r *gormRepos) Categories() repository.CategoryRepo { return NewCategoryRepository(r.db) }
func (r *gormRepos) Materials() repository.MaterialRepo  { return NewMaterialRepository(r.db) }
func (r *gormRepos) StockLots() repository.StockLotRepo  { return NewStockLotRepository(r.db) }
func (r *gormRepos) StockMovements() repository.StockMovementRepo {
	return NewStockMovementRepository(r.db)
}
func (r *gormRepos) AuditLogs() repository.AuditLogRepo { return NewAuditLogRepository(r.db) }
func (r *gormRepos) SystemSettings() repository.SystemSettingRepo {
	return NewSystemSettingRepository(r.db)
}

type UnitOfWork struct{ db *gorm.DB }

func NewUnitOfWork(db *gorm.DB) *UnitOfWork { return &UnitOfWork{db: db} }

func (u *UnitOfWork) Repos() repository.Repos { return &gormRepos{db: u.db} }

func (u *UnitOfWork) WithTx(ctx context.Context, fn func(r repository.Repos) error) error {
	return database.WithTx(ctx, u.db, func(tx *gorm.DB) error {
		return fn(&gormRepos{db: tx})
	})
}
