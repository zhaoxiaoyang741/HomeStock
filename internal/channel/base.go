package channel

import (
	"context"
	"sync"
)

// BaseChannel provides common infrastructure for concrete channel implementations.
//
// Embed BaseChannel in your channel struct and call InitBase during construction
// to get shared running-state tracking, access control, and message routing.
type BaseChannel struct {
	name    string
	running bool
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc

	// allowList restricts which sender IDs can interact with this channel.
	// An empty or nil slice means all senders are allowed.
	allowList []string

	// onInboundMessage is the callback set by ChannelManager to publish
	// inbound messages to the MessageBus.
	onInboundMessage func(ctx context.Context, msg InboundMessage)
}

// InitBase initializes the BaseChannel with a name and optional allow list.
// Must be called by embedding structs during construction.
func (b *BaseChannel) InitBase(name string, allowList []string) {
	b.name = name
	if allowList == nil {
		allowList = []string{}
	}
	b.allowList = allowList
}

// Name returns the channel name.
func (b *BaseChannel) Name() string { return b.name }

// SetInboundHandler sets the callback for routing inbound messages.
// Called by ChannelManager during channel setup.
func (b *BaseChannel) SetInboundHandler(fn func(ctx context.Context, msg InboundMessage)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onInboundMessage = fn
}

// HandleInbound checks access control and forwards the message to the handler.
// Concrete channels should call this when they receive a message.
func (b *BaseChannel) HandleInbound(ctx context.Context, msg InboundMessage) {
	if !b.IsAllowed(msg.SenderID) {
		return
	}
	b.mu.RLock()
	fn := b.onInboundMessage
	b.mu.RUnlock()
	if fn != nil {
		fn(ctx, msg)
	}
}

// IsAllowed checks the allow list. An empty allow list permits all senders.
func (b *BaseChannel) IsAllowed(senderID string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.allowList) == 0 {
		return true
	}
	for _, id := range b.allowList {
		if id == senderID {
			return true
		}
	}
	return false
}

// BaseStart creates the internal context and marks the channel as running.
// Concrete channels should call this at the beginning of their Start method.
func (b *BaseChannel) BaseStart(ctx context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ctx, b.cancel = context.WithCancel(ctx)
	b.running = true
}

// BaseStop cancels the internal context and marks the channel as stopped.
// Concrete channels should defer (or call) this at the end of their Stop method.
func (b *BaseChannel) BaseStop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cancel != nil {
		b.cancel()
	}
	b.running = false
}

// IsRunning returns whether the channel is currently active.
func (b *BaseChannel) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// Context returns the internal context created by BaseStart.
func (b *BaseChannel) Context() context.Context {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.ctx
}
