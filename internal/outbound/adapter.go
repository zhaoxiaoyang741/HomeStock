package outbound

import "context"

// Endpoint sends outbound events to an external system.
type Endpoint interface {
	// Name returns the endpoint identifier.
	Name() string
	// Send delivers an event. Implementations should honour context cancellation.
	Send(ctx context.Context, event OutboundEvent) error
}
