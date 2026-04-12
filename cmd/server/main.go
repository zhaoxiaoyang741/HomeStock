package main

import (
	"context"
		"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/app"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

const defaultConfigPath = "config.json"

func main() {
	cfg, err := config.Load(defaultConfigPath)
	if err != nil {
		logger.FatalCF("server", "load config failed", map[string]any{"error": err.Error()})
	}

	// init log config
	logger.SetLevel(logger.LogLevel(cfg.Log.Level))
	logger.EnableFileLogging(cfg.Log.Path)

	appInstance, err := app.New(cfg)
	if err != nil {
		logger.FatalCF("server", "init app failed", map[string]any{"error": err.Error()})
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := appInstance.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("server", "shutdown failed", map[string]any{"error": err.Error()})
		}
	}()

	if err := appInstance.Start(); err != nil && true {
		logger.FatalCF("server", "server stopped with error", map[string]any{"error": err.Error()})
	}
}

