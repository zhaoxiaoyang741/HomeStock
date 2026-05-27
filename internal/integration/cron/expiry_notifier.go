package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/outbound"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// ExpiringStockNotifier checks for near-expiry stock lots and pushes
// expiry alerts via the outbound manager.
type ExpiringStockNotifier struct {
	uow        repository.UnitOfWork
	expiryDays int
	outboundMgr *outbound.Manager
}

// NewExpiringStockNotifier creates a notifier that flags lots expiring within expiryDays.
func NewExpiringStockNotifier(uow repository.UnitOfWork, expiryDays int, outboundMgr *outbound.Manager) *ExpiringStockNotifier {
	if expiryDays <= 0 {
		expiryDays = 7
	}
	return &ExpiringStockNotifier{
		uow:         uow,
		expiryDays:  expiryDays,
		outboundMgr: outboundMgr,
	}
}

func (n *ExpiringStockNotifier) Name() string { return "expiry_notifier" }

// ExpiryAlertPayload is the structured payload sent in an expiry alert event.
type ExpiryAlertPayload struct {
	Summary string          `json:"summary"`
	Count   int             `json:"count"`
	Lots    []ExpiryLotItem `json:"lots"`
}

// ExpiryLotItem describes a single near-expiry stock lot.
type ExpiryLotItem struct {
	Name     string  `json:"name"`
	Spec     string  `json:"spec"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
	Location string  `json:"location"`
	ExpireAt string  `json:"expire_at"`
}

func (n *ExpiringStockNotifier) Run(ctx context.Context) error {
	lots, err := n.uow.Repos().StockLots().List(repository.StockLotFilter{
		ExpiringSoon:     true,
		ExpiringSoonDays: n.expiryDays,
		ShowZeroStock:    false,
	})
	if err != nil {
		return fmt.Errorf("query expiring lots: %w", err)
	}

	if len(lots) == 0 {
		logger.InfoCF("cron", "expiry scan: no near-expiry lots found", nil)
		return nil
	}

	logger.WarnCF("cron", fmt.Sprintf("expiry scan: %d lot(s) expiring within %d days", len(lots), n.expiryDays), map[string]any{
		"count": len(lots),
	})
	for _, lot := range lots {
		expireStr := "unknown"
		if lot.ExpireAt != nil {
			expireStr = lot.ExpireAt.Format("2006-01-02")
		}
		name := lot.Material.Name
		spec := lot.Material.Spec
		nameStr := name
		if spec != "" {
			nameStr = name + " (" + spec + ")"
		}
		logger.WarnCF("cron", fmt.Sprintf("  %s: %.1f %s (expires %s, location: %s)",
			nameStr, lot.QuantityOnHand, lot.Unit, expireStr, lot.Location), nil)
	}

	// Check notification config (read fresh for hot-reload support).
	cfg := config.Get().Cron
	if !cfg.NotifyEnabled {
		logger.InfoCF("cron", "expiry notification skipped: notify_disabled", nil)
		return nil
	}
	if !isWithinTimeWindow(cfg.NotifyTimeStart, cfg.NotifyTimeEnd) {
		logger.InfoCF("cron", "expiry notification skipped: outside time window [%s, %s)",
			map[string]any{"window_start": cfg.NotifyTimeStart, "window_end": cfg.NotifyTimeEnd},
		)
		return nil
	}

	items := make([]ExpiryLotItem, 0, len(lots))
	for _, lot := range lots {
		expireStr := "未知"
		if lot.ExpireAt != nil {
			expireStr = lot.ExpireAt.Format("2006-01-02")
		}
		items = append(items, ExpiryLotItem{
			Name:     lot.Material.Name,
			Spec:     lot.Material.Spec,
			Quantity: lot.QuantityOnHand,
			Unit:     lot.Unit,
			Location: lot.Location,
			ExpireAt: expireStr,
		})
	}

	event := outbound.OutboundEvent{
		Type:      outbound.EventExpiryAlert,
		Timestamp: time.Now(),
		Payload: ExpiryAlertPayload{
			Summary: fmt.Sprintf("%d 个批次将在 %d 天内过期", len(items), n.expiryDays),
			Count:   len(items),
			Lots:    items,
		},
	}

	if err := n.outboundMgr.Send(ctx, event); err != nil {
		logger.ErrorCF("cron", "failed to send expiry alert via outbound",
			map[string]any{"error": err.Error()})
	} else {
		logger.InfoCF("cron", "expiry alert sent via outbound",
			map[string]any{"count": len(items)})
	}

	return nil
}

// isWithinTimeWindow checks whether the current time falls within [start, end).
// Supports cross-day windows (e.g. start="22:00", end="06:00").
// If start equals end, the window is considered always open.
func isWithinTimeWindow(start, end string) bool {
	now := time.Now()
	currentMinutes := now.Hour()*60 + now.Minute()

	startMinutes := parseHHMM(start)
	endMinutes := parseHHMM(end)

	if startMinutes == endMinutes {
		return true
	}
	if startMinutes < endMinutes {
		return currentMinutes >= startMinutes && currentMinutes < endMinutes
	}
	return currentMinutes >= startMinutes || currentMinutes < endMinutes
}

// parseHHMM parses a "HH:MM" string into total minutes since midnight.
// Returns 0 on parse failure.
func parseHHMM(s string) int {
	if len(s) < 5 {
		return 0
	}
	h, m := 0, 0
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0
	}
	return h*60 + m
}
