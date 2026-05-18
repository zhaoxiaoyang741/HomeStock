package channel

import "errors"

// Sentinel errors for channel operations.
var (
	// ErrNotRunning is returned when attempting to use a stopped channel.
	ErrNotRunning = errors.New("channel: not running")
	// ErrRateLimit is returned when the channel is being rate-limited.
	ErrRateLimit = errors.New("channel: rate limited")
	// ErrTemporary is returned for transient failures that may succeed on retry.
	ErrTemporary = errors.New("channel: temporary error")
	// ErrSendFailed is returned when a send operation fails permanently.
	ErrSendFailed = errors.New("channel: send failed")
	// ErrNotSupported is returned when the channel does not support the requested operation.
	ErrNotSupported = errors.New("channel: not supported")
)

// IsTemporary returns true if err is a recoverable transient error.
func IsTemporary(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrRateLimit) || errors.Is(err, ErrTemporary)
}
