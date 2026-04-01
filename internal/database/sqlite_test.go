package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

	if !db.Migrator().HasTable(&model.Item{}) {
		t.Fatal("items table was not created")
	}

	if !db.Migrator().HasTable(&model.Notification{}) {
		t.Fatal("notifications table was not created")
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
		Driver: "postgres",
		DSN:    "postgres://db",
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

func TestItemAndNotificationDefaults(t *testing.T) {
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

	item := &model.Item{
		Name: "鸡蛋",
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("Create(item) error = %v", err)
	}

	if item.ID == "" {
		t.Fatal("expected item ID to be generated")
	}
	if item.TenantID != "default" {
		t.Fatalf("TenantID = %q", item.TenantID)
	}
	if item.Quantity != 1 {
		t.Fatalf("Quantity = %v", item.Quantity)
	}
	if item.Unit != "个" {
		t.Fatalf("Unit = %q", item.Unit)
	}
	if item.PurchasedAt.IsZero() {
		t.Fatal("expected PurchasedAt to be populated")
	}

	notification := &model.Notification{
		ItemID:   item.ID,
		NotifyAt: time.Now(),
	}
	if err := db.Create(notification).Error; err != nil {
		t.Fatalf("Create(notification) error = %v", err)
	}

	if notification.ID == "" {
		t.Fatal("expected notification ID to be generated")
	}
	if notification.Status != "pending" {
		t.Fatalf("Status = %q", notification.Status)
	}
	if notification.Channel != "feishu" {
		t.Fatalf("Channel = %q", notification.Channel)
	}
}
