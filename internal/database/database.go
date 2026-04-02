package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

// InitBaseData seeds required default records. It is idempotent: existing rows are left untouched.
func InitBaseData(db *gorm.DB) error {
	defaultCategory := model.Category{
		ID:        "cat1234567",
		TenantID:  "default",
		Name:      "默认分类",
		CreatedAt: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC),
	}

	var count int64
	if err := db.Model(&model.Category{}).Where("id = ?", defaultCategory.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	err := db.Create(&defaultCategory).Error

	if err != nil {
		return fmt.Errorf("Init database base data error %d", err.Error())
	}

	return nil
}
