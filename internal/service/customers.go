package service

import (
	"context"

	"notifications/internal/domain"
)

type CustomerRepo interface {
	Get(ctx context.Context, creditNumber string) (*domain.Customer, error)
	List(ctx context.Context) ([]*domain.Customer, error)
	Create(ctx context.Context, c domain.Customer) (*domain.Customer, error)
	Update(ctx context.Context, c domain.Customer) (*domain.Customer, error)
}

type Customers struct {
	repo CustomerRepo
}

func NewCustomers(repo CustomerRepo) *Customers {
	return &Customers{repo: repo}
}

func (s *Customers) Get(ctx context.Context, creditNumber string) (*domain.Customer, error) {
	return s.repo.Get(ctx, creditNumber)
}

func (s *Customers) List(ctx context.Context) ([]*domain.Customer, error) {
	return s.repo.List(ctx)
}

func (s *Customers) Create(ctx context.Context, c domain.Customer) (*domain.Customer, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, c)
}

func (s *Customers) Update(ctx context.Context, c domain.Customer) (*domain.Customer, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, c)
}
