package grpcapi

import (
	"testing"

	"kazakhexpress/order-service/internal/order"
)

func TestJSONCodecName(t *testing.T) {
	codec := jsonCodec{}
	if codec.Name() != "json" {
		t.Fatalf("name = %q, want json", codec.Name())
	}
}

func TestJSONCodecMarshalUnmarshal(t *testing.T) {
	codec := jsonCodec{}
	input := order.Order{
		ID:         "ord-1",
		CustomerID: "customer-1",
		Status:     order.StatusCreated,
		TotalKZT:   1000,
	}

	data, err := codec.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var output order.Order
	if err := codec.Unmarshal(data, &output); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if output.ID != input.ID || output.CustomerID != input.CustomerID {
		t.Fatalf("output = %+v, want %+v", output, input)
	}
}
