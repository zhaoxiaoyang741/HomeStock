package model

// AutoMigrateModels returns the domain models managed by GORM migrations.
func AutoMigrateModels() []any {
	return []any{
		&Category{},
		&Material{},
		&StockLot{},
		&StockMovement{},
		&Notification{},
		&ScheduledTask{},
		&ScheduledTaskRun{},
		&AuditLog{},
		&SystemSetting{},
	}
}
