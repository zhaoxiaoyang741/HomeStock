package app

import (
	"gorm.io/gorm"

	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func initServices(db *gorm.DB, authCfg config.AuthConfig) (
	uow *gormrepo.UnitOfWork,
	materialSvc *service.MaterialService,
	inventorySvc *service.InventoryService,
	authSvc *service.AuthService,
) {
	uow = gormrepo.NewUnitOfWork(db)
	materialSvc = service.NewMaterialService(uow)
	inventorySvc = service.NewInventoryService(uow)
	authSvc = service.NewAuthService(db, authCfg.JWTSecret, authCfg.TokenDurationMinutes)
	return
}
