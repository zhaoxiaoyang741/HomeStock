package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

// AuditLogFilter constrains list queries for audit logs.
type AuditLogFilter struct {
	TenantID  string
	Action    string
	Channel   string
	UserName  string // partial match via LIKE
	StartDate time.Time
	EndDate   time.Time
	Page      int // 1-based
	PageSize  int // default 20, max 100
}

// AuditLogPage is a paginated result of audit logs.
type AuditLogPage struct {
	Logs     []model.AuditLog
	Total    int64
	Page     int
	PageSize int
}

// AuditLogRepository stores and fetches audit log records.
type AuditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a repository backed by GORM.
func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create inserts a new audit log entry.
func (r *AuditLogRepository) Create(log *model.AuditLog) error {
	return r.db.Create(log).Error
}

// List returns a paginated, filtered slice of audit logs.
func (r *AuditLogRepository) List(f AuditLogFilter) (*AuditLogPage, error) {
	tenantID := normalizeTenantID(f.TenantID)
	page := f.Page
	if page < 1 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	q := r.db.Model(&model.AuditLog{}).Where("tenant_id = ?", tenantID)

	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.Channel != "" {
		q = q.Where("channel = ?", f.Channel)
	}
	if f.UserName != "" {
		q = q.Where("user_name LIKE ?", "%"+f.UserName+"%")
	}
	if !f.StartDate.IsZero() {
		q = q.Where("created_at >= ?", f.StartDate)
	}
	if !f.EndDate.IsZero() {
		q = q.Where("created_at <= ?", f.EndDate)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var logs []model.AuditLog
	offset := (page - 1) * pageSize
	if err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
		return nil, err
	}

	return &AuditLogPage{
		Logs:     logs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
