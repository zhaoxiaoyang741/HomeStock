package app

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
	"github.com/zhaoxiaoyang741/HomeStock/internal/httpserver"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/internal/taskcenter"
	"github.com/zhaoxiaoyang741/HomeStock/internal/tasks"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

type App struct {
	server        *httpserver.Server
	db            *gorm.DB
	sqlDB         *sql.DB
	taskCenterSvc *taskcenter.TaskCenterService
}

func New(cfg *config.Config) (*App, error) {
	db, err := database.OpenAndMigrate(cfg.Database)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Services
	uow := gormrepo.NewUnitOfWork(db)
	materialSvc := service.NewMaterialService(uow)
	inventorySvc := service.NewInventoryService(uow)
	systemSettingsSvc := service.NewSystemSettingsService(uow, cfg)
	notificationSvc := service.NewNotificationService(uow, systemSettingsSvc)
	taskCenterSvc := taskcenter.NewTaskCenterService(
		uow,
		tasks.NewExpiryNotificationTaskDefinition(notificationSvc),
	)

	// Repositories exposed directly for handlers that don't need a service layer
	notificationRepo := gormrepo.NewNotificationRepository(db)

	// Handlers
	categoryHandler := handler.NewCategoryHandler(service.NewCategoryService(uow))
	materialHandler := handler.NewMaterialHandler(materialSvc, inventorySvc)
	stockLotHandler := handler.NewStockLotHandler(inventorySvc)
	stockMovementHandler := handler.NewStockMovementHandler(gormrepo.NewStockMovementRepository(db))
	auditLogHandler := handler.NewAuditLogHandler(service.NewAuditService(uow))
	systemSettingsHandler := handler.NewSystemSettingsHandler(systemSettingsSvc)
	scheduledTaskHandler := handler.NewScheduledTaskHandler(taskCenterSvc)
	schedulerHandler := handler.NewSchedulerHandler(taskCenterSvc)
	notificationHandler := handler.NewNotificationHandler(notificationRepo)

	server := httpserver.New(cfg.Server,
		categoryHandler.RegisterRoutes,
		materialHandler.RegisterRoutes,
		stockLotHandler.RegisterRoutes,
		stockMovementHandler.RegisterRoutes,
		auditLogHandler.RegisterRoutes,
		systemSettingsHandler.RegisterRoutes,
		scheduledTaskHandler.RegisterRoutes,
		schedulerHandler.RegisterRoutes,
		notificationHandler.RegisterRoutes,
	)

	return &App{
		server:        server,
		db:            db,
		sqlDB:         sqlDB,
		taskCenterSvc: taskCenterSvc,
	}, nil
}

func (a *App) Start() error {
	if err := a.taskCenterSvc.Start(context.Background()); err != nil {
		return err
	}
	return a.server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.taskCenterSvc.Stop()
	return a.server.Shutdown(ctx)
}

func (a *App) Close() error {
	if a == nil || a.sqlDB == nil {
		return nil
	}
	return a.sqlDB.Close()
}
