//go:build integration

package messaging

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"kazakhexpress/payment-service/internal/payment"

	"github.com/nats-io/nats.go"
)

func TestNATSPublisherPublishesPaymentSucceeded(t *testing.T) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		t.Skip("NATS_URL is required for integration tests")
	}

	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("nats.Connect() error = %v", err)
	}
	t.Cleanup(nc.Close)

	sub, err := nc.SubscribeSync("payment.succeeded")
	if err != nil {
		t.Fatalf("SubscribeSync() error = %v", err)
	}
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	event := payment.PaymentEvent{
		PaymentID: "it-pay-nats", OrderID: "it-order-nats", CustomerID: "it-user-nats",
		AmountKZT: 5000, Status: payment.StatusSucceeded, OccurredAt: time.Now().UTC(),
	}
	if err := NewNATSPublisher(nc).PublishPaymentSucceeded(context.Background(), event); err != nil {
		t.Fatalf("PublishPaymentSucceeded() error = %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	msg, err := sub.NextMsg(3 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg() error = %v", err)
	}
	var got payment.PaymentEvent
	if err := json.Unmarshal(msg.Data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.PaymentID != event.PaymentID || got.Status != payment.StatusSucceeded {
		t.Fatalf("event = %+v, want payment_id=%s status=%s", got, event.PaymentID, payment.StatusSucceeded)
	}
}
