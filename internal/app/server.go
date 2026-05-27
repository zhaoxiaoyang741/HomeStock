package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	"github.com/zhaoxiaoyang741/HomeStock/internal/gateway"
	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/database"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/server"
)

// Server is the application container, owning the database connection, HTTP
// server, and an embedded Gateway that manages the Agent runtime.
type Server struct {
	configPath string
	server     *server.Server
	db         *gorm.DB
	sqlDB      *sql.DB
	gateway    *gateway.Gateway
}

// New is the composition root. It initializes database, services, Gateway,
// and the HTTP server in dependency order.
func New(cfg *config.Config, configPath string) (*Server, error) {
	// 1. Database
	db, sqlDB, err := initDatabase(cfg.Database)
	if err != nil {
		return nil, err
	}

	// 2. Services
	uow, materialSvc, inventorySvc, authSvc := initServices(db, cfg.Auth)

	// Auto-create admin user if not exists (first startup)
	if err := initAdminUser(authSvc); err != nil {
		return nil, err
	}

	// Auto-generate JWT secret if none configured
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = authSvc.GetSecretHex()
		logger.InfoCF("app", "auto-generated JWT secret (set HOMESTOCK_AUTH_JWT_SECRET to persist across restarts)", map[string]any{
			"secret_hex": cfg.Auth.JWTSecret,
		})
	}

	// 3. Gateway (cron, outbound, hot-reload)
	gw, err := gateway.New(cfg, configPath, uow)
	if err != nil {
		return nil, err
	}

	// 4. HTTP server (routes mounted from Gateway handlers + app handlers)
	srv := initServer(cfg.Server, db, uow, authSvc, gw, materialSvc, inventorySvc)

	return &Server{
		configPath: configPath,
		server:     srv,
		db:         db,
		sqlDB:      sqlDB,
		gateway:    gw,
	}, nil
}

// Start begins all subsystems: Gateway (agent loop, cron, channels) and the
// HTTP server (blocking).
func (s *Server) Start() error {
	if err := s.gateway.Start(); err != nil {
		return err
	}
	return s.server.Start()
}

// Shutdown gracefully stops all subsystems in reverse dependency order:
// HTTP server → Gateway → Database.
func (s *Server) Shutdown(ctx context.Context) error {
	// 1. Stop HTTP first — no new requests
	if err := s.server.Shutdown(ctx); err != nil {
		logger.ErrorCF("app", "http server shutdown error", map[string]any{"error": err.Error()})
	}

	// 2. Stop Gateway (agent, cron, channels)
	if err := s.gateway.Stop(ctx); err != nil {
		logger.ErrorCF("app", "gateway stop error", map[string]any{"error": err.Error()})
	}

	// 3. Close DB
	return s.sqlDB.Close()
}

// ---------------------------------------------------------------------------
// Initialization helpers
// ---------------------------------------------------------------------------

func initDatabase(cfg config.DatabaseConfig) (*gorm.DB, *sql.DB, error) {
	db, err := database.OpenAndMigrate(cfg)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	return db, sqlDB, nil
}

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

func initAdminUser(authSvc *service.AuthService) error {
	ctx := context.Background()

	key := make([]byte, 12)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("generate admin password: %w", err)
	}
	password := hex.EncodeToString(key)

	_, err := authSvc.Register(ctx, "admin", password, "Admin")
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			return nil
		}
		return fmt.Errorf("create admin user: %w", err)
	}

	logger.InfoCF("app", "========================================", nil)
	logger.InfoCF("app", "  Admin user created on first startup!", nil)
	logger.InfoCF("app", "  Username: admin", nil)
	logger.InfoCF("app", fmt.Sprintf("  Password: %s", password), nil)
	logger.InfoCF("app", "  Please change the password after login.", nil)
	logger.InfoCF("app", "========================================", nil)
	return nil
}

func initServer(
	cfg config.ServerConfig,
	db *gorm.DB,
	uow *gormrepo.UnitOfWork,
	authSvc *service.AuthService,
	gw *gateway.Gateway,
	materialSvc *service.MaterialService,
	inventorySvc *service.InventoryService,
) *server.Server {
	authHandler := handler.NewAuthHandler(authSvc)
	categoryService := service.NewCategoryService(uow)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	materialHandler := handler.NewMaterialHandler(materialSvc, inventorySvc)
	stockLotHandler := handler.NewStockLotHandler(inventorySvc)
	stockMovementHandler := handler.NewStockMovementHandler(gormrepo.NewStockMovementRepository(db))
	auditService := service.NewAuditService(uow)
	auditLogHandler := handler.NewAuditLogHandler(auditService)

	authMw := httpreq.JWTAuthMiddleware(authSvc)

	// Collect all route registrations: app-level handlers + Gateway webhook handlers
	protected := []server.RegisterRoutesFunc{
		categoryHandler.RegisterRoutes,
		materialHandler.RegisterRoutes,
		stockLotHandler.RegisterRoutes,
		stockMovementHandler.RegisterRoutes,
		auditLogHandler.RegisterRoutes,
		handler.NewBatchHandler(inventorySvc, materialSvc).RegisterRoutes,
		authHandler.RegisterProtectedRoutes,
	}

	srv := server.New(cfg,
		[]server.RegisterRoutesFunc{
			authHandler.RegisterRoutes,
		},
		protected,
		server.AuthMiddleware(authMw),
	)

	// Register ops endpoint for manual config reload
	srv.Engine().POST("/reload", func(c *gin.Context) {
		if err := gw.Reload(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "reload failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "reload completed"})
	})

	return srv
}
