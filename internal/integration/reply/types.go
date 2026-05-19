package reply

// RenderContext carries formatting preferences derived from the target channel.
type RenderContext struct {
	Fancy bool // true for Feishu (markdown tables), false for WeChat and others
}

// InventoryItemData holds data for a single inventory item in replies.
type InventoryItemData struct {
	Name     string
	Spec     string
	Quantity float64
	Unit     string
	Location string
	ExpireAt string // pre-formatted date string, e.g. "2026-06-01" or "无"
}

// ConsumeDetailData holds data for a single consumption line item.
type ConsumeDetailData struct {
	LotID             string
	ConsumedQuantity  float64
	RemainingQuantity float64
}

// ExpiryItemData holds data for a single expiring item.
type ExpiryItemData struct {
	Name     string
	Spec     string
	Quantity float64
	Unit     string
	Location string
	ExpireAt string // pre-formatted date string
}

// ConfirmCandidateData holds data for a name disambiguation candidate.
type ConfirmCandidateData struct {
	Label string // "A", "B", "C", etc.
	Name  string
	Spec  string
	Unit  string
}

// UpdateResultData holds data for a lot update reply.
type UpdateResultData struct {
	LotID    string
	Quantity float64
	Unit     string
}
