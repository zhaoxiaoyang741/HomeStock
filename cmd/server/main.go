package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/app"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

func main() {
	configPath := flag.String("config", "", "Path to config.json")
	flag.Parse()

	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = os.Getenv("HOMESTOCK_CONFIG_PATH")
	}
	if cfgPath == "" {
		cfgPath = "config.json"
	}

	absConfigPath, err := filepath.Abs(cfgPath)
	if err != nil {
		logger.FatalCF("server", "resolve config path failed", map[string]any{"error": err.Error()})
	}

	cfg, err := config.Load(absConfigPath)
	if err != nil {
		logger.FatalCF("server", "load config failed", map[string]any{"error": err.Error()})
	}

	// init log config
	logger.SetLevel(logger.LogLevel(cfg.Log.Level))
	logger.EnableFileLogging(cfg.Log.Path)

	appInstance, err := app.New(cfg, absConfigPath)
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

	if err := appInstance.Start(); err != nil && !errors.Is(err, context.Canceled) {
		logger.FatalCF("server", "server stopped with error", map[string]any{"error": err.Error()})
	}
}
