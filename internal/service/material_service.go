package service

import (
	"context"
	"strings"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type MaterialService struct { uow repository.UnitOfWork }

func NewMaterialService(uow repository.UnitOfWork) *MaterialService { return &MaterialService{uow: uow} }

func (s *MaterialService) Create(ctx context.Context, actor Actor, name, spec, categoryID, defaultUnit string) (*model.Material, error) {
	var created *model.Material
	err := s.uow.WithTx(ctx, func(r repository.Repos) error {
		m := &model.Material{
			TenantID:    actor.TenantID,
			Name:        strings.TrimSpace(name),
			Spec:        strings.TrimSpace(spec),
			CategoryID:  strings.TrimSpace(categoryID),
			DefaultUnit: strings.TrimSpace(defaultUnit),
		}
		if err := r.Materials().Create(m); err != nil { return err }
		var err2 error
		created, err2 = r.Materials().Get(m.ID, m.TenantID)
		if err2 != nil { return err2 }
		_ = r.AuditLogs().Create(&model.AuditLog{TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID, Channel: actor.Channel, Action: "create", EntityType: "material", EntityID: created.ID, EntityName: created.Name})
		return nil
	})
	if err != nil { return nil, err }
	return created, nil
}

func (s *MaterialService) List(ctx context.Context, f repository.MaterialFilter) ([]repository.MaterialSummary, error) {
	return s.uow.Repos().Materials().List(f)
}

func (s *MaterialService) GetDetail(ctx context.Context, id, tenantID string) (*repository.MaterialDetail, error) {
	return s.uow.Repos().Materials().GetDetail(id, tenantID)
}
