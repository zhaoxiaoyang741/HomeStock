package channel

// InboundMessage represents a message received from a channel.
type InboundMessage struct {
	// Channel is the source channel name, e.g. "feishu".
	Channel string

	// ChatID is the conversation ID (group chat or direct message).
	ChatID string

	// SenderID is the platform-specific user ID of the sender.
	SenderID string

	// SenderName is the display name of the sender, if available.
	SenderName string

	// Text is the message text content.
	Text string

	// MediaType indicates the message type: "text", "voice", "image", etc.
	// Reserved for future use.
	MediaType string

	// FileKey is the platform file key for media messages, used for ASR/OCR.
	// Reserved for future use.
	FileKey string

	// Raw holds the platform-specific raw event data.
	Raw any
}

// OutboundMessage represents a message to be sent through a channel.
type OutboundMessage struct {
	// Channel is the target channel name, e.g. "feishu".
	Channel string

	// ChatID is the conversation ID to send the message to.
	ChatID string

	// Text is the message text content.
	Text string
}
