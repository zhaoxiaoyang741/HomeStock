package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/reply"
	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/channel"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// ExpiringStockNotifier checks for near-expiry stock lots and sends notifications
// via enabled channels within the configured time window.
type ExpiringStockNotifier struct {
	uow        repository.UnitOfWork
	expiryDays int
	channelMgr *channel.Manager
}

// NewExpiringStockNotifier creates a notifier that flags lots expiring within expiryDays.
func NewExpiringStockNotifier(uow repository.UnitOfWork, expiryDays int, channelMgr *channel.Manager) *ExpiringStockNotifier {
	if expiryDays <= 0 {
		expiryDays = 7
	}
	return &ExpiringStockNotifier{
		uow:        uow,
		expiryDays: expiryDays,
		channelMgr: channelMgr,
	}
}

func (n *ExpiringStockNotifier) Name() string { return "expiry_notifier" }

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

	n.sendNotifications(ctx, lots)
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
		// Same value: no restriction.
		return true
	}
	if startMinutes < endMinutes {
		// Normal window: same day.
		return currentMinutes >= startMinutes && currentMinutes < endMinutes
	}
	// Cross-day window: e.g. 22:00-06:00.
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

// buildExpiryMessage creates a formatted notification message from expiring lots.
// channelName is used to select the appropriate formatting (plain text vs markdown table).
func buildExpiryMessage(lots []model.StockLot, channelName string) string {
	items := make([]reply.ExpiryItemData, 0, len(lots))
	for _, lot := range lots {
		expireStr := "未知"
		if lot.ExpireAt != nil {
			expireStr = lot.ExpireAt.Format("2006-01-02")
		}
		items = append(items, reply.ExpiryItemData{
			Name:     lot.Material.Name,
			Spec:     lot.Material.Spec,
			Quantity: lot.QuantityOnHand,
			Unit:     lot.Unit,
			Location: lot.Location,
			ExpireAt: expireStr,
		})
	}
	rc := reply.ForChannel(channelName)
	return reply.ExpiryWarning(rc, items)
}

// sendNotifications builds per-channel expiry messages and sends them.
func (n *ExpiringStockNotifier) sendNotifications(ctx context.Context, lots []model.StockLot) {
	for _, name := range n.channelMgr.GetEnabledChannels() {
		ch, ok := n.channelMgr.GetChannel(name)
		if !ok {
			continue
		}
		provider, ok := ch.(channel.NotifyTargetProvider)
		if !ok {
			logger.DebugCF("cron", "channel does not implement NotifyTargetProvider, skipping",
				map[string]any{"channel": name})
			continue
		}
		chatID := provider.NotifyChatID()
		if chatID == "" {
			logger.WarnCF("cron", "channel has no notification target (no messages received yet), skipping",
				map[string]any{"channel": name})
			continue
		}
		if err := n.channelMgr.RouteOutbound(ctx, channel.OutboundMessage{
			Channel: name,
			ChatID:  chatID,
			Text:    buildExpiryMessage(lots, name),
		}); err != nil {
			logger.ErrorCF("cron", "failed to send expiry notification",
				map[string]any{"channel": name, "chat_id": chatID, "error": err.Error()})
		} else {
			logger.InfoCF("cron", "expiry notification sent",
				map[string]any{"channel": name, "chat_id": chatID})
		}
	}
}
