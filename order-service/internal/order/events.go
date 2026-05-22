package order

import "context"

type OrderCompletedEvent struct {
	OrderID    string               `json:"order_id"`
	CustomerID string               `json:"customer_id"`
	Items      []OrderCompletedItem `json:"items"`
}

type OrderCompletedItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type EventPublisher interface {
	PublishOrderCompleted(ctx context.Context, event OrderCompletedEvent) error
}

type NoopPublisher struct{}

func (NoopPublisher) PublishOrderCompleted(context.Context, OrderCompletedEvent) error { return nil }
