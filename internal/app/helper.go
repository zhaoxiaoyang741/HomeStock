package app

import "github.com/zhaoxiaoyang741/HomeStock/pkg/config"

// firstEnabledModel returns the first enabled model config, or falls back
// to the first entry if none have enabled explicitly set.
func firstEnabledModel(list []config.ModelConfig) *config.ModelConfig {
	if len(list) == 0 {
		return nil
	}
	for i := range list {
		if list[i].Enabled {
			return &list[i]
		}
	}
	// Backward compatibility: if no entry has enabled:true, use the first one.
	return &list[0]
}
