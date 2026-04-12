package app

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
	"github.com/zhaoxiaoyang741/HomeStock/internal/httpserver"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

type App struct {
	server *httpserver.Server
	db     *gorm.DB
	sqlDB  *sql.DB
}

func New(cfg *config.Config) (*App, error) {
	db, err := database.OpenAndMigrate(cfg.Database)
	if err != nil { return nil, err }
	sqlDB, err := db.DB(); if err != nil { return nil, err }

	// Repositories (GORM impl in current package)
	auditRepo := repository.NewAuditLogRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	materialRepo := repository.NewMaterialRepository(db)
	lotRepo := repository.NewStockLotRepository(db)
	moveRepo := repository.NewStockMovementRepository(db)

	// Services
	materialSvc := service.NewMaterialService(db, materialRepo, auditRepo)
	inventorySvc := service.NewInventoryService(db, materialRepo, lotRepo, moveRepo, auditRepo)

	// Handlers
	categoryHandler := handler.NewCategoryHandler(categoryRepo, auditRepo)
	materialHandler := handler.NewMaterialHandler(materialSvc, inventorySvc)
	stockLotHandler := handler.NewStockLotHandler(inventorySvc)
	stockMovementHandler := handler.NewStockMovementHandler(repository.NewStockMovementRepository(db))
	auditLogHandler := handler.NewAuditLogHandler(auditRepo)

	server := httpserver.New(cfg.Server,
		categoryHandler.RegisterRoutes,
		materialHandler.RegisterRoutes,
		stockLotHandler.RegisterRoutes,
		stockMovementHandler.RegisterRoutes,
		auditLogHandler.RegisterRoutes,
	)

	return &App{server: server, db: db, sqlDB: sqlDB}, nil
}

func (a *App) Start() error { return a.server.Start() }
func (a *App) Shutdown(ctx context.Context) error { return a.server.Shutdown(ctx) }
func (a *App) Close() error { if a == nil || a.sqlDB == nil { return nil }; return a.sqlDB.Close() }
