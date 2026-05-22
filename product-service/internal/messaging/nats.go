package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"kazakhexpress/product-service/internal/product"

	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	conn *nats.Conn
}

func NewNATSPublisher(conn *nats.Conn) *NATSPublisher {
	return &NATSPublisher{conn: conn}
}

func (p *NATSPublisher) PublishProductCreated(ctx context.Context, event product.Event) error {
	return p.publish(ctx, "product.created", event)
}

func (p *NATSPublisher) PublishStockUpdated(ctx context.Context, event product.Event) error {
	return p.publish(ctx, "product.stock.updated", event)
}

func (p *NATSPublisher) PublishStockReserved(ctx context.Context, event product.Event) error {
	return p.publish(ctx, "product.stock.reserved", event)
}

func (p *NATSPublisher) PublishStockReleased(ctx context.Context, event product.Event) error {
	return p.publish(ctx, "product.stock.released", event)
}

func (p *NATSPublisher) publish(ctx context.Context, subject string, event product.Event) error {
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
