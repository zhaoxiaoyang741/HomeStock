package gormrepo

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

type SystemSettingRepository struct{ db *gorm.DB }

func NewSystemSettingRepository(db *gorm.DB) *SystemSettingRepository {
	return &SystemSettingRepository{db: db}
}

func (r *SystemSettingRepository) GetByKey(key string) (*model.SystemSetting, error) {
	var setting model.SystemSetting
	if err := r.db.Model(&model.SystemSetting{}).Where("key = ?", strings.TrimSpace(key)).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *SystemSettingRepository) Upsert(setting *model.SystemSetting) error {
	key := strings.TrimSpace(setting.Key)
	if key == "" {
		key = "global"
	}
	existing, err := r.GetByKey(key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			setting.Key = key
			return r.db.Create(setting).Error
		}
		return err
	}
	setting.ID = existing.ID
	setting.Key = existing.Key
	setting.CreatedAt = existing.CreatedAt
	return r.db.Save(setting).Error
}
