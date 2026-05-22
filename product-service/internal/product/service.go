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
	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return Product{}, err
	}
	if s.publisher != nil {
		_ = s.publisher.PublishProductCreated(ctx, Event{ProductID: created.ID, Name: created.Name, Stock: created.Stock, OccurredAt: now})
	}
	return created, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Product, error) {
	return s.repo.List(ctx, filter)
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
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		return Product{}, err
	}
	if s.publisher != nil {
		_ = s.publisher.PublishStockUpdated(ctx, Event{ProductID: id, Stock: stock, OccurredAt: updated.UpdatedAt})
	}
	return updated, nil
}

func (s *Service) ReserveStock(ctx context.Context, id string, quantity int) (Product, error) {
	if id == "" || quantity <= 0 {
		return Product{}, ErrInvalidInput
	}
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Product{}, err
	}
	if p.Stock < quantity {
		return Product{}, ErrInvalidInput
	}
	p.Stock -= quantity
	p.UpdatedAt = time.Now().UTC()
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		return Product{}, err
	}
	if s.publisher != nil {
		_ = s.publisher.PublishStockReserved(ctx, Event{ProductID: id, Quantity: quantity, Stock: updated.Stock, OccurredAt: updated.UpdatedAt})
	}
	return updated, nil
}

func (s *Service) ReleaseStock(ctx context.Context, id string, quantity int) (Product, error) {
	if id == "" || quantity <= 0 {
		return Product{}, ErrInvalidInput
	}
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Product{}, err
	}
	p.Stock += quantity
	p.UpdatedAt = time.Now().UTC()
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		return Product{}, err
	}
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
	image := ProductImage{
		ID:        fmt.Sprintf("img-%d", now.UnixNano()),
		ProductID: input.ProductID,
		Object:    object,
		URL:       url,
		CreatedAt: now,
	}
	return s.repo.AddImage(ctx, image)
}
