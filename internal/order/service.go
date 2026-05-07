package order

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidInput = errors.New("invalid order input")

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Order, error) {
	if input.CustomerID == "" || len(input.Items) == 0 {
		return Order{}, ErrInvalidInput
	}

	var total int64
	for _, item := range input.Items {
		if item.ProductID == "" || item.Quantity <= 0 || item.PriceKZT < 0 {
			return Order{}, ErrInvalidInput
		}
		total += int64(item.Quantity) * item.PriceKZT
	}

	now := time.Now().UTC()
	order := Order{
		ID:         fmt.Sprintf("ord-%d", now.UnixNano()),
		CustomerID: input.CustomerID,
		Items:      input.Items,
		Status:     StatusCreated,
		TotalKZT:   total,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return s.repo.Create(ctx, order)
}

func (s *Service) List(ctx context.Context) ([]Order, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id string) (Order, error) {
	if id == "" {
		return Order{}, ErrInvalidInput
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) UpdateStatus(ctx context.Context, id string, status Status) (Order, error) {
	if id == "" || !isAllowedStatus(status) {
		return Order{}, ErrInvalidInput
	}

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Order{}, err
	}

	order.Status = status
	order.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, order)
}

func isAllowedStatus(status Status) bool {
	switch status {
	case StatusCreated, StatusPaid, StatusShipped, StatusCompleted, StatusCanceled:
		return true
	default:
		return false
	}
}
