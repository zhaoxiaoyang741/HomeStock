package cron

import (
	"context"
	"fmt"

	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// ExpiringStockNotifier checks for near-expiry stock lots and logs warnings.
type ExpiringStockNotifier struct {
	uow        repository.UnitOfWork
	expiryDays int
}

// NewExpiringStockNotifier creates a notifier that flags lots expiring within expiryDays.
func NewExpiringStockNotifier(uow repository.UnitOfWork, expiryDays int) *ExpiringStockNotifier {
	if expiryDays <= 0 {
		expiryDays = 7
	}
	return &ExpiringStockNotifier{
		uow:        uow,
		expiryDays: expiryDays,
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
	return nil
}
