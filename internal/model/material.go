package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Material stores the reusable master data for a product or household supply.
type Material struct {
	ID          string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TenantID    string    `gorm:"index:idx_materials_tenant_name_spec,priority:1;type:varchar(36);not null;default:'default'" json:"tenant_id"`
	Name        string    `gorm:"index:idx_materials_tenant_name_spec,priority:2;type:varchar(255);not null" json:"name"`
	Spec        string    `gorm:"index:idx_materials_tenant_name_spec,priority:3;type:varchar(100);default:''" json:"spec"`
	CategoryID  string    `gorm:"type:varchar(10);index" json:"category_id"`
	Category    *Category `gorm:"foreignKey:CategoryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"category"`
	DefaultUnit string    `gorm:"type:varchar(20);not null;default:'件'" json:"default_unit"`
	Status      string    `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
	if material.Status == "" {
		material.Status = "active"
	}
	return nil
}
