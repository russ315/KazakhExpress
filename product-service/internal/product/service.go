package product

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidInput = errors.New("invalid product input")

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Product, error) {
	if input.Name == "" || input.PriceKZT < 0 || input.Stock < 0 {
		return Product{}, ErrInvalidInput
	}

	now := time.Now().UTC()
	p := Product{
		ID:          fmt.Sprintf("prd-%d", now.UnixNano()),
		Name:        input.Name,
		Description: input.Description,
		PriceKZT:    input.PriceKZT,
		Stock:       input.Stock,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.repo.Create(ctx, p)
}

func (s *Service) List(ctx context.Context) ([]Product, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id string) (Product, error) {
	if id == "" {
		return Product{}, ErrInvalidInput
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) UpdateStock(ctx context.Context, id string, stock int) (Product, error) {
	if id == "" || stock < 0 {
		return Product{}, ErrInvalidInput
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Product{}, err
	}

	p.Stock = stock
	p.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, p)
}
