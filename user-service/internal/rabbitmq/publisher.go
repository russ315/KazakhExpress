package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventType string

const (
	EventUserCreated EventType = "user.created"
	EventUserUpdated EventType = "user.updated"
)

type UserEvent struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Event     EventType `json:"event"`
	Timestamp time.Time `json:"timestamp"`
}

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewPublisher(amqpURL string) (*Publisher, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		"user.events",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &Publisher{conn: conn, channel: ch}, nil
}

func (p *Publisher) PublishUserEvent(ctx context.Context, event UserEvent) error {
	if p.channel == nil {
		log.Printf("RabbitMQ not connected, skipping event: %s", event.Event)
		return nil
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return p.channel.PublishWithContext(ctx,
		"user.events",
		string(event.Event),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
			DeliveryMode: amqp.Persistent,
		},
	)
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
