package product

import (
	"context"
	"time"
)

type StockEvent struct {
	ProductID     string    `json:"product_id"`
	ReservationID string    `json:"reservation_id"`
	Quantity      int       `json:"quantity"`
	Stock         int       `json:"stock"`
	ReservedStock int       `json:"reserved_stock"`
	Available     int       `json:"available"`
	Timestamp     time.Time `json:"timestamp"`
}

type ProductEvent struct {
	ProductID string    `json:"product_id"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
}

type EventPublisher interface {
	PublishProductCreated(ctx context.Context, event ProductEvent) error
	PublishProductUpdated(ctx context.Context, event ProductEvent) error
	PublishProductDeleted(ctx context.Context, event ProductEvent) error
	PublishStockReserved(ctx context.Context, event StockEvent) error
	PublishStockReleased(ctx context.Context, event StockEvent) error
}

type NoopPublisher struct{}

func (NoopPublisher) PublishProductCreated(context.Context, ProductEvent) error { return nil }
func (NoopPublisher) PublishProductUpdated(context.Context, ProductEvent) error { return nil }
func (NoopPublisher) PublishProductDeleted(context.Context, ProductEvent) error { return nil }
func (NoopPublisher) PublishStockReserved(context.Context, StockEvent) error    { return nil }
func (NoopPublisher) PublishStockReleased(context.Context, StockEvent) error    { return nil }
