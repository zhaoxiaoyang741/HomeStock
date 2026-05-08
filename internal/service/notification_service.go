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
	"github.com/zhaoxiaoyang741/HomeStock/internal/taskcenter"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// NotificationService checks expiring lots and dispatches Feishu notifications.
type NotificationService struct {
	uow         repository.UnitOfWork
	settingsSvc *SystemSettingsService
	httpClient  *http.Client
}

const expiryNotificationLogComponent = "expiry_notification"

func NewNotificationService(uow repository.UnitOfWork, settingsSvc *SystemSettingsService) *NotificationService {
	return &NotificationService{
		uow:         uow,
		settingsSvc: settingsSvc,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

// CheckAndNotify queries lots expiring within remindDays, creates Notification records,
// and sends Feishu webhooks for each new notification.
func (s *NotificationService) CheckAndNotify(ctx context.Context) (taskcenter.TaskResult, error) {
	result := expiryNotificationTaskResult{}
	startedAt := time.Now()

	logger.InfoCF(expiryNotificationLogComponent, "task execution started", nil)

	settings, err := s.settingsSvc.Get(ctx)
	if err != nil {
		logger.ErrorCF(expiryNotificationLogComponent, "failed to load system settings", map[string]any{
			"error": err.Error(),
		})
		return result.toTaskResult(), fmt.Errorf("get system settings: %w", err)
	}

	logger.InfoCF(expiryNotificationLogComponent, "task settings loaded", map[string]any{
		"remind_days":        settings.Reminder.RemindDays,
		"notify_enabled":     settings.Notify.Enabled,
		"webhook_configured": settings.Notify.FeishuWebhookConfigured,
	})

	if !settings.Notify.Enabled || !settings.Notify.FeishuWebhookConfigured {
		result.Summary = "notifications are disabled or webhook is not configured"
		logger.WarnCF(expiryNotificationLogComponent, "task execution skipped because notifications are unavailable", map[string]any{
			"notify_enabled":     settings.Notify.Enabled,
			"webhook_configured": settings.Notify.FeishuWebhookConfigured,
			"summary":            result.Summary,
		})
		return result.toTaskResult(), nil
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
		logger.ErrorCF(expiryNotificationLogComponent, "failed to query expiring lots", map[string]any{
			"remind_days": remindDays,
			"error":       err.Error(),
		})
		return result.toTaskResult(), fmt.Errorf("list expiring lots: %w", err)
	}
	result.ScannedCount = len(lots)

	logger.InfoCF(expiryNotificationLogComponent, "expiring lots loaded", map[string]any{
		"remind_days": remindDays,
		"lot_count":   len(lots),
	})

	today := time.Now()
	notifRepo := s.uow.Repos().Notifications()

	for _, lot := range lots {
		exists, err := notifRepo.ExistsForLotToday(lot.ID, today)
		if err != nil {
			logger.ErrorCF(expiryNotificationLogComponent, "failed to check existing notification", lotLogFields(lot, map[string]any{
				"error": err.Error(),
			}))
			return result.toTaskResult(), fmt.Errorf("check exists lot %s: %w", lot.ID, err)
		}
		if exists {
			result.SkippedExistingCount++
			logger.InfoCF(expiryNotificationLogComponent, "skipped lot because notification already exists today", lotLogFields(lot, map[string]any{
				"skipped_existing_count": result.SkippedExistingCount,
			}))
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
			logger.ErrorCF(expiryNotificationLogComponent, "failed to create notification record", lotLogFields(lot, map[string]any{
				"error": err.Error(),
			}))
			return result.toTaskResult(), fmt.Errorf("create notification for lot %s: %w", lot.ID, err)
		}

		logger.InfoCF(expiryNotificationLogComponent, "notification record created", lotLogFields(lot, map[string]any{
			"notification_id": n.ID,
			"channel":         n.Channel,
		}))

		// We need the actual webhook from stored settings (not masked).
		// Re-fetch via the raw stored settings through SystemSettingsService.
		webhook, err := s.settingsSvc.GetFeishuWebhook(ctx)
		if err != nil || webhook == "" {
			if updateErr := notifRepo.UpdateStatus(n.ID, "failed"); updateErr != nil {
				logger.WarnCF(expiryNotificationLogComponent, "failed to update notification status after missing webhook", lotLogFields(lot, map[string]any{
					"notification_id": n.ID,
					"status":          "failed",
					"error":           updateErr.Error(),
				}))
			}
			result.FailedCount++

			fields := lotLogFields(lot, map[string]any{
				"notification_id": n.ID,
				"failed_count":    result.FailedCount,
			})
			if err != nil {
				fields["error"] = err.Error()
			}
			logger.WarnCF(expiryNotificationLogComponent, "webhook unavailable for notification delivery", fields)
			continue
		}

		sendErr := s.sendFeishu(webhook, msg)
		if sendErr != nil {
			if updateErr := notifRepo.UpdateStatus(n.ID, "failed"); updateErr != nil {
				logger.WarnCF(expiryNotificationLogComponent, "failed to update notification status after send failure", lotLogFields(lot, map[string]any{
					"notification_id": n.ID,
					"status":          "failed",
					"error":           updateErr.Error(),
				}))
			}
			result.FailedCount++
			logger.WarnCF(expiryNotificationLogComponent, "notification delivery failed", lotLogFields(lot, map[string]any{
				"notification_id": n.ID,
				"failed_count":    result.FailedCount,
				"error":           sendErr.Error(),
			}))
			continue
		}

		if updateErr := notifRepo.UpdateStatus(n.ID, "sent"); updateErr != nil {
			logger.WarnCF(expiryNotificationLogComponent, "failed to update notification status after send success", lotLogFields(lot, map[string]any{
				"notification_id": n.ID,
				"status":          "sent",
				"error":           updateErr.Error(),
			}))
		}
		result.SentCount++
		logger.InfoCF(expiryNotificationLogComponent, "notification delivered", lotLogFields(lot, map[string]any{
			"notification_id": n.ID,
			"sent_count":      result.SentCount,
		}))
	}

	finalResult := result.toTaskResult()
	logger.InfoCF(expiryNotificationLogComponent, "task execution finished", map[string]any{
		"duration_ms":            time.Since(startedAt).Milliseconds(),
		"scanned_count":          result.ScannedCount,
		"sent_count":             result.SentCount,
		"failed_count":           result.FailedCount,
		"skipped_existing_count": result.SkippedExistingCount,
		"summary":                finalResult.Summary,
	})
	return finalResult, nil
}

type expiryNotificationTaskResult struct {
	ScannedCount         int    `json:"scanned_count"`
	SentCount            int    `json:"sent_count"`
	FailedCount          int    `json:"failed_count"`
	SkippedExistingCount int    `json:"skipped_existing_count"`
	Summary              string `json:"summary"`
}

func (r expiryNotificationTaskResult) toTaskResult() taskcenter.TaskResult {
	summary := strings.TrimSpace(r.Summary)
	if summary == "" {
		summary = fmt.Sprintf(
			"scanned=%d sent=%d failed=%d skipped_existing=%d",
			r.ScannedCount,
			r.SentCount,
			r.FailedCount,
			r.SkippedExistingCount,
		)
	}
	r.Summary = summary
	return taskcenter.TaskResult{
		Summary: summary,
		Payload: r,
	}
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

func lotLogFields(lot model.StockLot, extra map[string]any) map[string]any {
	fields := map[string]any{
		"lot_id":           lot.ID,
		"material_id":      lot.MaterialID,
		"quantity_on_hand": lot.QuantityOnHand,
		"unit":             lot.Unit,
		"location":         strings.TrimSpace(lot.Location),
		"status":           lot.Status,
	}
	if lot.Material != nil {
		if strings.TrimSpace(lot.Material.Name) != "" {
			fields["material_name"] = lot.Material.Name
		}
		if strings.TrimSpace(lot.Material.Spec) != "" {
			fields["material_spec"] = lot.Material.Spec
		}
	}
	if lot.ExpireAt != nil {
		fields["expire_at"] = lot.ExpireAt.Format(time.RFC3339)
	}
	for key, value := range extra {
		fields[key] = value
	}
	return fields
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
