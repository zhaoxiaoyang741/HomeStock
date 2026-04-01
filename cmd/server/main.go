package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
	"github.com/zhaoxiaoyang741/HomeStock/internal/httpserver"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

const defaultConfigPath = "config.json"

func main() {
	cfg, err := config.Load(defaultConfigPath)
	if err != nil {
		logger.FatalCF("server", "load config failed", map[string]any{
			"error": err.Error(),
		})
	}

	db, err := database.OpenAndMigrate(cfg.Database)
	if err != nil {
		logger.FatalCF("server", "open database failed", map[string]any{
			"error": err.Error(),
		})
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.FatalCF("server", "get sql db handle failed", map[string]any{
			"error": err.Error(),
		})
	}
	defer sqlDB.Close()

	itemHandler := handler.NewItemHandler(repository.NewItemRepository(db))
	server := httpserver.New(cfg.Server, itemHandler.RegisterRoutes)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("server", "shutdown failed", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	if err := server.Start(); err != nil && !errors.Is(err, context.Canceled) {
		logger.FatalCF("server", "server stopped with error", map[string]any{
			"error": err.Error(),
		})
	}
}
