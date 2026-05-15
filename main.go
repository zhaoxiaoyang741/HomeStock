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
	configPath := flag.String("config", "", "配置文件的路径")
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
		logger.FatalCF("app", "解析配置文件路径失败", map[string]any{"error": err.Error()})
	}

	cfg, err := config.Load(absConfigPath)
	if err != nil {
		logger.FatalCF("app", "加载配置失败", map[string]any{"error": err.Error()})
	}

	logger.SetLevel(logger.LogLevel(cfg.Log.Level))
	logger.EnableFileLogging(cfg.Log.Path)

	srv, err := app.New(cfg, absConfigPath)
	if err != nil {
		logger.FatalCF("app", "初始化应用失败", map[string]any{"error": err.Error()})
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("app", "关闭服务失败", map[string]any{"error": err.Error()})
		}
	}()

	if err := srv.Start(); err != nil && !errors.Is(err, context.Canceled) {
		logger.FatalCF("app", "服务异常停止", map[string]any{"error": err.Error()})
	}
}
