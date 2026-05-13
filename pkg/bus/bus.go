package bus

import (
	"context"
	"sync"
)

// MessageBufferSize is the default channel buffer size for the MessageBus.
const MessageBufferSize = 64

// MessageBus decouples messaging channels from the AgentLoop.
//
// Channels publish inbound messages; the AgentLoop consumes them.
// The AgentLoop publishes outbound messages; the ChannelManager routes them.
//
// Close signals shutdown via a dedicated done channel — data channels
// are never closed so senders never panic.
type MessageBus struct {
	inbound  chan InboundMessage
	outbound chan OutboundMessage
	done     chan struct{}
	once     sync.Once
}

// NewMessageBus creates a MessageBus with the given buffer sizes.
func NewMessageBus(bufSize int) *MessageBus {
	if bufSize <= 0 {
		bufSize = MessageBufferSize
	}
	return &MessageBus{
		inbound:  make(chan InboundMessage, bufSize),
		outbound: make(chan OutboundMessage, bufSize),
		done:     make(chan struct{}),
	}
}

// PublishInbound sends a message from a channel into the bus.
// Returns ctx.Err() if the bus is closed or context is cancelled.
func (b *MessageBus) PublishInbound(ctx context.Context, msg InboundMessage) error {
	select {
	case <-b.done:
		return context.Canceled
	default:
	}
	select {
	case <-b.done:
		return context.Canceled
	case <-ctx.Done():
		return ctx.Err()
	case b.inbound <- msg:
		return nil
	}
}

// InboundChan returns the read-only inbound channel for the AgentLoop to consume.
func (b *MessageBus) InboundChan() <-chan InboundMessage {
	return b.inbound
}

// PublishOutbound sends a response message from the AgentLoop into the bus.
func (b *MessageBus) PublishOutbound(ctx context.Context, msg OutboundMessage) error {
	select {
	case <-b.done:
		return context.Canceled
	default:
	}
	select {
	case <-b.done:
		return context.Canceled
	case <-ctx.Done():
		return ctx.Err()
	case b.outbound <- msg:
		return nil
	}
}

// OutboundChan returns the read-only outbound channel for the ChannelManager to consume.
func (b *MessageBus) OutboundChan() <-chan OutboundMessage {
	return b.outbound
}

// Close signals that the bus is shut down. Publish calls will return Canceled.
// Safe to call multiple times. Data channels are not closed — consumers should
// use context cancellation to stop reading.
func (b *MessageBus) Close() {
	b.once.Do(func() {
		close(b.done)
	})
}

// Done returns a channel that's closed when Close is called.
func (b *MessageBus) Done() <-chan struct{} {
	return b.done
}
