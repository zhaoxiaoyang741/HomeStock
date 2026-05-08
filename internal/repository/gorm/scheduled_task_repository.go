package gormrepo

import (
	"strings"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

type ScheduledTaskRepository struct{ db *gorm.DB }

func NewScheduledTaskRepository(db *gorm.DB) *ScheduledTaskRepository {
	return &ScheduledTaskRepository{db: db}
}

func (r *ScheduledTaskRepository) Create(task *model.ScheduledTask) error {
	return r.db.Create(task).Error
}

func (r *ScheduledTaskRepository) GetByCode(code string) (*model.ScheduledTask, error) {
	var task model.ScheduledTask
	if err := r.db.Where("code = ?", strings.TrimSpace(code)).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *ScheduledTaskRepository) List() ([]*model.ScheduledTask, error) {
	var items []*model.ScheduledTask
	if err := r.db.Order("code ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ScheduledTaskRepository) Save(task *model.ScheduledTask) error {
	return r.db.Save(task).Error
}
