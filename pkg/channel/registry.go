package channel

import (
	"fmt"
	"sync"
)

// Factory creates a Channel from its channel-specific config section.
// The config argument is the per-channel config value (e.g. FeishuChannelConfig).
type Factory func(config any) (Channel, error)

var (
	registryMu   sync.RWMutex
	factories    = make(map[string]Factory)
)

// RegisterFactory registers a channel factory under the given name.
// Panics if a factory with the same name is already registered.
func RegisterFactory(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := factories[name]; exists {
		panic(fmt.Sprintf("channel factory %q already registered", name))
	}
	factories[name] = factory
}

// GetFactory returns the factory registered for the given name.
func GetFactory(name string) (Factory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := factories[name]
	return f, ok
}

// RangeFactories calls fn for each registered factory.
// Iteration stops if fn returns false.
func RangeFactories(fn func(name string, factory Factory) bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	for name, factory := range factories {
		if !fn(name, factory) {
			return
		}
	}
}
