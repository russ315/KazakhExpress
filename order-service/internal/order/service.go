package order

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

var ErrInvalidInput = errors.New("invalid order input")

type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, event Event) error
	PublishOrderCancelled(ctx context.Context, event Event) error
	PublishOrderCompleted(ctx context.Context, event Event) error
}

type StatusCache interface {
	SetStatus(ctx context.Context, orderID string, status Status) error
}

type Service struct {
	repo      Repository
	publisher EventPublisher
	cache     StatusCache
}

func NewService(repo Repository, publisher EventPublisher, cache StatusCache) *Service {
	return &Service{repo: repo, publisher: publisher, cache: cache}
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

	created, err := s.repo.Create(ctx, order)
	if err != nil {
		return Order{}, err
	}
	s.cacheStatus(ctx, created)
	s.publish(ctx, "order.created", func() error {
		return s.publisher.PublishOrderCreated(ctx, eventFromOrder(created, ""))
	})
	return created, nil
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

	updated, err := s.repo.UpdateStatus(ctx, id, order.Status, status, "manual status update")
	if err != nil {
		return Order{}, err
	}
	s.cacheStatus(ctx, updated)
	if updated.Status == StatusCompleted {
		s.publish(ctx, "order.completed", func() error {
			return s.publisher.PublishOrderCompleted(ctx, eventFromOrder(updated, ""))
		})
	}

	return updated, nil
}

func (s *Service) Cancel(ctx context.Context, id string, reason string) (Order, error) {
	if id == "" {
		return Order{}, ErrInvalidInput
	}
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Order{}, err
	}
	if order.Status == StatusCompleted || order.Status == StatusCanceled {
		return Order{}, ErrInvalidInput
	}

	updated, err := s.repo.UpdateStatus(ctx, id, order.Status, StatusCanceled, reason)
	if err != nil {
		return Order{}, err
	}
	s.cacheStatus(ctx, updated)
	s.publish(ctx, "order.cancelled", func() error {
		return s.publisher.PublishOrderCancelled(ctx, eventFromOrder(updated, reason))
	})
	return updated, nil
}

func (s *Service) HandlePaymentSucceeded(ctx context.Context, event PaymentEvent) error {
	if event.OrderID == "" {
		return ErrInvalidInput
	}
	order, err := s.repo.GetByID(ctx, event.OrderID)
	if err != nil {
		return err
	}
	updated, err := s.repo.UpdateStatus(ctx, event.OrderID, order.Status, StatusPaid, "payment.succeeded")
	if err != nil {
		return err
	}
	s.cacheStatus(ctx, updated)
	return nil
}

func (s *Service) HandlePaymentFailed(ctx context.Context, event PaymentEvent) error {
	if event.OrderID == "" {
		return ErrInvalidInput
	}
	order, err := s.repo.GetByID(ctx, event.OrderID)
	if err != nil {
		return err
	}
	updated, err := s.repo.UpdateStatus(ctx, event.OrderID, order.Status, StatusPaymentFailed, event.Reason)
	if err != nil {
		return err
	}
	s.cacheStatus(ctx, updated)
	return nil
}

func (s *Service) HandleStockReserved(ctx context.Context, event StockReservedEvent) error {
	if event.OrderID == "" {
		return ErrInvalidInput
	}
	_, err := s.repo.GetByID(ctx, event.OrderID)
	return err
}

func isAllowedStatus(status Status) bool {
	switch status {
	case StatusCreated, StatusPaid, StatusPaymentFailed, StatusShipped, StatusCompleted, StatusCanceled:
		return true
	default:
		return false
	}
}

func eventFromOrder(order Order, reason string) Event {
	return Event{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
		Status:     order.Status,
		TotalKZT:   order.TotalKZT,
		Reason:     reason,
		OccurredAt: time.Now().UTC(),
	}
}

func (s *Service) cacheStatus(ctx context.Context, order Order) {
	if s.cache == nil {
		return
	}
	if err := s.cache.SetStatus(ctx, order.ID, order.Status); err != nil {
		log.Printf("cache order status %s: %v", order.ID, err)
	}
}

func (s *Service) publish(ctx context.Context, name string, operation func() error) {
	if s.publisher == nil {
		return
	}
	if err := operation(); err != nil {
		log.Printf("publish %s: %v", name, err)
	}
}
