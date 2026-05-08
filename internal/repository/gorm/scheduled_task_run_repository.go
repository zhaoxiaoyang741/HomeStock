package gormrepo

import (
	"strings"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type ScheduledTaskRunRepository struct{ db *gorm.DB }

func NewScheduledTaskRunRepository(db *gorm.DB) *ScheduledTaskRunRepository {
	return &ScheduledTaskRunRepository{db: db}
}

func (r *ScheduledTaskRunRepository) Create(run *model.ScheduledTaskRun) error {
	return r.db.Create(run).Error
}

func (r *ScheduledTaskRunRepository) Save(run *model.ScheduledTaskRun) error {
	return r.db.Save(run).Error
}

func (r *ScheduledTaskRunRepository) List(f repository.ScheduledTaskRunFilter) (*repository.ScheduledTaskRunPage, error) {
	page, pageSize := f.Page, f.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	q := r.db.Model(&model.ScheduledTaskRun{})
	if taskCode := strings.TrimSpace(f.TaskCode); taskCode != "" {
		q = q.Where("task_code = ?", taskCode)
	}
	if status := strings.TrimSpace(f.Status); status != "" {
		q = q.Where("status = ?", status)
	}
	if triggerSource := strings.TrimSpace(f.TriggerSource); triggerSource != "" {
		q = q.Where("trigger_source = ?", triggerSource)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var items []*model.ScheduledTaskRun
	if err := q.Order("started_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, err
	}

	return &repository.ScheduledTaskRunPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
