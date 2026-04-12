package gormrepo

import ("gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type AuditLogRepository struct{ db *gorm.DB }

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository { return &AuditLogRepository{db: db} }

func (r *AuditLogRepository) Create(log *model.AuditLog) error { return r.db.Create(log).Error }

func (r *AuditLogRepository) List(f repository.AuditLogFilter) (*repository.AuditLogPage, error) {
	tenantID := normalizeTenantID(f.TenantID)
	page := f.Page; if page < 1 { page = 1 }
	pageSize := f.PageSize; if pageSize < 1 { pageSize = 20 }; if pageSize > 100 { pageSize = 100 }
	q := r.db.Model(&model.AuditLog{}).Where("tenant_id = ?", tenantID)
	if f.Action != "" { q = q.Where("action = ?", f.Action) }
	if f.Channel != "" { q = q.Where("channel = ?", f.Channel) }
	if f.UserName != "" { q = q.Where("user_name LIKE ?", "%"+f.UserName+"%") }
	if !f.StartDate.IsZero() { q = q.Where("created_at >= ?", f.StartDate) }
	if !f.EndDate.IsZero() { q = q.Where("created_at <= ?", f.EndDate) }
	var total int64; if err := q.Count(&total).Error; err != nil { return nil, err }
	var logs []model.AuditLog
	offset := (page - 1) * pageSize
	if err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil { return nil, err }
	return &repository.AuditLogPage{ Logs: logs, Total: total, Page: page, PageSize: pageSize }, nil
}

