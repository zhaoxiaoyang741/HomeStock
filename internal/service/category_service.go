package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type CategoryService struct { uow repository.UnitOfWork }

func NewCategoryService(uow repository.UnitOfWork) *CategoryService { return &CategoryService{uow: uow} }

func (s *CategoryService) Create(ctx context.Context, actor Actor, name string) (*model.Category, error) {
	var created *model.Category
	name = strings.TrimSpace(name)
		err := s.uow.WithTx(ctx, func(r repository.Repos) error {
		cat := &model.Category{TenantID: actor.TenantID, Name: name}
		if err := r.Categories().Create(cat); err != nil { return err }
		var err2 error
		created, err2 = r.Categories().Get(cat.ID, cat.TenantID)
		if err2 != nil { return err2 }
		_ = r.AuditLogs().Create(&model.AuditLog{TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID, Channel: actor.Channel, Action: "create", EntityType: "category", EntityID: created.ID, EntityName: created.Name, ChangesDetail: mustJSON(changes{After: created})})
		return nil
	})
	if err != nil { return nil, err }
	return created, nil
}

func (s *CategoryService) List(ctx context.Context, tenantID string) ([]model.Category, error) {
	return s.uow.Repos().Categories().List(tenantID)
}

func (s *CategoryService) Get(ctx context.Context, id, tenantID string) (*model.Category, error) {
	return s.uow.Repos().Categories().Get(id, tenantID)
}

func (s *CategoryService) Update(ctx context.Context, actor Actor, id, tenantID string, name *string) (*model.Category, error) {
	var updated *model.Category
	err := s.uow.WithTx(ctx, func(r repository.Repos) error {
		cat, err := r.Categories().Get(id, tenantID)
		if err != nil { return err }
		before := map[string]string{"name": cat.Name}
		if name != nil { n := strings.TrimSpace(*name); if n != "" { cat.Name = n } }
		if err := r.Categories().Update(cat); err != nil { return err }
		var err2 error
		updated, err2 = r.Categories().Get(cat.ID, cat.TenantID)
		if err2 != nil { return err2 }
		_ = r.AuditLogs().Create(&model.AuditLog{TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID, Channel: actor.Channel, Action: "update", EntityType: "category", EntityID: updated.ID, EntityName: updated.Name, ChangesDetail: mustJSON(changes{Before: before, After: map[string]string{"name": updated.Name}})})
		return nil
	})
	if err != nil { return nil, err }
	return updated, nil
}

func (s *CategoryService) Delete(ctx context.Context, actor Actor, id, tenantID string) error {
	return s.uow.WithTx(ctx, func(r repository.Repos) error {
		cat, err := r.Categories().Get(id, tenantID)
		if err != nil { return err }
		if err := r.Categories().Delete(id, tenantID); err != nil { return err }
		_ = r.AuditLogs().Create(&model.AuditLog{TenantID: actor.TenantID, UserName: actor.UserName, UserID: actor.UserID, Channel: actor.Channel, Action: "delete", EntityType: "category", EntityID: cat.ID, EntityName: cat.Name, ChangesDetail: mustJSON(changes{Before: cat})})
		return nil
	})
}

type changes struct { Before any `json:"before,omitempty"`; After any `json:"after,omitempty"` }

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

