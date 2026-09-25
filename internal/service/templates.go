package service

import (
	"context"

	"notifications/internal/domain"
)

type TemplateRepo interface {
	Get(ctx context.Context, typ string) (*domain.Template, error)
	List(ctx context.Context) ([]*domain.Template, error)
	Upsert(ctx context.Context, typ, body string) (*domain.Template, error)
}

type Templates struct {
	repo TemplateRepo
}

func NewTemplates(repo TemplateRepo) *Templates {
	return &Templates{repo: repo}
}

func (s *Templates) Get(ctx context.Context, typ string) (*domain.Template, error) {
	return s.repo.Get(ctx, typ)
}

func (s *Templates) List(ctx context.Context) ([]*domain.Template, error) {
	return s.repo.List(ctx)
}

func (s *Templates) Upsert(ctx context.Context, typ, body string) (*domain.Template, error) {
	if err := domain.ValidateTemplateType(typ); err != nil {
		return nil, err
	}
	return s.repo.Upsert(ctx, typ, body)
}
