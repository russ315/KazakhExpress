package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"kazakhexpress/order-service/internal/order"

	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	conn *nats.Conn
}

func NewNATSPublisher(conn *nats.Conn) *NATSPublisher {
	return &NATSPublisher{conn: conn}
}

func (p *NATSPublisher) PublishOrderCreated(ctx context.Context, event order.Event) error {
	return p.publish(ctx, "order.created", event)
}

func (p *NATSPublisher) PublishOrderCancelled(ctx context.Context, event order.Event) error {
	return p.publish(ctx, "order.cancelled", event)
}

func (p *NATSPublisher) PublishOrderCompleted(ctx context.Context, event order.Event) error {
	return p.publish(ctx, "order.completed", event)
}

func (p *NATSPublisher) publish(ctx context.Context, subject string, event order.Event) error {
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

type NATSConsumer struct {
	conn    *nats.Conn
	service *order.Service
	subs    []*nats.Subscription
}

func NewNATSConsumer(conn *nats.Conn, service *order.Service) *NATSConsumer {
	return &NATSConsumer{conn: conn, service: service}
}

func (c *NATSConsumer) Start(ctx context.Context) error {
	if err := c.subscribePayment(ctx, "payment.succeeded", c.service.HandlePaymentSucceeded); err != nil {
		return err
	}
	if err := c.subscribePayment(ctx, "payment.failed", c.service.HandlePaymentFailed); err != nil {
		return err
	}
	sub, err := c.conn.QueueSubscribe("product.stock.reserved", "order-service", func(msg *nats.Msg) {
		var event order.StockReservedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("decode product.stock.reserved: %v", err)
			return
		}
		if err := c.service.HandleStockReserved(ctx, event); err != nil {
			log.Printf("handle product.stock.reserved: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe product.stock.reserved: %w", err)
	}
	c.subs = append(c.subs, sub)
	return nil
}

func (c *NATSConsumer) Close() {
	for _, sub := range c.subs {
		_ = sub.Drain()
	}
}

func (c *NATSConsumer) subscribePayment(ctx context.Context, subject string, handler func(context.Context, order.PaymentEvent) error) error {
	sub, err := c.conn.QueueSubscribe(subject, "order-service", func(msg *nats.Msg) {
		var event order.PaymentEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("decode %s: %v", subject, err)
			return
		}
		if err := handler(ctx, event); err != nil {
			log.Printf("handle %s: %v", subject, err)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", subject, err)
	}
	c.subs = append(c.subs, sub)
	return nil
}
