package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type EventType string

type UserEvent struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Event     EventType `json:"event"`
	Timestamp time.Time `json:"timestamp"`
}

type Publisher struct {
	conn *nats.Conn
}

func NewPublisher(conn *nats.Conn) *Publisher {
	return &Publisher{conn: conn}
}

func (p *Publisher) PublishUserEvent(ctx context.Context, event UserEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal user event: %w", err)
	}
	if err := p.conn.Publish(string(event.Event), payload); err != nil {
		return fmt.Errorf("publish user event: %w", err)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
