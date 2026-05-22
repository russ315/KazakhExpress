package product

import (
	"context"
	"sync"
)

type Repository interface {
	Create(ctx context.Context, p Product) (Product, error)
	List(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	Update(ctx context.Context, p Product) (Product, error)
	Delete(ctx context.Context, id string) error
	ReserveStock(ctx context.Context, productID, reservationID string, quantity int) (Product, error)
	ReleaseStock(ctx context.Context, productID, reservationID string, quantity int) (Product, error)
}

type reservation struct {
	productID string
	quantity  int
}

type MemoryRepository struct {
	mu           sync.RWMutex
	products     map[string]Product
	reservations map[string]reservation
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		products:     make(map[string]Product),
		reservations: make(map[string]reservation),
	}
}

func (r *MemoryRepository) Create(ctx context.Context, p Product) (Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[p.ID] = p
	return WithAvailability(p), nil
}

func (r *MemoryRepository) List(ctx context.Context) ([]Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Product, 0, len(r.products))
	for _, p := range r.products {
		out = append(out, WithAvailability(p))
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
	return WithAvailability(p), nil
}

func (r *MemoryRepository) Update(ctx context.Context, p Product) (Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.products[p.ID]; !ok {
		return Product{}, ErrNotFound
	}
	r.products[p.ID] = p
	return WithAvailability(p), nil
}

func (r *MemoryRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.products[id]; !ok {
		return ErrNotFound
	}
	delete(r.products, id)
	for resID, res := range r.reservations {
		if res.productID == id {
			delete(r.reservations, resID)
		}
	}
	return nil
}

func (r *MemoryRepository) ReserveStock(ctx context.Context, productID, reservationID string, quantity int) (Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.products[productID]
	if !ok {
		return Product{}, ErrNotFound
	}
	if existing, exists := r.reservations[reservationID]; exists {
		if existing.productID != productID {
			return Product{}, ErrInvalidInput
		}
		return WithAvailability(p), nil
	}
	if quantity > p.Stock-p.ReservedStock {
		return Product{}, ErrInsufficientStock
	}
	p.ReservedStock += quantity
	r.products[productID] = p
	r.reservations[reservationID] = reservation{productID: productID, quantity: quantity}
	return WithAvailability(p), nil
}

func (r *MemoryRepository) ReleaseStock(ctx context.Context, productID, reservationID string, quantity int) (Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.products[productID]
	if !ok {
		return Product{}, ErrNotFound
	}
	res, exists := r.reservations[reservationID]
	if !exists || res.productID != productID {
		return Product{}, ErrReservationNotFound
	}
	releaseQty := quantity
	if releaseQty <= 0 || releaseQty > res.quantity {
		releaseQty = res.quantity
	}
	p.ReservedStock -= releaseQty
	if p.ReservedStock < 0 {
		p.ReservedStock = 0
	}
	remaining := res.quantity - releaseQty
	if remaining <= 0 {
		delete(r.reservations, reservationID)
	} else {
		r.reservations[reservationID] = reservation{productID: productID, quantity: remaining}
	}
	r.products[productID] = p
	return WithAvailability(p), nil
}
