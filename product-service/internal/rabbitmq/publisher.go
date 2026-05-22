package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"kazakhexpress/product-service/internal/product"

	amqp "github.com/rabbitmq/amqp091-go"
)

const exchangeName = "product.events"

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewPublisher(amqpURL string) (*Publisher, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("connect rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	return &Publisher{conn: conn, channel: ch}, nil
}

func (p *Publisher) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

type EventAdapter struct {
	publisher *Publisher
}

func NewEventAdapter(publisher *Publisher) *EventAdapter {
	return &EventAdapter{publisher: publisher}
}

func (a *EventAdapter) PublishProductCreated(ctx context.Context, event product.ProductEvent) error {
	return a.publish(ctx, "product.created", event)
}

func (a *EventAdapter) PublishProductUpdated(ctx context.Context, event product.ProductEvent) error {
	return a.publish(ctx, "product.updated", event)
}

func (a *EventAdapter) PublishProductDeleted(ctx context.Context, event product.ProductEvent) error {
	return a.publish(ctx, "product.deleted", event)
}

func (a *EventAdapter) PublishStockReserved(ctx context.Context, event product.StockEvent) error {
	return a.publish(ctx, "stock.reserved", event)
}

func (a *EventAdapter) PublishStockReleased(ctx context.Context, event product.StockEvent) error {
	return a.publish(ctx, "stock.released", event)
}

func (a *EventAdapter) publish(ctx context.Context, routingKey string, payload any) error {
	if a.publisher == nil || a.publisher.channel == nil {
		log.Printf("rabbitmq not connected, skipping event: %s", routingKey)
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return a.publisher.channel.PublishWithContext(ctx,
		exchangeName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now().UTC(),
			DeliveryMode: amqp.Persistent,
		},
	)
}
