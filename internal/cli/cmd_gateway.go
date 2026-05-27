package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

	// Channel factory registration side effects
	_ "github.com/zhaoxiaoyang741/HomeStock/internal/app"
	"github.com/zhaoxiaoyang741/HomeStock/internal/gateway"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/database"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

func newGatewayCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gateway",
		Short: "Start only the Gateway (agent system, channels, cron) without HTTP",
		Long: "Starts the HomeStock gateway subsystem — agent loop, channel " +
			"integrations (Feishu/WeChat), cron scheduler, and hot-reload " +
			"watcher — without the HTTP server.\n\n" +
			"Useful for running the bot in headless mode. Use --config to " +
			"specify a custom config path. Gracefully shuts down on " +
			"SIGINT/SIGTERM.",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := resolveConfigPath(cmd)
			absConfigPath, err := filepath.Abs(configPath)
			if err != nil {
				return asRuntimeError(fmt.Errorf("resolve config path: %w", err))
			}

			cfg, err := config.Load(absConfigPath)
			if err != nil {
				return asRuntimeError(fmt.Errorf("load config: %w", err))
			}

			logger.SetLevel(logger.LogLevel(cfg.Log.Level))
			logger.EnableFileLogging(cfg.Log.Path)

			// Initialize DB and services (same dependency chain as app.New
			// but without the HTTP server).
			db, sqlDB, err := initDB(cfg.Database)
			if err != nil {
				return asRuntimeError(fmt.Errorf("init database: %w", err))
			}
			defer sqlDB.Close()

			uow := gormrepo.NewUnitOfWork(db)
			materialSvc := service.NewMaterialService(uow)
			inventorySvc := service.NewInventoryService(uow)

			gw, err := gateway.New(cfg, absConfigPath, materialSvc, inventorySvc, uow, db)
			if err != nil {
				return asRuntimeError(fmt.Errorf("create gateway: %w", err))
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			go func() {
				<-ctx.Done()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := gw.Stop(shutdownCtx); err != nil {
					logger.ErrorCF("gateway", "shutdown failed", map[string]any{"error": err.Error()})
				}
			}()

			if err := gw.Start(); err != nil && !errors.Is(err, context.Canceled) {
				return asRuntimeError(fmt.Errorf("gateway stopped with error: %w", err))
			}
			return nil
		},
	}
}

func initDB(cfg config.DatabaseConfig) (*gorm.DB, *sql.DB, error) {
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
