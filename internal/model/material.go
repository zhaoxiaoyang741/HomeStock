package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Material stores the reusable master data for a product or household supply.
type Material struct {
	ID                    string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TenantID              string    `gorm:"index:idx_materials_tenant_name_spec,priority:1;type:varchar(36);not null;default:'default'" json:"tenant_id"`
	Name                  string    `gorm:"index:idx_materials_tenant_name_spec,priority:2;type:varchar(255);not null" json:"name"`
	Spec                  string    `gorm:"index:idx_materials_tenant_name_spec,priority:3;type:varchar(100);default:''" json:"spec"`
	CategoryID            string    `gorm:"type:varchar(10);index" json:"category_id"`
	Category              *Category `gorm:"foreignKey:CategoryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"category"`
	DefaultUnit           string    `gorm:"type:varchar(20);not null;default:'件'" json:"default_unit"`
	PurchaseUnit          string    `gorm:"type:varchar(20);default:''" json:"purchase_unit"`            // 采购单位，如"箱"
	StockUnit             string    `gorm:"type:varchar(20);default:''" json:"stock_unit"`               // 库存单位（最小单位），如"瓶"
	UnitFactor            float64   `gorm:"not null;default:1" json:"unit_factor"`                       // 采购→库存转换系数，如 1箱=12瓶 → factor=12
	DefaultBestBeforeDays int       `gorm:"not null;default:0" json:"default_best_before_days"`          // 默认保质期天数，0=当天（无设置），-1=永不过期
	MinStockAmount        float64   `gorm:"not null;default:0" json:"min_stock_amount"`                  // 最低库存阈值
	TrackByUnit           bool      `gorm:"not null;default:false" json:"track_by_unit"`                 // 逐件跟踪模式
	Status                string    `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (material *Material) BeforeCreate(*gorm.DB) error {
	if material.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		material.ID = id
	}
	if material.TenantID == "" {
		material.TenantID = "default"
	}
	material.Name = strings.TrimSpace(material.Name)
	material.Spec = strings.TrimSpace(material.Spec)
	if material.DefaultUnit == "" {
		material.DefaultUnit = "件"
	}
	// When StockUnit is not set, default to the same as DefaultUnit
	material.StockUnit = strings.TrimSpace(material.StockUnit)
	if material.StockUnit == "" {
		material.StockUnit = material.DefaultUnit
	}
	material.PurchaseUnit = strings.TrimSpace(material.PurchaseUnit)
	if material.UnitFactor <= 0 {
		material.UnitFactor = 1
	}
	if material.Status == "" {
		material.Status = "active"
	}
	return nil
}
