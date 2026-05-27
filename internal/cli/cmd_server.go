package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/zhaoxiaoyang741/HomeStock/internal/app"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

func newServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start the full HomeStock server (HTTP API + Gateway)",
		Long: "Starts the complete HomeStock server including the HTTP API, " +
			"agent loop, channel integrations (Feishu/WeChat), cron scheduler, " +
			"and hot-reload watcher.\n\n" +
			"Use --config to specify a custom config path. The command loads " +
			"the configuration, initializes the database, and gracefully " +
			"shuts down on SIGINT/SIGTERM.",
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

			srv, err := app.New(cfg, absConfigPath)
			if err != nil {
				return asRuntimeError(fmt.Errorf("init app: %w", err))
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			go func() {
				<-ctx.Done()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := srv.Shutdown(shutdownCtx); err != nil {
					logger.ErrorCF("server", "shutdown failed", map[string]any{"error": err.Error()})
				}
			}()

			if err := srv.Start(); err != nil && !errors.Is(err, context.Canceled) {
				return asRuntimeError(fmt.Errorf("server stopped with error: %w", err))
			}
			return nil
		},
	}
}
