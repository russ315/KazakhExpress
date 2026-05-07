package order

import (
	"context"
	"errors"
	"sync"
)

var ErrNotFound = errors.New("order not found")

type Repository interface {
	Create(ctx context.Context, order Order) (Order, error)
	List(ctx context.Context) ([]Order, error)
	GetByID(ctx context.Context, id string) (Order, error)
	Update(ctx context.Context, order Order) (Order, error)
}

type MemoryRepository struct {
	mu     sync.RWMutex
	orders map[string]Order
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		orders: make(map[string]Order),
	}
}

func (r *MemoryRepository) Create(ctx context.Context, order Order) (Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.orders[order.ID] = order
	return order, nil
}

func (r *MemoryRepository) List(ctx context.Context) ([]Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orders := make([]Order, 0, len(r.orders))
	for _, order := range r.orders {
		orders = append(orders, order)
	}

	return orders, nil
}

func (r *MemoryRepository) GetByID(ctx context.Context, id string) (Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[id]
	if !ok {
		return Order{}, ErrNotFound
	}

	return order, nil
}

func (r *MemoryRepository) Update(ctx context.Context, order Order) (Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.orders[order.ID]; !ok {
		return Order{}, ErrNotFound
	}

	r.orders[order.ID] = order
	return order, nil
}
