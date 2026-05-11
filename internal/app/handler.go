package app

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	"github.com/zhaoxiaoyang741/HomeStock/internal/agent"
	"github.com/zhaoxiaoyang741/HomeStock/internal/channel/feishu"
	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
	"github.com/zhaoxiaoyang741/HomeStock/internal/hotreload"
	"github.com/zhaoxiaoyang741/HomeStock/internal/httpserver"
	"github.com/zhaoxiaoyang741/HomeStock/internal/llm"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// initModelHandler creates the ModelHandler and wires the model swap callback.
func initModelHandler(configPath string, modelCfg *config.ModelConfig, agentLoop *agent.AgentLoop) *handler.ModelHandler {
	modelHandler := handler.NewModelHandler(configPath)
	if modelCfg != nil {
		modelHandler.SetActiveName(modelCfg.ModelName)
	}
	modelHandler.SetSwapFn(func(name string, cfg config.ModelConfig) error {
		provider, err := llm.NewProvider(cfg)
		if err != nil {
			return fmt.Errorf("create provider for model %q: %w", name, err)
		}
		agentLoop.SwapProvider(provider)
		return nil
	})
	return modelHandler
}

// initHotReload creates the hot-reload orchestrator.
func initHotReload(
	configPath string,
	agentLoop *agent.AgentLoop,
	feishuCh *feishu.FeishuChannel,
	oauthSvc *feishu.OAuthService,
	modelHnd *handler.ModelHandler,
) *hotreload.Orchestrator {
	return hotreload.NewOrchestrator(configPath, agentLoop, feishuCh, oauthSvc, modelHnd)
}

// initServer creates all HTTP handlers, registers routes, and builds the HTTP server.
func initServer(
	cfg config.ServerConfig,
	db *gorm.DB,
	uow *gormrepo.UnitOfWork,
	authSvc *service.AuthService,
	orch *hotreload.Orchestrator,
	materialSvc *service.MaterialService,
	inventorySvc *service.InventoryService,
	feishuHandler *handler.FeishuHandler,
	modelHandler *handler.ModelHandler,
) *httpserver.Server {
	authHandler := handler.NewAuthHandler(authSvc)
	categoryService := service.NewCategoryService(uow)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	materialHandler := handler.NewMaterialHandler(materialSvc, inventorySvc)
	stockLotHandler := handler.NewStockLotHandler(inventorySvc)
	stockMovementHandler := handler.NewStockMovementHandler(gormrepo.NewStockMovementRepository(db))
	auditService := service.NewAuditService(uow)
	auditLogHandler := handler.NewAuditLogHandler(auditService)

	authMw := httpreq.JWTAuthMiddleware(authSvc)

	server := httpserver.New(cfg,
		[]httpserver.RegisterRoutesFunc{
			authHandler.RegisterRoutes,
		},
		[]httpserver.RegisterRoutesFunc{
			categoryHandler.RegisterRoutes,
			materialHandler.RegisterRoutes,
			stockLotHandler.RegisterRoutes,
			stockMovementHandler.RegisterRoutes,
			auditLogHandler.RegisterRoutes,
			feishuHandler.RegisterRoutes,
			modelHandler.RegisterRoutes,
			authHandler.RegisterProtectedRoutes,
		},
		httpserver.AuthMiddleware(authMw),
	)

	// Register ops endpoint for manual config reload
	server.Engine().POST("/reload", func(c *gin.Context) {
		if err := orch.Reload(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "reload failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "reload completed"})
	})

	return server
}
