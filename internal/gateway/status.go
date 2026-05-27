package gateway

// GatewayStatus represents a snapshot of the Gateway runtime state.
type GatewayStatus struct {
	Uptime   string `json:"uptime"`
	CronJobs int    `json:"cron_jobs"`
}
