package product

import (
	"context"
	"time"
)

type CacheService interface {
	GetProduct(ctx context.Context, id string) (Product, bool, error)
	SetProduct(ctx context.Context, p Product, ttl time.Duration) error
	InvalidateProduct(ctx context.Context, id string) error
}

type NoopCache struct{}

func (NoopCache) GetProduct(context.Context, string) (Product, bool, error) {
	return Product{}, false, nil
}
func (NoopCache) SetProduct(context.Context, Product, time.Duration) error { return nil }
func (NoopCache) InvalidateProduct(context.Context, string) error          { return nil }
