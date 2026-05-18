package hotreload

import (
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// ConfigDiff describes which parts of the configuration have changed.
type ConfigDiff struct {
	ModelChanged    bool
	LogChanged      bool
	PortChanged     bool
	DBChanged       bool
	ChannelsChanged map[string]bool // channel name → whether its config changed
}

// CalcDiff compares two Config structs and returns what changed.
func CalcDiff(old, new *config.Config) ConfigDiff {
	var diff ConfigDiff

	// Model — compare the active (first enabled) model's operational fields
	oldActive := firstEnabledModel(old.ModelList)
	newActive := firstEnabledModel(new.ModelList)
	if oldActive != nil && newActive != nil {
		diff.ModelChanged = oldActive.Model != newActive.Model ||
			oldActive.Provider != newActive.Provider ||
			oldActive.APIKey != newActive.APIKey ||
			oldActive.APIBase != newActive.APIBase ||
			oldActive.Enabled != newActive.Enabled
	} else if oldActive != nil || newActive != nil {
		diff.ModelChanged = true
	}

	// Channels — generic diff by name
	diff.ChannelsChanged = make(map[string]bool)
	allNames := make(map[string]struct{})
	for name := range old.Channels {
		allNames[name] = struct{}{}
	}
	for name := range new.Channels {
		allNames[name] = struct{}{}
	}
	for name := range allNames {
		oldRaw, oldOk := old.Channels[name]
		newRaw, newOk := new.Channels[name]
		if !oldOk || !newOk || string(oldRaw) != string(newRaw) {
			diff.ChannelsChanged[name] = true
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

// firstEnabledModel returns the first enabled model, or the first entry if
// none are explicitly enabled (backward compatibility).
func firstEnabledModel(list []config.ModelConfig) *config.ModelConfig {
	if len(list) == 0 {
		return nil
	}
	for i := range list {
		if list[i].Enabled {
			return &list[i]
		}
	}
	return &list[0]
}
