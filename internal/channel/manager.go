package channel

import (
	"context"
	"sync"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// Manager manages the lifecycle of all messaging channels.
//
// Channels are added via AddChannel or created from config through the factory
// registry. StartAll begins all channels, StopAll shuts them down gracefully.
type Manager struct {
	mu       sync.RWMutex
	channels map[string]Channel

	// Optional callback for outbound message routing.
	// Set by the app layer when MessageBus is available.
	outboundHandler func(ctx context.Context, msg OutboundMessage)

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager creates a Manager with the given channels.
func NewManager(channels ...Channel) *Manager {
	m := &Manager{
		channels: make(map[string]Channel),
	}
	for _, ch := range channels {
		if ch != nil {
			m.channels[ch.Name()] = ch
		}
	}
	return m
}

// AddChannel adds a channel to the manager. If a channel with the same name
// already exists, it is replaced.
func (m *Manager) AddChannel(ch Channel) {
	if ch == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[ch.Name()] = ch
}

// SetOutboundHandler sets the callback for routing outbound messages.
// The app layer should wire this to MessageBus.PublishOutbound.
func (m *Manager) SetOutboundHandler(fn func(ctx context.Context, msg OutboundMessage)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outboundHandler = fn
}

// StartAll starts all managed channels and begins outbound routing.
func (m *Manager) StartAll(ctx context.Context) error {
	logger.InfoCF("channel", "starting all channels", nil)

	m.mu.Lock()
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.mu.Unlock()

	m.mu.RLock()
	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.RUnlock()

	for _, ch := range channels {
		ch := ch
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			logger.InfoCF("channel", "starting channel", map[string]any{"name": ch.Name()})
			if err := ch.Start(m.ctx); err != nil {
				logger.ErrorCF("channel", "channel start failed", map[string]any{
					"name":  ch.Name(),
					"error": err.Error(),
				})
			}
		}()
	}

	return nil
}

// StopAll gracefully stops all managed channels.
func (m *Manager) StopAll(ctx context.Context) error {
	logger.InfoCF("channel", "stopping all channels", nil)

	m.mu.RLock()
	cancel := m.cancel
	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.RUnlock()

	if cancel != nil {
		cancel()
	}

	for _, ch := range channels {
		if err := ch.Stop(ctx); err != nil {
			logger.ErrorCF("channel", "channel stop failed", map[string]any{
				"name":  ch.Name(),
				"error": err.Error(),
			})
		}
	}

	m.wg.Wait()
	return nil
}

// GetChannel returns a channel by name.
func (m *Manager) GetChannel(name string) (Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.channels[name]
	return ch, ok
}

// GetEnabledChannels returns the names of all managed channels.
func (m *Manager) GetEnabledChannels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}
