package wechat

import (
	"sync"
	"time"
)

// LoginStatus represents the current state of WeChat QR code login.
type LoginStatus int

const (
	LoginStatusIdle    LoginStatus = iota // no login in progress
	LoginStatusWaiting                    // waiting for user to scan QR code
	LoginStatusScanned                    // QR code scanned, waiting for confirmation
	LoginStatusSuccess                    // login successful, bot is active
	LoginStatusExpired                    // QR code expired
)

func (s LoginStatus) String() string {
	switch s {
	case LoginStatusIdle:
		return "idle"
	case LoginStatusWaiting:
		return "waiting"
	case LoginStatusScanned:
		return "scanned"
	case LoginStatusSuccess:
		return "success"
	case LoginStatusExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// LoginSession holds the current QR code login session state.
type LoginSession struct {
	mu       sync.RWMutex
	UUID     string      `json:"uuid"`
	QrURL    string      `json:"qr_url"`
	Status   LoginStatus `json:"status"`
	ExpireAt time.Time   `json:"expire_at"`
}

// NewLoginSession creates a new login session.
func NewLoginSession(uuid string) *LoginSession {
	return &LoginSession{
		UUID:     uuid,
		QrURL:    "https://login.weixin.qq.com/l/" + uuid,
		Status:   LoginStatusWaiting,
		ExpireAt: time.Now().Add(2 * time.Minute),
	}
}

// GetUUID returns the current QR code UUID.
func (s *LoginSession) GetUUID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.UUID
}

// GetQrURL returns the full QR code image URL.
func (s *LoginSession) GetQrURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.QrURL
}

// GetStatus returns the current login status.
func (s *LoginSession) GetStatus() LoginStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// SetStatus updates the login status.
func (s *LoginSession) SetStatus(status LoginStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
}

// IsExpired returns true if the QR code has expired.
func (s *LoginSession) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().After(s.ExpireAt)
}

// Snapshot returns a copy of the current session state.
func (s *LoginSession) Snapshot() LoginSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return LoginSession{
		UUID:     s.UUID,
		QrURL:    s.QrURL,
		Status:   s.Status,
		ExpireAt: s.ExpireAt,
	}
}
