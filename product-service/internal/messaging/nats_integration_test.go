//go:build integration

package messaging

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"kazakhexpress/product-service/internal/product"

	"github.com/nats-io/nats.go"
)

func TestNATSPublisherPublishesStockReserved(t *testing.T) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		t.Skip("NATS_URL is required for integration tests")
	}
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("nats.Connect() error = %v", err)
	}
	t.Cleanup(nc.Close)

	sub, err := nc.SubscribeSync("product.stock.reserved")
	if err != nil {
		t.Fatalf("SubscribeSync() error = %v", err)
	}
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	event := product.Event{ProductID: "it-product-nats", Quantity: 2, Stock: 8, OccurredAt: time.Now().UTC()}
	if err := NewNATSPublisher(nc).PublishStockReserved(context.Background(), event); err != nil {
		t.Fatalf("PublishStockReserved() error = %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	msg, err := sub.NextMsg(3 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg() error = %v", err)
	}
	var got product.Event
	if err := json.Unmarshal(msg.Data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.ProductID != event.ProductID || got.Quantity != event.Quantity {
		t.Fatalf("event = %+v", got)
	}
}
