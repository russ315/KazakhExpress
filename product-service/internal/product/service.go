package product

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var ErrInvalidInput = errors.New("invalid product input")

type ImageStorage interface {
	Save(ctx context.Context, objectName, contentType string, content []byte) (string, error)
}

type EventPublisher interface {
	PublishProductCreated(ctx context.Context, event Event) error
	PublishStockUpdated(ctx context.Context, event Event) error
	PublishStockReserved(ctx context.Context, event Event) error
	PublishStockReleased(ctx context.Context, event Event) error
}

type Service struct {
	repo      Repository
	storage   ImageStorage
	publisher EventPublisher
}

func NewService(repo Repository, storage ImageStorage, publisher EventPublisher) *Service {
	return &Service{repo: repo, storage: storage, publisher: publisher}
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

	dbStart := time.Now()
	created, err := s.repo.Create(ctx, p)
	if err != nil {
		ProductDBOperationsTotal.WithLabelValues("create", "error").Inc()
		ProductDBOperationDurationSeconds.WithLabelValues("create").Observe(time.Since(dbStart).Seconds())
		return Product{}, err
	}
	ProductDBOperationsTotal.WithLabelValues("create", "success").Inc()
	ProductDBOperationDurationSeconds.WithLabelValues("create").Observe(time.Since(dbStart).Seconds())

	ProductsCreatedTotal.Inc()

	if s.publisher != nil {
		_ = s.publisher.PublishProductCreated(ctx, Event{ProductID: created.ID, Name: created.Name, Stock: created.Stock, OccurredAt: now})
	}
	return created, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Product, error) {
	dbStart := time.Now()
	res, err := s.repo.List(ctx, filter)
	if err != nil {
		ProductDBOperationsTotal.WithLabelValues("list", "error").Inc()
		ProductDBOperationDurationSeconds.WithLabelValues("list").Observe(time.Since(dbStart).Seconds())
		return nil, err
	}
	ProductDBOperationsTotal.WithLabelValues("list", "success").Inc()
	ProductDBOperationDurationSeconds.WithLabelValues("list").Observe(time.Since(dbStart).Seconds())
	return res, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (Product, error) {
	if id == "" {
		return Product{}, ErrInvalidInput
	}
	dbStart := time.Now()
	res, err := s.repo.GetByID(ctx, id)
	if err != nil {
		ProductDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
		ProductDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStart).Seconds())
		return Product{}, err
	}
	ProductDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
	ProductDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStart).Seconds())
	return res, nil
}

func (s *Service) UpdateStock(ctx context.Context, id string, stock int) (Product, error) {
	if id == "" || stock < 0 {
		return Product{}, ErrInvalidInput
	}
	dbStartGet := time.Now()
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		ProductDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
		ProductDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())
		return Product{}, err
	}
	ProductDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
	ProductDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())

	p.Stock = stock
	p.UpdatedAt = time.Now().UTC()

	dbStartUpdate := time.Now()
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		ProductDBOperationsTotal.WithLabelValues("update_stock", "error").Inc()
		ProductDBOperationDurationSeconds.WithLabelValues("update_stock").Observe(time.Since(dbStartUpdate).Seconds())
		return Product{}, err
	}
	ProductDBOperationsTotal.WithLabelValues("update_stock", "success").Inc()
	ProductDBOperationDurationSeconds.WithLabelValues("update_stock").Observe(time.Since(dbStartUpdate).Seconds())

	ProductStockUpdatesTotal.WithLabelValues("update").Inc()

	if s.publisher != nil {
		_ = s.publisher.PublishStockUpdated(ctx, Event{ProductID: id, Stock: stock, OccurredAt: updated.UpdatedAt})
	}
	return updated, nil
}

