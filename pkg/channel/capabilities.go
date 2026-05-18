package channel

import (
	"context"
	"net/http"
)

// InboundHandlerSetter is implemented by channels that can receive
// and process inbound messages (e.g., chat platforms).
// The manager uses this to wire the message bus callback during setup.
type InboundHandlerSetter interface {
	SetInboundHandler(fn func(ctx context.Context, msg InboundMessage))
}

// TypingCapable channels can show a typing/thinking indicator.
type TypingCapable interface {
	ShowTyping(ctx context.Context, chatID string) error
}

// MediaSender channels can send media attachments (images, audio, files).
type MediaSender interface {
	SendMedia(ctx context.Context, msg OutboundMessage) error
}

// WebhookHandler channels receive incoming messages via HTTP webhook.
// Implementations provide the path and handle requests as an http.Handler.
type WebhookHandler interface {
	// WebhookPath returns the URL path for this channel's webhook,
	// e.g. "/api/v1/feishu/webhook".
	WebhookPath() string
	http.Handler
}

// HealthChecker channels expose a detailed health status.
type HealthChecker interface {
	Health(ctx context.Context) map[string]any
}

// Configurable channels support runtime reconfiguration from a generic config map.
type Configurable interface {
	Reconfigure(ctx context.Context, cfg map[string]any) error
}

// MessageLengthProvider channels advertise their maximum supported message length.
type MessageLengthProvider interface {
	MaxMessageLength() int
}
