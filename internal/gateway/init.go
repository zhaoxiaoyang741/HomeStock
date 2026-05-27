package gateway

import (
	"time"

	appcron "github.com/zhaoxiaoyang741/HomeStock/internal/integration/cron"
	"github.com/zhaoxiaoyang741/HomeStock/internal/outbound"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/cron"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// initCron creates the cron scheduler and registers the expiry stock notifier.
func initCron(uow *gormrepo.UnitOfWork, cfg config.CronConfig, outboundMgr *outbound.Manager) *cron.Service {
	svc := cron.New()

	if cfg.Enabled {
		interval, err := time.ParseDuration(cfg.ExpiryCheckPollInterval)
		if err != nil || interval <= 0 {
			interval = 6 * time.Hour
		}
		svc.Register(
			appcron.NewExpiringStockNotifier(uow, cfg.ExpiryCheckIntervalDays, outboundMgr),
			cron.ScheduleDef{Interval: interval},
		)
		logger.InfoCF("gateway", "cron: registered expiry_notifier", map[string]any{
			"interval":    interval.String(),
			"expiry_days": cfg.ExpiryCheckIntervalDays,
		})
	}

	return svc
}
