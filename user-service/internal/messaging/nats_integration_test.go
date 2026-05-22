//go:build integration

package messaging

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestPublisherPublishesUserEvent(t *testing.T) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		t.Skip("NATS_URL is required for integration tests")
	}
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("nats.Connect() error = %v", err)
	}
	t.Cleanup(nc.Close)

	sub, err := nc.SubscribeSync("user.created")
	if err != nil {
		t.Fatalf("SubscribeSync() error = %v", err)
	}
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	event := UserEvent{UserID: "it-user-nats", Email: "it@example.com", FirstName: "It", LastName: "User", Event: "user.created", Timestamp: time.Now().UTC()}
	if err := NewPublisher(nc).PublishUserEvent(context.Background(), event); err != nil {
		t.Fatalf("PublishUserEvent() error = %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	msg, err := sub.NextMsg(3 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg() error = %v", err)
	}
	var got UserEvent
	if err := json.Unmarshal(msg.Data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.UserID != event.UserID || got.Event != event.Event {
		t.Fatalf("event = %+v", got)
	}
}
