package outbound

import "time"

// EventType categorises an outbound event.
type EventType string

const (
	// EventExpiryAlert is sent when a stock lot is nearing its expiry date.
	EventExpiryAlert EventType = "expiry_alert"
)

// OutboundEvent is the payload delivered to every registered endpoint.
type OutboundEvent struct {
	Type      EventType            `json:"type"`
	Timestamp time.Time            `json:"timestamp"`
	Payload   interface{}          `json:"payload"`
	Metadata  map[string]string    `json:"metadata,omitempty"`
}
