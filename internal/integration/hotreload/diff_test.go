package hotreload

import (
	"encoding/json"
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
	if diff.ChannelsChanged["feishu"] || diff.LogChanged || diff.PortChanged || diff.DBChanged {
		t.Error("expected only ModelChanged to be true")
	}
}

func TestCalcDiff_FeishuChanged(t *testing.T) {
	oldCfg := baseCfg()
	newCfg := baseCfg()

	feishuCfg, _ := newCfg.FeishuConfig()
	feishuCfg.AppID = "new-app-id"
	_ = newCfg.SetChannelConfig("feishu", feishuCfg)

	diff := CalcDiff(oldCfg, newCfg)
	if !diff.ChannelsChanged["feishu"] {
		t.Error("expected ChannelsChanged[feishu]=true when AppID differs")
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
	if !diff.DBChanged {
		t.Error("expected DBChanged=true")
	}
}

func TestCalcDiff_NoChange(t *testing.T) {
	cfg := baseCfg()
	diff := CalcDiff(cfg, cfg)
	if diff.ModelChanged || diff.LogChanged || diff.PortChanged || diff.DBChanged || len(diff.ChannelsChanged) > 0 {
		t.Errorf("expected no changes, got %+v", diff)
	}
}

// helpers

func rawJSON(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

func baseCfg() *config.Config {
	return &config.Config{
		Server:   config.ServerConfig{Port: "8888"},
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: "./data/inventory.db"},
		Channels: map[string]json.RawMessage{
			"feishu": rawJSON(config.FeishuChannelConfig{
				Enabled:   false,
				AppID:     "cli_xxx",
				AppSecret: "secret",
			}),
			"wechat": rawJSON(config.WechatChannelConfig{
				Enabled:    false,
				Token:      "",
				AccountID:  "",
				BaseURL:    "https://ilinkai.weixin.qq.com/",
				CDNBaseURL: "https://novac2c.cdn.weixin.qq.com/c2c",
				Proxy:      "",
			}),
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
