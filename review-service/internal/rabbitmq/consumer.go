package rabbitmq

import (
	"context"
	"encoding/json"
	"log"

	"kazakhexpress/review-service/internal/review"

	amqp "github.com/rabbitmq/amqp091-go"
)

const orderExchange = "kazakhexpress.events"

type Consumer struct {
	service *review.Service
}

func NewConsumer(service *review.Service) *Consumer {
	return &Consumer{service: service}
}

func (c *Consumer) Start(ctx context.Context, conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	queue, err := ch.QueueDeclare("review-service.order-completed", true, false, false, false, nil)
	if err != nil {
		return err
	}

	if err := ch.ExchangeDeclare(orderExchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(queue.Name, "order.completed", orderExchange, false, nil); err != nil {
		return err
	}

	deliveries, err := ch.Consume(queue.Name, "review-service", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}
				if err := c.handleOrderCompleted(ctx, delivery.Body); err != nil {
					log.Printf("order.completed handler failed: %v", err)
					_ = delivery.Nack(false, true)
					continue
				}
				_ = delivery.Ack(false)
			}
		}
	}()

	log.Printf("review-service consuming order.completed from %s", orderExchange)
	return nil
}

func (c *Consumer) handleOrderCompleted(ctx context.Context, body []byte) error {
	var event review.OrderCompletedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return err
	}
	return c.service.HandleOrderCompleted(ctx, event)
}
