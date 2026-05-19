package channel

import "context"

// NotifyTargetProvider is an optional interface channels can implement
// to provide a default notification target ChatID for system notifications.
// Channels that participate in system-level notifications (e.g. expiry alerts)
// should store the ChatID of the user who last sent a message and return it here.
type NotifyTargetProvider interface {
	NotifyChatID() string
}

// Channel is the abstraction for a messaging channel.
//
// Each concrete channel (e.g. Feishu, Telegram) implements this interface
// and is managed by the ChannelManager.
type Channel interface {
	// Name returns the unique name of this channel, e.g. "feishu".
	Name() string

	// Start establishes the connection and begins receiving messages.
	// This should block until the connection is ready or an error occurs.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the channel connection.
	Stop(ctx context.Context) error

	// Send delivers an outbound message through this channel.
	Send(ctx context.Context, msg OutboundMessage) error

	// IsRunning returns whether the channel is currently connected and active.
	IsRunning() bool
}
