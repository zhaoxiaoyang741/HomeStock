package gormrepo

import (
	"time"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type NotificationRepository struct{ db *gorm.DB }

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) CreateBatch(notifications []*model.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	return r.db.Create(&notifications).Error
}

func (r *NotificationRepository) ExistsForLotToday(lotID string, date time.Time) (bool, error) {
	var count int64
	dateStr := date.Format("2006-01-02")
	err := r.db.Model(&model.Notification{}).
		Where("lot_id = ? AND DATE(notify_at) = ?", lotID, dateStr).
		Count(&count).Error
	return count > 0, err
}

func (r *NotificationRepository) List(f repository.NotificationFilter) (*repository.NotificationPage, error) {
	page, pageSize := f.Page, f.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	q := r.db.Model(&model.Notification{})
	if f.LotID != "" {
		q = q.Where("lot_id = ?", f.LotID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var items []*model.Notification
	if err := q.Preload("Lot.Material").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, err
	}

	return &repository.NotificationPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (r *NotificationRepository) UpdateStatus(id, status string) error {
	return r.db.Model(&model.Notification{}).
		Where("id = ?", id).
		Update("status", status).Error
}
