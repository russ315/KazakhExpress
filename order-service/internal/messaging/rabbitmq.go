package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"kazakhexpress/order-service/internal/order"

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
		return nil, fmt.Errorf("declare order exchange: %w", err)
	}
	return &RabbitPublisher{channel: channel, exchange: exchange}, nil
}

func (p *RabbitPublisher) PublishOrderCreated(ctx context.Context, event order.Event) error {
	return p.publish(ctx, "order.created", event)
}

func (p *RabbitPublisher) PublishOrderCancelled(ctx context.Context, event order.Event) error {
	return p.publish(ctx, "order.cancelled", event)
}

func (p *RabbitPublisher) PublishOrderCompleted(ctx context.Context, event order.Event) error {
	return p.publish(ctx, "order.completed", event)
}

func (p *RabbitPublisher) publish(ctx context.Context, key string, event order.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", key, err)
	}
	return p.channel.PublishWithContext(ctx, p.exchange, key, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         payload,
	})
}

type RabbitConsumer struct {
	channel  *amqp.Channel
	exchange string
	service  *order.Service
}

func NewRabbitConsumer(channel *amqp.Channel, exchange string, service *order.Service) (*RabbitConsumer, error) {
	if exchange == "" {
		exchange = defaultExchange
	}
	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare order exchange: %w", err)
	}
	return &RabbitConsumer{channel: channel, exchange: exchange, service: service}, nil
}

func (c *RabbitConsumer) Start(ctx context.Context) error {
	if err := c.bind("order.payment_succeeded.queue", "payment.succeeded"); err != nil {
		return err
	}
	if err := c.bind("order.payment_failed.queue", "payment.failed"); err != nil {
		return err
	}
	if err := c.bind("order.stock_reserved.queue", "product.stock.reserved"); err != nil {
		return err
	}

	deliveries, err := c.channel.Consume("order.payment_succeeded.queue", "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume payment.succeeded: %w", err)
	}
	failedDeliveries, err := c.channel.Consume("order.payment_failed.queue", "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume payment.failed: %w", err)
	}
	stockDeliveries, err := c.channel.Consume("order.stock_reserved.queue", "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume product.stock.reserved: %w", err)
	}

	go c.consumePayment(ctx, deliveries, c.service.HandlePaymentSucceeded)
	go c.consumePayment(ctx, failedDeliveries, c.service.HandlePaymentFailed)
	go c.consumeStock(ctx, stockDeliveries)
	return nil
}

func (c *RabbitConsumer) bind(queueName string, key string) error {
	if _, err := c.channel.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare %s: %w", queueName, err)
	}
	if err := c.channel.QueueBind(queueName, key, c.exchange, false, nil); err != nil {
		return fmt.Errorf("bind %s: %w", key, err)
	}
	return nil
}

func (c *RabbitConsumer) consumePayment(ctx context.Context, deliveries <-chan amqp.Delivery, handler func(context.Context, order.PaymentEvent) error) {
	for {
		select {
		case <-ctx.Done():
			return
		case delivery, ok := <-deliveries:
			if !ok {
				return
			}
			var event order.PaymentEvent
			if err := json.Unmarshal(delivery.Body, &event); err != nil {
				log.Printf("decode payment event: %v", err)
				_ = delivery.Nack(false, false)
				continue
			}
			if err := handler(ctx, event); err != nil {
				log.Printf("handle payment event for order %s: %v", event.OrderID, err)
				_ = delivery.Nack(false, true)
				continue
			}
			_ = delivery.Ack(false)
		}
	}
}

func (c *RabbitConsumer) consumeStock(ctx context.Context, deliveries <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case delivery, ok := <-deliveries:
			if !ok {
				return
			}
			var event order.StockReservedEvent
			if err := json.Unmarshal(delivery.Body, &event); err != nil {
				log.Printf("decode stock event: %v", err)
				_ = delivery.Nack(false, false)
				continue
			}
			if err := c.service.HandleStockReserved(ctx, event); err != nil {
				log.Printf("handle stock reserved for order %s: %v", event.OrderID, err)
				_ = delivery.Nack(false, true)
				continue
			}
			_ = delivery.Ack(false)
		}
	}
}
