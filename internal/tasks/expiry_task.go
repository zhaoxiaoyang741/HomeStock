package tasks

import (
	"context"

	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/internal/taskcenter"
)

func NewExpiryNotificationTaskDefinition(notificationSvc *service.NotificationService) taskcenter.TaskDefinition {
	return taskcenter.TaskDefinition{
		Code:              taskcenter.LegacySchedulerTaskCode,
		Name:              "Expiry Notification",
		Description:       "Scan expiring stock lots and deliver reminder notifications.",
		DefaultCronSpec:   "0 8 * * *",
		DefaultEnabled:    true,
		RunTimeoutSeconds: 300,
		Run: func(ctx context.Context, actor taskcenter.Actor) (taskcenter.TaskResult, error) {
			return notificationSvc.CheckAndNotify(ctx)
		},
	}
}
