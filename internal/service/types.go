package service

// Actor is the business-layer identity used across services.
type Actor struct {
	UserName string
	UserID   string
	Channel  string
	TenantID string
}