func (s *Service) ReserveStock(ctx context.Context, id string, quantity int) (Product, error) {
	if id == "" || quantity <= 0 {
		return Product{}, ErrInvalidInput
	}
	dbStartGet := time.Now()
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		ProductDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
		ProductDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())
		return Product{}, err
	}
	ProductDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
	ProductDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())

	if p.Stock < quantity {
		return Product{}, ErrInvalidInput
	}
	p.Stock -= quantity
	p.UpdatedAt = time.Now().UTC()

	dbStartUpdate := time.Now()
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		ProductDBOperationsTotal.WithLabelValues("reserve_stock", "error").Inc()
		ProductDBOperationDurationSeconds.WithLabelValues("reserve_stock").Observe(time.Since(dbStartUpdate).Seconds())
		return Product{}, err
	}
	ProductDBOperationsTotal.WithLabelValues("reserve_stock", "success").Inc()
	ProductDBOperationDurationSeconds.WithLabelValues("reserve_stock").Observe(time.Since(dbStartUpdate).Seconds())

	ProductStockUpdatesTotal.WithLabelValues("reserve").Inc()

	if s.publisher != nil {
		_ = s.publisher.PublishStockReserved(ctx, Event{ProductID: id, Quantity: quantity, Stock: updated.Stock, OccurredAt: updated.UpdatedAt})
	}
	return updated, nil
}

func (s *Service) ReleaseStock(ctx context.Context, id string, quantity int) (Product, error) {
	if id == "" || quantity <= 0 {
		return Product{}, ErrInvalidInput
	}
	dbStartGet := time.Now()
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		ProductDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
		ProductDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())
		return Product{}, err
	}
	ProductDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
	ProductDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())

	p.Stock += quantity
	p.UpdatedAt = time.Now().UTC()

	dbStartUpdate := time.Now()
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		ProductDBOperationsTotal.WithLabelValues("release_stock", "error").Inc()
		ProductDBOperationDurationSeconds.WithLabelValues("release_stock").Observe(time.Since(dbStartUpdate).Seconds())
		return Product{}, err
	}
	ProductDBOperationsTotal.WithLabelValues("release_stock", "success").Inc()
	ProductDBOperationDurationSeconds.WithLabelValues("release_stock").Observe(time.Since(dbStartUpdate).Seconds())

	ProductStockUpdatesTotal.WithLabelValues("release").Inc()

	if s.publisher != nil {
		_ = s.publisher.PublishStockReleased(ctx, Event{ProductID: id, Quantity: quantity, Stock: updated.Stock, OccurredAt: updated.UpdatedAt})
	}
	return updated, nil
}

func (s *Service) AddImage(ctx context.Context, input ImageInput) (ProductImage, error) {
	if input.ProductID == "" || len(input.Content) == 0 || s.storage == nil {
		return ProductImage{}, ErrInvalidInput
	}
	now := time.Now().UTC()
	name := strings.TrimSpace(filepath.Base(input.Filename))
	if name == "." || name == "" {
		name = "product-image"
	}
	object := fmt.Sprintf("products/%s/%d-%s", input.ProductID, now.UnixNano(), name)
	url, err := s.storage.Save(ctx, object, input.ContentType, input.Content)
	if err != nil {
		return ProductImage{}, err
	}
	ProductImageUploadsTotal.WithLabelValues(input.ContentType).Inc()

	image := ProductImage{
		ID:        fmt.Sprintf("img-%d", now.UnixNano()),
		ProductID: input.ProductID,
		Object:    object,
		URL:       url,
		CreatedAt: now,
	}

	dbStart := time.Now()
	res, err := s.repo.AddImage(ctx, image)
	if err != nil {
		ProductDBOperationsTotal.WithLabelValues("add_image", "error").Inc()
		ProductDBOperationDurationSeconds.WithLabelValues("add_image").Observe(time.Since(dbStart).Seconds())
		return ProductImage{}, err
	}
	ProductDBOperationsTotal.WithLabelValues("add_image", "success").Inc()
	ProductDBOperationDurationSeconds.WithLabelValues("add_image").Observe(time.Since(dbStart).Seconds())

	return res, nil
}
