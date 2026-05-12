package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func openPostgres(cfg appconfig.DatabaseConfig) (*gorm.DB, error) {
	dsn := cfg.DSN
	if dsn == "" {
		return nil, fmt.Errorf("postgres dsn is required")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open postgres database %q: %w", dsn, err)
	}

	return db, nil
}
