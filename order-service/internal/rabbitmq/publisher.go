package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kazakhexpress/order-service/internal/order"

	amqp "github.com/rabbitmq/amqp091-go"
)

const exchangeName = "kazakhexpress.events"

type Publisher struct {
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
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, err
	}
	return &Publisher{channel: ch}, nil
}

func (p *Publisher) PublishOrderCompleted(ctx context.Context, event order.OrderCompletedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.channel.PublishWithContext(ctx, exchangeName, "order.completed", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		Timestamp:    time.Now().UTC(),
		DeliveryMode: amqp.Persistent,
	})
}
