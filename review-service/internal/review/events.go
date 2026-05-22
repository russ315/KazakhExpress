package review

import (
	"context"
	"time"
)

type ReviewEvent struct {
	ReviewID  string    `json:"review_id"`
	ProductID string    `json:"product_id"`
	UserID    string    `json:"user_id"`
	Rating    int       `json:"rating"`
	Timestamp time.Time `json:"timestamp"`
}

type RatingEvent struct {
	ProductID   string    `json:"product_id"`
	RatingAvg   float64   `json:"rating_avg"`
	RatingCount int       `json:"rating_count"`
	Timestamp   time.Time `json:"timestamp"`
}

type EventPublisher interface {
	PublishReviewCreated(ctx context.Context, event ReviewEvent) error
	PublishReviewUpdated(ctx context.Context, event ReviewEvent) error
	PublishReviewDeleted(ctx context.Context, event ReviewEvent) error
	PublishProductRatingUpdated(ctx context.Context, event RatingEvent) error
}

type NoopPublisher struct{}

func (NoopPublisher) PublishReviewCreated(context.Context, ReviewEvent) error       { return nil }
func (NoopPublisher) PublishReviewUpdated(context.Context, ReviewEvent) error       { return nil }
func (NoopPublisher) PublishReviewDeleted(context.Context, ReviewEvent) error       { return nil }
func (NoopPublisher) PublishProductRatingUpdated(context.Context, RatingEvent) error { return nil }
