package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"github.com/glebarez/sqlite"
	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// Open initializes a SQLite connection from config.
func Open(cfg appconfig.DatabaseConfig) (*gorm.DB, error) {
	driver := strings.TrimSpace(cfg.Driver)
	if driver == "" {
		driver = "sqlite"
	}

	if driver != "sqlite" {
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}

	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		return nil, fmt.Errorf("sqlite dsn is required")
	}

	if err := ensureSQLitePath(dsn); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database %q: %w", dsn, err)
	}

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}

	return db, nil
}

// OpenAndMigrate opens the database and applies the current schema.
func OpenAndMigrate(cfg appconfig.DatabaseConfig) (*gorm.DB, error) {
	db, err := Open(cfg)
	if err != nil {
		return nil, err
	}

	if err := Migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

// Migrate applies the current GORM schema.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(model.AutoMigrateModels()...); err != nil {
		return fmt.Errorf("auto-migrate database: %w", err)
	}

	return nil
}

func ensureSQLitePath(dsn string) error {
	if isSQLiteMemoryDSN(dsn) || strings.HasPrefix(dsn, "file:") {
		return nil
	}

	dir := filepath.Dir(dsn)
	if dir == "." || dir == "" {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create sqlite directory %q: %w", dir, err)
	}

	return nil
}

func isSQLiteMemoryDSN(dsn string) bool {
	return dsn == ":memory:" || strings.Contains(dsn, "mode=memory")
}
