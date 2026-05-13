package hotreload

import (
	"testing"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestCalcDiff_ModelChanged(t *testing.T) {
	oldCfg := cfgWithModel("openai", "gpt-4o", "https://api.openai.com/v1")
	newCfg := cfgWithModel("openai", "gpt-4o", "https://my-proxy.com/v1")

	diff := CalcDiff(oldCfg, newCfg)
	if !diff.ModelChanged {
		t.Error("expected ModelChanged=true when APIBase differs")
	}
	if diff.FeishuChanged || diff.LogChanged || diff.PortChanged || diff.DatabaseChanged {
		t.Error("expected only ModelChanged to be true")
	}
}

func TestCalcDiff_FeishuChanged(t *testing.T) {
	oldCfg := baseCfg()
	newCfg := baseCfg()
	newCfg.Channels.Feishu.AppID = "new-app-id"

	diff := CalcDiff(oldCfg, newCfg)
	if !diff.FeishuChanged {
		t.Error("expected FeishuChanged=true when AppID differs")
	}
}

func TestCalcDiff_PortChanged(t *testing.T) {
	oldCfg := baseCfg()
	newCfg := baseCfg()
	newCfg.Server.Port = "9999"

	diff := CalcDiff(oldCfg, newCfg)
	if !diff.PortChanged {
		t.Error("expected PortChanged=true")
	}
}

func TestCalcDiff_LogChanged(t *testing.T) {
	oldCfg := baseCfg()
	newCfg := baseCfg()
	newCfg.Log.Level = 1

	diff := CalcDiff(oldCfg, newCfg)
	if !diff.LogChanged {
		t.Error("expected LogChanged=true")
	}
}

func TestCalcDiff_DatabaseChanged(t *testing.T) {
	oldCfg := baseCfg()
	newCfg := baseCfg()
	newCfg.Database.DSN = "postgres://localhost/newdb"

	diff := CalcDiff(oldCfg, newCfg)
	if !diff.DatabaseChanged {
		t.Error("expected DatabaseChanged=true")
	}
}

func TestCalcDiff_NoChange(t *testing.T) {
	cfg := baseCfg()
	diff := CalcDiff(cfg, cfg)
	if diff != (ConfigDiff{}) {
		t.Errorf("expected no changes, got %+v", diff)
	}
}

// helpers

func baseCfg() *config.Config {
	return &config.Config{
		Server:   config.ServerConfig{Port: "8888"},
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: "./data/inventory.db"},
		Channels: config.ChannelsConfig{
			Feishu: config.FeishuChannelConfig{
				Enabled:   false,
				AppID:     "cli_xxx",
				AppSecret: "secret",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "default", Model: "gpt-4o", Provider: "openai", Enabled: true, APIBase: "https://api.openai.com/v1"},
		},
		Log: config.LogConfig{Level: 0, Path: "logs/info.log"},
	}
}

func cfgWithModel(provider, model, apiBase string) *config.Config {
	cfg := baseCfg()
	cfg.ModelList[0].Provider = provider
	cfg.ModelList[0].Model = model
	cfg.ModelList[0].APIBase = apiBase
	return cfg
}
