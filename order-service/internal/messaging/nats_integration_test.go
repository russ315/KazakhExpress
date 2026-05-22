//go:build integration

package messaging

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"kazakhexpress/order-service/internal/order"

	"github.com/nats-io/nats.go"
)

func TestNATSPublisherPublishesOrderCreated(t *testing.T) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		t.Skip("NATS_URL is required for integration tests")
	}
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("nats.Connect() error = %v", err)
	}
	t.Cleanup(nc.Close)

	sub, err := nc.SubscribeSync("order.created")
	if err != nil {
		t.Fatalf("SubscribeSync() error = %v", err)
	}
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	event := order.Event{OrderID: "it-order-nats", CustomerID: "it-user-nats", Status: order.StatusCreated, TotalKZT: 1000, OccurredAt: time.Now().UTC()}
	if err := NewNATSPublisher(nc).PublishOrderCreated(context.Background(), event); err != nil {
		t.Fatalf("PublishOrderCreated() error = %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	msg, err := sub.NextMsg(3 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg() error = %v", err)
	}
	var got order.Event
	if err := json.Unmarshal(msg.Data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.OrderID != event.OrderID || got.Status != order.StatusCreated {
		t.Fatalf("event = %+v", got)
	}
}
