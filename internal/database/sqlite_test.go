package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestOpenAndMigrateSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data", "inventory.db")

	db, err := OpenAndMigrate(appconfig.DatabaseConfig{
		Driver: "sqlite",
		DSN:    dbPath,
	})
	if err != nil {
		t.Fatalf("OpenAndMigrate() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("sqlite file was not created: %v", err)
	}

	if !db.Migrator().HasTable(&model.Category{}) {
		t.Fatal("categories table was not created")
	}
	if !db.Migrator().HasTable(&model.Material{}) {
		t.Fatal("materials table was not created")
	}
	if !db.Migrator().HasTable(&model.StockLot{}) {
		t.Fatal("stock_lots table was not created")
	}
	if !db.Migrator().HasTable(&model.StockMovement{}) {
		t.Fatal("stock_movements table was not created")
	}

	var foreignKeys int
	if err := db.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil {
		t.Fatalf("PRAGMA foreign_keys error = %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("PRAGMA foreign_keys = %d", foreignKeys)
	}
}

func TestOpenRejectsUnsupportedDriver(t *testing.T) {
	_, err := Open(appconfig.DatabaseConfig{
		Driver: "mysql",
		DSN:    "mysql://db",
	})
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestOpenCreatesNestedSQLiteDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "db", "inventory.db")

	db, err := Open(appconfig.DatabaseConfig{
		DSN: dbPath,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if _, err := os.Stat(filepath.Dir(dbPath)); err != nil {
		t.Fatalf("sqlite directory was not created: %v", err)
	}
}

func TestCategoryAndMaterialDefaults(t *testing.T) {
	db, err := OpenAndMigrate(appconfig.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	})
	if err != nil {
		t.Fatalf("OpenAndMigrate() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	category := &model.Category{Name: "food"}
	if err := db.Create(category).Error; err != nil {
		t.Fatalf("Create(category) error = %v", err)
	}
	if len(category.ID) != 10 {
		t.Fatalf("category ID length = %d", len(category.ID))
	}
	if category.ID[:3] != "cat" {
		t.Fatalf("category ID prefix = %q", category.ID[:3])
	}
	if category.TenantID != "default" {
		t.Fatalf("category TenantID = %q", category.TenantID)
	}

	material := &model.Material{
		Name:       "eggs",
		CategoryID: category.ID,
	}
	if err := db.Create(material).Error; err != nil {
		t.Fatalf("Create(material) error = %v", err)
	}
	if material.ID == "" {
		t.Fatal("expected material ID to be generated")
	}
	if material.TenantID != "default" {
		t.Fatalf("TenantID = %q", material.TenantID)
	}
	if material.DefaultUnit != "件" {
		t.Fatalf("DefaultUnit = %q", material.DefaultUnit)
	}

	lot := &model.StockLot{
		MaterialID:     material.ID,
		QuantityOnHand: 12,
	}
	if err := db.Create(lot).Error; err != nil {
		t.Fatalf("Create(lot) error = %v", err)
	}
	if lot.ID == "" {
		t.Fatal("expected lot ID to be generated")
	}
	if lot.TenantID != "default" {
		t.Fatalf("lot TenantID = %q", lot.TenantID)
	}
	if lot.Unit != "件" {
		t.Fatalf("Unit = %q", lot.Unit)
	}
	if lot.ReceivedAt.IsZero() {
		t.Fatal("expected ReceivedAt to be populated")
	}
}
