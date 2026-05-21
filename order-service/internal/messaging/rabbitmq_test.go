package messaging

import (
	"encoding/json"
	"testing"
	"time"

	"kazakhexpress/order-service/internal/order"
)

func TestOrderEventJSONRoundTrip(t *testing.T) {
	event := order.Event{
		OrderID:    "ord-1",
		CustomerID: "customer-1",
		Status:     order.StatusCreated,
		TotalKZT:   2500,
		Reason:     "created",
		OccurredAt: time.Now().UTC(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded order.Event
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.OrderID != event.OrderID || decoded.CustomerID != event.CustomerID {
		t.Fatalf("decoded = %+v, want %+v", decoded, event)
	}
	if decoded.Status != event.Status || decoded.TotalKZT != event.TotalKZT {
		t.Fatalf("decoded fields mismatch: %+v", decoded)
	}
}

func TestPaymentEventJSONRoundTrip(t *testing.T) {
	event := order.PaymentEvent{
		PaymentID:  "pay-1",
		OrderID:    "ord-1",
		CustomerID: "customer-1",
		Status:     "succeeded",
		Reason:     "",
		OccurredAt: time.Now().UTC(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded order.PaymentEvent
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.PaymentID != event.PaymentID || decoded.OrderID != event.OrderID {
		t.Fatalf("decoded = %+v, want %+v", decoded, event)
	}
}

func TestStockReservedEventJSONRoundTrip(t *testing.T) {
	event := order.StockReservedEvent{
		OrderID:    "ord-1",
		CustomerID: "customer-1",
		OccurredAt: time.Now().UTC(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded order.StockReservedEvent
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.OrderID != event.OrderID {
		t.Fatalf("order_id = %s, want %s", decoded.OrderID, event.OrderID)
	}
}

func TestDefaultExchangeConstant(t *testing.T) {
	if defaultExchange != "kazakhexpress.events" {
		t.Fatalf("defaultExchange = %q, want kazakhexpress.events", defaultExchange)
	}
}
