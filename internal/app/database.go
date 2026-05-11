package app

import (
	"database/sql"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func initDatabase(cfg config.DatabaseConfig) (*gorm.DB, *sql.DB, error) {
	db, err := database.OpenAndMigrate(cfg)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	return db, sqlDB, nil
}
