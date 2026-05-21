package user

import (
	"context"
	"fmt"

	rmq "kazakhexpress/user-service/internal/rabbitmq"
)

type RabbitMQEventAdapter struct {
	publisher *rmq.Publisher
}

func NewRabbitMQEventAdapter(publisher *rmq.Publisher) *RabbitMQEventAdapter {
	return &RabbitMQEventAdapter{publisher: publisher}
}

func (a *RabbitMQEventAdapter) PublishUserEvent(ctx context.Context, event interface{}) error {
	userEvent, ok := event.(UserEvent)
	if !ok {
		return fmt.Errorf("expected UserEvent, got %T", event)
	}

	return a.publisher.PublishUserEvent(ctx, rmq.UserEvent{
		UserID:    userEvent.UserID,
		Email:     userEvent.Email,
		FirstName: userEvent.FirstName,
		LastName:  userEvent.LastName,
		Event:     rmq.EventType(userEvent.Event),
		Timestamp: userEvent.Timestamp,
	})
}

func (a *RabbitMQEventAdapter) Close() error {
	return a.publisher.Close()
}
