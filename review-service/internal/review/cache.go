package review

import (
	"context"
	"time"
)

type CacheService interface {
	GetRating(ctx context.Context, productID string) (ProductRating, bool, error)
	SetRating(ctx context.Context, rating ProductRating, ttl time.Duration) error
	InvalidateRating(ctx context.Context, productID string) error

	GetReviewsPage(ctx context.Context, productID string, page int) (ListPage, bool, error)
	SetReviewsPage(ctx context.Context, productID string, page ListPage, ttl time.Duration) error
	InvalidateReviews(ctx context.Context, productID string) error
}

type NoopCache struct{}

func (NoopCache) GetRating(context.Context, string) (ProductRating, bool, error) {
	return ProductRating{}, false, nil
}
func (NoopCache) SetRating(context.Context, ProductRating, time.Duration) error { return nil }
func (NoopCache) InvalidateRating(context.Context, string) error                { return nil }
func (NoopCache) GetReviewsPage(context.Context, string, int) (ListPage, bool, error) {
	return ListPage{}, false, nil
}
func (NoopCache) SetReviewsPage(context.Context, string, ListPage, time.Duration) error { return nil }
func (NoopCache) InvalidateReviews(context.Context, string) error                     { return nil }
