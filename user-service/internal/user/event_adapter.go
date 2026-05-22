package user

import (
	"context"
	"fmt"

	"kazakhexpress/user-service/internal/messaging"
)

type NATSEventAdapter struct {
	publisher *messaging.Publisher
}

func NewNATSEventAdapter(publisher *messaging.Publisher) *NATSEventAdapter {
	return &NATSEventAdapter{publisher: publisher}
}

func (a *NATSEventAdapter) PublishUserEvent(ctx context.Context, event interface{}) error {
	userEvent, ok := event.(UserEvent)
	if !ok {
		return fmt.Errorf("expected UserEvent, got %T", event)
	}

	return a.publisher.PublishUserEvent(ctx, messaging.UserEvent{
		UserID:    userEvent.UserID,
		Email:     userEvent.Email,
		FirstName: userEvent.FirstName,
		LastName:  userEvent.LastName,
		Event:     messaging.EventType(userEvent.Event),
		Timestamp: userEvent.Timestamp,
	})
}
