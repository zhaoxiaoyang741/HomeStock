package agent

// InboundMessage represents a message received from a messaging channel.
// Aligned with channel.InboundMessage to keep the agent package decoupled.
type InboundMessage struct {
	Channel    string // source channel name, e.g. "feishu"
	ChatID     string // conversation ID (group chat or direct message)
	SenderID   string // platform-specific user ID
	SenderName string // display name of the sender
	Text       string // message text content
	MediaType  string // "text", "voice", "image" (reserved)
	FileKey    string // media file key for ASR/OCR (reserved)
}

// OutboundMessage represents a message to be sent through a channel.
type OutboundMessage struct {
	Channel string // target channel name
	ChatID  string // conversation ID
	Text    string // message text content
}
