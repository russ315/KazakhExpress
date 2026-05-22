package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"kazakhexpress/review-service/internal/review"

	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	conn *nats.Conn
}

func NewNATSPublisher(conn *nats.Conn) *NATSPublisher {
	return &NATSPublisher{conn: conn}
}

func (p *NATSPublisher) PublishReviewCreated(ctx context.Context, event review.Event) error {
	return p.publish(ctx, "review.created", event)
}

func (p *NATSPublisher) PublishReviewUpdated(ctx context.Context, event review.Event) error {
	return p.publish(ctx, "review.updated", event)
}

func (p *NATSPublisher) PublishReviewDeleted(ctx context.Context, event review.Event) error {
	return p.publish(ctx, "review.deleted", event)
}

func (p *NATSPublisher) PublishRatingUpdated(ctx context.Context, rating review.Rating) error {
	return p.publish(ctx, "product.rating.updated", rating)
}

func (p *NATSPublisher) publish(ctx context.Context, subject string, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", subject, err)
	}
	if err := p.conn.Publish(subject, payload); err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
