package channel

import (
	"context"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/bus"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// WebhookRegistrar is implemented by handlers that register HTTP routes
// on the /api/v1 router group.
type WebhookRegistrar interface {
	// Name returns the handler name, e.g. "feishu", "model".
	Name() string

	// RegisterRoutes mounts the handler's endpoints under the API group.
	RegisterRoutes(api *gin.RouterGroup)
}

// ConfigChangeHandler handles runtime config changes for a specific subsystem
// (typically a channel). Implementations are registered with the hot-reload
// orchestrator so that changes detected from config file polling are applied.
type ConfigChangeHandler interface {
	// Name returns the name of the handler, matching the channel name.
	Name() string

	// HandleConfigChange applies a config change detected by the hot-reload
	// orchestrator. The implementation compares oldCfg and newCfg and takes
	// the appropriate action (reconfigure, stop, start, etc.).
	HandleConfigChange(ctx context.Context, oldCfg, newCfg *config.Config) error
}

// HandlerDeps bundles all dependencies needed by handler factories.
type HandlerDeps struct {
	Config     *config.Config
	ConfigPath string
	ChannelMgr *Manager
	MsgBus     *bus.MessageBus
	DB         *gorm.DB
}

// HandlerFactory creates a WebhookRegistrar from HandlerDeps.
// The optional ConfigChangeHandler is registered with the hot-reload
// orchestrator for automatic config change handling.
type HandlerFactory func(deps *HandlerDeps) (WebhookRegistrar, ConfigChangeHandler, error)

var (
	handlerRegistryMu sync.RWMutex
	handlerFactories  = make(map[string]HandlerFactory)
)

// RegisterHandlerFactory registers a handler factory under the given name.
// Panics if a factory with the same name is already registered.
func RegisterHandlerFactory(name string, factory HandlerFactory) {
	handlerRegistryMu.Lock()
	defer handlerRegistryMu.Unlock()
	if _, exists := handlerFactories[name]; exists {
		panic(fmt.Sprintf("handler factory %q already registered", name))
	}
	handlerFactories[name] = factory
}

// GetHandlerFactory returns the factory registered for the given name.
func GetHandlerFactory(name string) (HandlerFactory, bool) {
	handlerRegistryMu.RLock()
	defer handlerRegistryMu.RUnlock()
	f, ok := handlerFactories[name]
	return f, ok
}

// RangeHandlerFactories calls fn for each registered handler factory.
// Iteration stops if fn returns false.
func RangeHandlerFactories(fn func(name string, factory HandlerFactory) bool) {
	handlerRegistryMu.RLock()
	defer handlerRegistryMu.RUnlock()
	for name, factory := range handlerFactories {
		if !fn(name, factory) {
			return
		}
	}
}
