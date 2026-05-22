package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kazakhexpress/review-service/internal/review"

	amqp "github.com/rabbitmq/amqp091-go"
)

const exchangeName = "review.events"

type Publisher struct {
	channel *amqp.Channel
}

func NewPublisher(conn *amqp.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	if err := ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare exchange: %w", err)
	}
	return &Publisher{channel: ch}, nil
}

func (p *Publisher) Close() error {
	if p.channel != nil {
		return p.channel.Close()
	}
	return nil
}

type EventAdapter struct {
	publisher *Publisher
}

func NewEventAdapter(publisher *Publisher) *EventAdapter {
	return &EventAdapter{publisher: publisher}
}

func (a *EventAdapter) PublishReviewCreated(ctx context.Context, event review.ReviewEvent) error {
	return a.publish(ctx, "review.created", event)
}

func (a *EventAdapter) PublishReviewUpdated(ctx context.Context, event review.ReviewEvent) error {
	return a.publish(ctx, "review.updated", event)
}

func (a *EventAdapter) PublishReviewDeleted(ctx context.Context, event review.ReviewEvent) error {
	return a.publish(ctx, "review.deleted", event)
}

func (a *EventAdapter) PublishProductRatingUpdated(ctx context.Context, event review.RatingEvent) error {
	return a.publish(ctx, "product.rating.updated", event)
}

func (a *EventAdapter) publish(ctx context.Context, routingKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return a.publisher.channel.PublishWithContext(ctx, exchangeName, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		Timestamp:    time.Now().UTC(),
		DeliveryMode: amqp.Persistent,
	})
}
