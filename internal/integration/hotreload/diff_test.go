package hotreload

import (
	"testing"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestCalcDiff_OutboundChanged(t *testing.T) {
	oldCfg := baseCfg()
	newCfg := baseCfg()
	newCfg.Outbound.Endpoints = append(newCfg.Outbound.Endpoints, config.EndpointConfig{
		Name: "test", URL: "http://example.com", Enabled: true,
	})

	diff := CalcDiff(oldCfg, newCfg)
	if !diff.OutboundChanged {
		t.Error("expected OutboundChanged=true when endpoints differ")
	}
	if diff.LogChanged || diff.PortChanged || diff.DBChanged || diff.AuthChanged {
		t.Error("expected only OutboundChanged to be true")
	}
}

func TestCalcDiff_AuthChanged(t *testing.T) {
	oldCfg := baseCfg()
	newCfg := baseCfg()
	newCfg.Auth.APIKeys = []string{"new-key"}

	diff := CalcDiff(oldCfg, newCfg)
	if !diff.AuthChanged {
		t.Error("expected AuthChanged=true when APIKeys differ")
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
	if diff.OutboundChanged || diff.LogChanged || diff.PortChanged || diff.DBChanged || diff.AuthChanged {
		t.Errorf("expected no changes, got %+v", diff)
	}
}

// helpers

func baseCfg() *config.Config {
	return &config.Config{
		Server:   config.ServerConfig{Port: "8888"},
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: "./data/inventory.db"},
		Log:      config.LogConfig{Level: 0, Path: "logs/info.log"},
	}
}
