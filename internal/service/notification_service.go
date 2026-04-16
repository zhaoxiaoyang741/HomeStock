package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

// NotificationService checks expiring lots and dispatches Feishu notifications.
type NotificationService struct {
	uow         repository.UnitOfWork
	settingsSvc *SystemSettingsService
	httpClient  *http.Client
}

func NewNotificationService(uow repository.UnitOfWork, settingsSvc *SystemSettingsService) *NotificationService {
	return &NotificationService{
		uow:         uow,
		settingsSvc: settingsSvc,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

// CheckAndNotify queries lots expiring within remindDays, creates Notification records,
// and sends Feishu webhooks for each new notification.
func (s *NotificationService) CheckAndNotify(ctx context.Context) error {
	settings, err := s.settingsSvc.Get(ctx)
	if err != nil {
		return fmt.Errorf("get system settings: %w", err)
	}

	if !settings.Notify.Enabled || !settings.Notify.FeishuWebhookConfigured {
		return nil
	}

	remindDays := settings.Reminder.RemindDays
	if remindDays <= 0 {
		remindDays = 3
	}

	lots, err := s.uow.Repos().StockLots().List(repository.StockLotFilter{
		ExpiringSoon:     true,
		ExpiringSoonDays: remindDays,
		Status:           "active",
	})
	if err != nil {
		return fmt.Errorf("list expiring lots: %w", err)
	}

	today := time.Now()
	notifRepo := s.uow.Repos().Notifications()

	for _, lot := range lots {
		exists, err := notifRepo.ExistsForLotToday(lot.ID, today)
		if err != nil {
			return fmt.Errorf("check exists lot %s: %w", lot.ID, err)
		}
		if exists {
			continue
		}

		msg := buildExpiryMessage(lot, remindDays)
		n := &model.Notification{
			LotID:    lot.ID,
			NotifyAt: today,
			Status:   "pending",
			Channel:  "feishu",
			Message:  msg,
		}
		if err := notifRepo.CreateBatch([]*model.Notification{n}); err != nil {
			return fmt.Errorf("create notification for lot %s: %w", lot.ID, err)
		}

		// We need the actual webhook from stored settings (not masked).
		// Re-fetch via the raw stored settings through SystemSettingsService.
		webhook, err := s.settingsSvc.GetFeishuWebhook(ctx)
		if err != nil || webhook == "" {
			_ = notifRepo.UpdateStatus(n.ID, "failed")
			continue
		}

		sendErr := s.sendFeishu(webhook, msg)
		if sendErr != nil {
			_ = notifRepo.UpdateStatus(n.ID, "failed")
		} else {
			_ = notifRepo.UpdateStatus(n.ID, "sent")
		}
	}
	return nil
}

func buildExpiryMessage(lot model.StockLot, remindDays int) string {
	name := lot.ID
	if lot.Material.Name != "" {
		name = lot.Material.Name
		if lot.Material.Spec != "" {
			name += "(" + lot.Material.Spec + ")"
		}
	}

	expireDesc := "未知"
	if lot.ExpireAt != nil {
		daysLeft := int(time.Until(*lot.ExpireAt).Hours() / 24)
		if daysLeft < 0 {
			expireDesc = "已过期"
		} else if daysLeft == 0 {
			expireDesc = "今天到期"
		} else {
			expireDesc = fmt.Sprintf("%d天后到期（%s）", daysLeft, lot.ExpireAt.Format("2006-01-02"))
		}
	}

	return fmt.Sprintf("【库存到期提醒】%s — %s，剩余 %.2f %s%s",
		name, expireDesc, lot.QuantityOnHand, lot.Unit,
		locationHint(lot.Location),
	)
}

func locationHint(location string) string {
	if strings.TrimSpace(location) == "" {
		return ""
	}
	return "，位置：" + location
}

type feishuTextMessage struct {
	MsgType string            `json:"msg_type"`
	Content feishuTextContent `json:"content"`
}

type feishuTextContent struct {
	Text string `json:"text"`
}

func (s *NotificationService) sendFeishu(webhookURL, message string) error {
	payload := feishuTextMessage{
		MsgType: "text",
		Content: feishuTextContent{Text: message},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("feishu webhook returned status %d", resp.StatusCode)
	}
	return nil
}
