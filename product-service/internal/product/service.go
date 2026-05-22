package product

import (
	"context"
	"fmt"
	"time"
)

const defaultCacheTTL = 5 * time.Minute

type Service struct {
	repo      Repository
	cache     CacheService
	publisher EventPublisher
}

func NewService(repo Repository, cache CacheService, publisher EventPublisher) *Service {
	if cache == nil {
		cache = NoopCache{}
	}
	if publisher == nil {
		publisher = NoopPublisher{}
	}
	return &Service{repo: repo, cache: cache, publisher: publisher}
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

	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return Product{}, err
	}

	s.invalidateProductCache(ctx, created.ID)
	_ = s.publisher.PublishProductCreated(ctx, ProductEvent{
		ProductID: created.ID,
		Name:      created.Name,
		Timestamp: now,
	})

	return created, nil
}

func (s *Service) List(ctx context.Context) ([]Product, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id string) (Product, error) {
	if id == "" {
		return Product{}, ErrInvalidInput
	}

	if cached, ok, err := s.cache.GetProduct(ctx, id); err == nil && ok {
		return cached, nil
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Product{}, err
	}

	_ = s.cache.SetProduct(ctx, p, defaultCacheTTL)
	return p, nil
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Product, error) {
	if id == "" {
		return Product{}, ErrInvalidInput
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Product{}, err
	}

	if input.Name != nil {
		if *input.Name == "" {
			return Product{}, ErrInvalidInput
		}
		p.Name = *input.Name
	}
	if input.Description != nil {
		p.Description = *input.Description
	}
	if input.PriceKZT != nil {
		if *input.PriceKZT < 0 {
			return Product{}, ErrInvalidInput
		}
		p.PriceKZT = *input.PriceKZT
	}

	p.UpdatedAt = time.Now().UTC()
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		return Product{}, err
	}

	s.invalidateProductCache(ctx, id)
	_ = s.publisher.PublishProductUpdated(ctx, ProductEvent{
		ProductID: updated.ID,
		Name:      updated.Name,
		Timestamp: updated.UpdatedAt,
	})

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidInput
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.invalidateProductCache(ctx, id)
	_ = s.publisher.PublishProductDeleted(ctx, ProductEvent{
		ProductID: p.ID,
		Name:      p.Name,
		Timestamp: time.Now().UTC(),
	})

	return nil
}

func (s *Service) UpdateStock(ctx context.Context, id string, stock int) (Product, error) {
	if id == "" || stock < 0 {
		return Product{}, ErrInvalidInput
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Product{}, err
	}

	if stock < p.ReservedStock {
		return Product{}, ErrInsufficientStock
	}

	p.Stock = stock
	p.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		return Product{}, err
	}

	s.invalidateProductCache(ctx, id)
	return updated, nil
}

func (s *Service) ReserveStock(ctx context.Context, id string, input ReserveStockInput) (Product, error) {
	if id == "" || input.Quantity <= 0 {
		return Product{}, ErrInvalidInput
	}

	reservationID := input.ReservationID
	if reservationID == "" {
		reservationID = fmt.Sprintf("res-%d", time.Now().UnixNano())
	}

	updated, err := s.repo.ReserveStock(ctx, id, reservationID, input.Quantity)
	if err != nil {
		return Product{}, err
	}

	s.invalidateProductCache(ctx, id)
	now := time.Now().UTC()
	_ = s.publisher.PublishStockReserved(ctx, StockEvent{
		ProductID:     updated.ID,
		ReservationID: reservationID,
		Quantity:      input.Quantity,
		Stock:         updated.Stock,
		ReservedStock: updated.ReservedStock,
		Available:     updated.Available,
		Timestamp:     now,
	})

	return updated, nil
}

func (s *Service) ReleaseStock(ctx context.Context, id string, input ReleaseStockInput) (Product, error) {
	if id == "" || input.ReservationID == "" {
		return Product{}, ErrInvalidInput
	}

	updated, err := s.repo.ReleaseStock(ctx, id, input.ReservationID, input.Quantity)
	if err != nil {
		return Product{}, err
	}

	s.invalidateProductCache(ctx, id)
	now := time.Now().UTC()
	_ = s.publisher.PublishStockReleased(ctx, StockEvent{
		ProductID:     updated.ID,
		ReservationID: input.ReservationID,
		Quantity:      input.Quantity,
		Stock:         updated.Stock,
		ReservedStock: updated.ReservedStock,
		Available:     updated.Available,
		Timestamp:     now,
	})

	return updated, nil
}

func (s *Service) invalidateProductCache(ctx context.Context, id string) {
	_ = s.cache.InvalidateProduct(ctx, id)
}
