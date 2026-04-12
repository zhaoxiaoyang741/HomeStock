package service

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type MaterialService struct {
	db        *gorm.DB
	repo      repository.MaterialRepo
	auditRepo repository.AuditLogRepo
}

func NewMaterialService(db *gorm.DB, repo repository.MaterialRepo, audit repository.AuditLogRepo) *MaterialService {
	return &MaterialService{db: db, repo: repo, auditRepo: audit}
}

func (s *MaterialService) Create(ctx context.Context, actor Actor, name, spec, categoryID, defaultUnit string) (*model.Material, error) {
	m := &model.Material{
		TenantID:    actor.TenantID,
		Name:        strings.TrimSpace(name),
		Spec:        strings.TrimSpace(spec),
		CategoryID:  strings.TrimSpace(categoryID),
		DefaultUnit: strings.TrimSpace(defaultUnit),
	}
	if err := s.repo.Create(m); err != nil {
		return nil, err
	}
	created, err := s.repo.Get(m.ID, m.TenantID)
	if err != nil {
		return nil, err
	}
	_ = s.auditRepo.Create(&model.AuditLog{
		TenantID: actor.TenantID,
		UserName: actor.UserName,
		UserID:   actor.UserID,
		Channel:  actor.Channel,
		Action:   "create",
		EntityType: "material",
		EntityID:   created.ID,
		EntityName: created.Name,
	})
	return created, nil
}

func (s *MaterialService) List(ctx context.Context, f repository.MaterialFilter) ([]repository.MaterialSummary, error) {
	return s.repo.List(f)
}

func (s *MaterialService) GetDetail(ctx context.Context, id, tenantID string) (*repository.MaterialDetail, error) {
	return s.repo.GetDetail(id, tenantID)
}
