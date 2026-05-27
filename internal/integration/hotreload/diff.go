package hotreload

import (
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// ConfigDiff describes which parts of the configuration have changed.
type ConfigDiff struct {
	LogChanged      bool
	PortChanged     bool
	DBChanged       bool
	OutboundChanged bool
	AuthChanged     bool
}

// CalcDiff compares two Config structs and returns what changed.
func CalcDiff(old, new *config.Config) ConfigDiff {
	var diff ConfigDiff

	// Outbound endpoints
	oldEndpoints := old.Outbound.Endpoints
	newEndpoints := new.Outbound.Endpoints
	if len(oldEndpoints) != len(newEndpoints) {
		diff.OutboundChanged = true
	} else {
		for i := range oldEndpoints {
			if oldEndpoints[i] != newEndpoints[i] {
				diff.OutboundChanged = true
				break
			}
		}
	}

	// Auth (API keys)
	oldKeys := old.Auth.APIKeys
	newKeys := new.Auth.APIKeys
	if len(oldKeys) != len(newKeys) {
		diff.AuthChanged = true
	} else {
		for i := range oldKeys {
			if oldKeys[i] != newKeys[i] {
				diff.AuthChanged = true
				break
			}
		}
	}

	// Logger
	diff.LogChanged = old.Log.Level != new.Log.Level ||
		old.Log.Path != new.Log.Path

	// Server port (requires restart — detected for warning)
	diff.PortChanged = old.Server.Port != new.Server.Port

	// Database config (requires restart)
	diff.DBChanged = old.Database.Driver != new.Database.Driver ||
		old.Database.DSN != new.Database.DSN

	return diff
}
