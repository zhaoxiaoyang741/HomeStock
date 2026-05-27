package outbound

import (
	"context"
	"fmt"
	"sync"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// Manager manages outbound endpoints and dispatches events to them.
type Manager struct {
	mu        sync.RWMutex
	endpoints map[string]Endpoint
	cfg       *config.OutboundConfig
}

// NewManager creates a Manager from the given config and populates endpoints.
func NewManager(cfg *config.OutboundConfig) *Manager {
	m := &Manager{
		endpoints: make(map[string]Endpoint),
		cfg:       cfg,
	}
	m.reload()
	return m
}

// Reload rebuilds endpoints from the current config.
func (m *Manager) Reload() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reload()
}

func (m *Manager) reload() {
	m.endpoints = make(map[string]Endpoint, len(m.cfg.Endpoints))
	for _, ep := range m.cfg.Endpoints {
		if ep.Enabled {
			m.endpoints[ep.Name] = newHTTPEndpoint(ep.Name, ep.URL)
		}
	}
}

// Send dispatches an event to all registered endpoints.
// Errors from individual endpoints are collected and returned as a single error.
func (m *Manager) Send(ctx context.Context, event OutboundEvent) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.endpoints) == 0 {
		return nil
	}

	var errs []error
	for name, ep := range m.endpoints {
		if err := ep.Send(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf("endpoint %q: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("send errors: %v", errs)
	}
	return nil
}

// Endpoints returns a snapshot of current endpoint configurations.
func (m *Manager) Endpoints() []config.EndpointConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]config.EndpointConfig, 0, len(m.cfg.Endpoints))
	result = append(result, m.cfg.Endpoints...)
	return result
}
