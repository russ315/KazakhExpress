package product

import (
	"context"
	"errors"
	"sync"
)

var ErrNotFound = errors.New("product not found")

type Repository interface {
	Create(ctx context.Context, p Product) (Product, error)
	List(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	Update(ctx context.Context, p Product) (Product, error)
}

type MemoryRepository struct {
	mu       sync.RWMutex
	products map[string]Product
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		products: make(map[string]Product),
	}
}

func (r *MemoryRepository) Create(ctx context.Context, p Product) (Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.products[p.ID] = p
	return p, nil
}

func (r *MemoryRepository) List(ctx context.Context) ([]Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Product, 0, len(r.products))
	for _, p := range r.products {
		out = append(out, p)
	}

	return out, nil
}

func (r *MemoryRepository) GetByID(ctx context.Context, id string) (Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.products[id]
	if !ok {
		return Product{}, ErrNotFound
	}

	return p, nil
}

func (r *MemoryRepository) Update(ctx context.Context, p Product) (Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.products[p.ID]; !ok {
		return Product{}, ErrNotFound
	}

	r.products[p.ID] = p
	return p, nil
}
