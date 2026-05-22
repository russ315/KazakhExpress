//go:build integration

package messaging

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"kazakhexpress/review-service/internal/review"

	"github.com/nats-io/nats.go"
)

func TestNATSPublisherPublishesReviewCreated(t *testing.T) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		t.Skip("NATS_URL is required for integration tests")
	}
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("nats.Connect() error = %v", err)
	}
	t.Cleanup(nc.Close)

	sub, err := nc.SubscribeSync("review.created")
	if err != nil {
		t.Fatalf("SubscribeSync() error = %v", err)
	}
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	event := review.Event{ReviewID: "it-review-nats", ProductID: "it-product-nats", CustomerID: "it-user-nats", Rating: 5, OccurredAt: time.Now().UTC()}
	if err := NewNATSPublisher(nc).PublishReviewCreated(context.Background(), event); err != nil {
		t.Fatalf("PublishReviewCreated() error = %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	msg, err := sub.NextMsg(3 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg() error = %v", err)
	}
	var got review.Event
	if err := json.Unmarshal(msg.Data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.ReviewID != event.ReviewID || got.Rating != event.Rating {
		t.Fatalf("event = %+v", got)
	}
}
