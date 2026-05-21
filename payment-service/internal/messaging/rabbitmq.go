package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"kazakhexpress/payment-service/internal/payment"

	amqp "github.com/rabbitmq/amqp091-go"
)

const defaultExchange = "kazakhexpress.events"

type RabbitPublisher struct {
	channel  *amqp.Channel
	exchange string
}

func NewRabbitPublisher(channel *amqp.Channel, exchange string) (*RabbitPublisher, error) {
	if exchange == "" {
		exchange = defaultExchange
	}
	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare payment exchange: %w", err)
	}
	return &RabbitPublisher{channel: channel, exchange: exchange}, nil
}

func (p *RabbitPublisher) PublishPaymentCreated(ctx context.Context, event payment.PaymentEvent) error {
	return p.publish(ctx, "payment.created", event)
}

func (p *RabbitPublisher) PublishPaymentSucceeded(ctx context.Context, event payment.PaymentEvent) error {
	return p.publish(ctx, "payment.succeeded", event)
}

func (p *RabbitPublisher) PublishPaymentFailed(ctx context.Context, event payment.PaymentEvent) error {
	return p.publish(ctx, "payment.failed", event)
}

func (p *RabbitPublisher) PublishPaymentRefunded(ctx context.Context, event payment.PaymentEvent) error {
	return p.publish(ctx, "payment.refunded", event)
}

func (p *RabbitPublisher) PublishPaymentCancelled(ctx context.Context, event payment.PaymentEvent) error {
	return p.publish(ctx, "payment.cancelled", event)
}

func (p *RabbitPublisher) publish(ctx context.Context, key string, event payment.PaymentEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal payment event: %w", err)
	}
	err = p.channel.PublishWithContext(ctx, p.exchange, key, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         payload,
	})
	if err != nil {
		return fmt.Errorf("publish %s: %w", key, err)
	}
	return nil
}
