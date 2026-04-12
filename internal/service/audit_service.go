package service

import (
	"context"

	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type AuditService struct {
	uow repository.UnitOfWork
}

func NewAuditService(uow repository.UnitOfWork) *AuditService { return &AuditService{uow: uow} }

func (s *AuditService) List(ctx context.Context, f repository.AuditLogFilter) (*repository.AuditLogPage, error) {
	return s.uow.Repos().AuditLogs().List(f)
}
