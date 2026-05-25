package gateway

// GatewayStatus represents a snapshot of the Gateway runtime state.
type GatewayStatus struct {
	Uptime      string          `json:"uptime"`
	Channels    []ChannelStatus `json:"channels"`
	ActiveModel string          `json:"active_model"`
	CronJobs    int             `json:"cron_jobs"`
}

// ChannelStatus represents the runtime state of a single channel.
type ChannelStatus struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
}
