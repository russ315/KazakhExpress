package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"kazakhexpress/payment-service/internal/payment"

	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	conn *nats.Conn
}

func NewNATSPublisher(conn *nats.Conn) *NATSPublisher {
	return &NATSPublisher{conn: conn}
}

func (p *NATSPublisher) PublishPaymentCreated(ctx context.Context, event payment.PaymentEvent) error {
	return p.publish(ctx, "payment.created", event)
}

func (p *NATSPublisher) PublishPaymentSucceeded(ctx context.Context, event payment.PaymentEvent) error {
	return p.publish(ctx, "payment.succeeded", event)
}

func (p *NATSPublisher) PublishPaymentFailed(ctx context.Context, event payment.PaymentEvent) error {
	return p.publish(ctx, "payment.failed", event)
}

func (p *NATSPublisher) PublishPaymentRefunded(ctx context.Context, event payment.PaymentEvent) error {
	return p.publish(ctx, "payment.refunded", event)
}

func (p *NATSPublisher) PublishPaymentCancelled(ctx context.Context, event payment.PaymentEvent) error {
	return p.publish(ctx, "payment.cancelled", event)
}

func (p *NATSPublisher) publish(ctx context.Context, subject string, event payment.PaymentEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal payment event: %w", err)
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
