package channel

import (
	"context"
	"sync"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// DefaultWorkerQueueSize is the default buffer size for per-channel work queues.
const DefaultWorkerQueueSize = 32

// DefaultMaxRetries is the default number of send retries for transient errors.
const DefaultMaxRetries = 3

// channelWorker manages a per-channel goroutine for outbound message delivery
// with retry logic for transient errors.
type channelWorker struct {
	ch      Channel
	work    chan OutboundMessage
	retries int
	wg      sync.WaitGroup
	stop    chan struct{}
}

// Manager manages the lifecycle of all messaging channels.
//
// Channels are added via AddChannel or created from config through the factory
// registry. StartAll begins all channels and their per-channel worker goroutines;
// StopAll shuts them down gracefully.
type Manager struct {
	mu       sync.RWMutex
	channels map[string]Channel
	workers  map[string]*channelWorker

	// outboundHandler is set by the app layer when MessageBus is available.
	outboundHandler func(ctx context.Context, msg OutboundMessage)

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager creates a Manager with the given channels.
func NewManager(channels ...Channel) *Manager {
	m := &Manager{
		channels: make(map[string]Channel),
		workers:  make(map[string]*channelWorker),
	}
	for _, ch := range channels {
		if ch != nil {
			m.channels[ch.Name()] = ch
		}
	}
	return m
}

// AddChannel adds a channel to the manager. If a channel with the same name
// already exists, it is replaced. If the manager is already started, a worker
// goroutine is also started for the new channel.
func (m *Manager) AddChannel(ch Channel) {
	if ch == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[ch.Name()] = ch

	// If the manager is already running, start a worker for this channel.
	if m.ctx != nil {
		m.startWorkerLocked(ch)
	}
}

// RemoveChannel removes a channel from the manager by name. No-op if not found.
// If the channel has a running worker, the worker is stopped first.
func (m *Manager) RemoveChannel(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if w, ok := m.workers[name]; ok {
		close(w.stop)
		w.wg.Wait()
		delete(m.workers, name)
	}
	delete(m.channels, name)
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

// SetOutboundHandler sets the callback for routing outbound messages to external handlers.
func (m *Manager) SetOutboundHandler(fn func(ctx context.Context, msg OutboundMessage)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outboundHandler = fn
}

// StartAll starts all managed channels and their per-channel worker goroutines.
func (m *Manager) StartAll(ctx context.Context) error {
	logger.InfoCF("channel", "starting all channels", nil)

	m.mu.Lock()
	m.ctx, m.cancel = context.WithCancel(ctx)

	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.Unlock()

	// Start each channel in its own goroutine.
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

	// Start per-channel worker goroutines for outbound delivery.
	m.mu.RLock()
	for _, ch := range m.channels {
		m.startWorkerLocked(ch)
	}
	m.mu.RUnlock()

	return nil
}

// StopAll gracefully stops all channels and their workers.
func (m *Manager) StopAll(ctx context.Context) error {
	logger.InfoCF("channel", "stopping all channels", nil)

	m.mu.Lock()
	cancel := m.cancel

	// Stop all workers.
	for name, w := range m.workers {
		close(w.stop)
		delete(m.workers, name)
	}
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	// Wait for all workers to finish.
	m.mu.RLock()
	workers := make([]*channelWorker, 0, len(m.workers))
	for _, w := range m.workers {
		workers = append(workers, w)
	}
	m.mu.RUnlock()
	for _, w := range workers {
		w.wg.Wait()
	}

	// Stop all channels.
	m.mu.RLock()
	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.RUnlock()

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

// RouteOutbound delivers an outbound message through the appropriate channel's
// worker queue. Returns ErrNotRunning if the channel or its worker is not active.
func (m *Manager) RouteOutbound(ctx context.Context, msg OutboundMessage) error {
	m.mu.RLock()
	worker, ok := m.workers[msg.Channel]
	m.mu.RUnlock()
	if !ok {
		return ErrNotRunning
	}
	select {
	case worker.work <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// startWorkerLocked creates and starts a per-channel worker goroutine.
// Must be called with m.mu held (at least RLock).
func (m *Manager) startWorkerLocked(ch Channel) {
	if _, exists := m.workers[ch.Name()]; exists {
		return // already running
	}

	worker := &channelWorker{
		ch:      ch,
		work:    make(chan OutboundMessage, DefaultWorkerQueueSize),
		retries: DefaultMaxRetries,
		stop:    make(chan struct{}),
	}
	m.workers[ch.Name()] = worker

	worker.wg.Add(1)
	go m.runWorker(worker)
}

// runWorker is the per-channel outbound delivery loop.
// It reads from the work queue, applies capability-aware pre-processing,
// and sends with retry for transient errors.
func (m *Manager) runWorker(w *channelWorker) {
	defer w.wg.Done()

	for {
		select {
		case msg := <-w.work:
			m.deliverWithRetry(w, msg)
		case <-w.stop:
			// Drain remaining messages before exiting.
			for {
				select {
				case msg := <-w.work:
					m.deliverWithRetry(w, msg)
				default:
					return
				}
			}
		}
	}
}

// deliverWithRetry sends a single message with retry logic for transient errors.
// Before sending, it applies capability-aware pre-processing (typing indicator, etc.).
// After sending, it handles post-send actions (edit placeholder, etc.).
func (m *Manager) deliverWithRetry(w *channelWorker, msg OutboundMessage) {
	ch := w.ch

	// Pre-send: show typing indicator if supported.
	if tc, ok := ch.(TypingCapable); ok {
		_ = tc.ShowTyping(m.ctx, msg.ChatID)
	}

	// Determine message length limit for splitting.
	var maxLen int
	if mlp, ok := ch.(MessageLengthProvider); ok {
		maxLen = mlp.MaxMessageLength()
	}

	var err error
	for attempt := 0; attempt <= w.retries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-time.After(backoff):
			case <-w.stop:
				return
			}
		}

		// Choose the appropriate send method based on channel capabilities.
		if msg.MediaType != "" && msg.MediaType != "text" {
			if ms, ok := ch.(MediaSender); ok {
				err = ms.SendMedia(m.ctx, msg)
			} else {
				// Channel doesn't support media; skip.
				err = nil
				break
			}
		} else if maxLen > 0 && len(msg.Text) > maxLen {
			err = m.sendSplit(ch, m.ctx, msg, maxLen)
		} else {
			err = ch.Send(m.ctx, msg)
		}

		if err == nil || !IsTemporary(err) {
			break
		}
	}

	if err != nil {
		logger.ErrorCF("channel", "send failed after retries", map[string]any{
			"channel": ch.Name(),
			"chat_id": msg.ChatID,
			"error":   err.Error(),
		})
	}
}

// sendSplit splits a long message into chunks and sends each one sequentially.
func (m *Manager) sendSplit(ch Channel, ctx context.Context, msg OutboundMessage, maxLen int) error {
	text := msg.Text
	for len(text) > 0 {
		chunk := text
		if len(chunk) > maxLen {
			chunk = text[:maxLen]
		}
		chunkMsg := msg
		chunkMsg.Text = chunk
		if err := ch.Send(ctx, chunkMsg); err != nil {
			return err
		}
		text = text[len(chunk):]
	}
	return nil
}

// GetWebhookHandlers returns all channels that implement WebhookHandler,
// mapped by their webhook path. The caller (typically the HTTP server) is
// responsible for registering these handlers on the appropriate router.
func (m *Manager) GetWebhookHandlers() map[string]WebhookHandler {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]WebhookHandler)
	for _, ch := range m.channels {
		if wh, ok := ch.(WebhookHandler); ok {
			result[wh.WebhookPath()] = wh
		}
	}
	return result
}
