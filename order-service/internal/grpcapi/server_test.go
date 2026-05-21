package grpcapi

import (
	"context"
	"errors"
	"testing"

	"kazakhexpress/order-service/internal/order"
)

type grpcTestRepo struct {
	orders map[string]order.Order
}

func newGRPCTestRepo() *grpcTestRepo {
	return &grpcTestRepo{orders: make(map[string]order.Order)}
}

func (r *grpcTestRepo) Create(ctx context.Context, o order.Order) (order.Order, error) {
	r.orders[o.ID] = o
	return o, nil
}

func (r *grpcTestRepo) List(ctx context.Context) ([]order.Order, error) {
	orders := make([]order.Order, 0, len(r.orders))
	for _, o := range r.orders {
		orders = append(orders, o)
	}
	return orders, nil
}

func (r *grpcTestRepo) GetByID(ctx context.Context, id string) (order.Order, error) {
	o, ok := r.orders[id]
	if !ok {
		return order.Order{}, order.ErrNotFound
	}
	return o, nil
}

func (r *grpcTestRepo) UpdateStatus(ctx context.Context, id string, from order.Status, to order.Status, reason string) (order.Order, error) {
	o, ok := r.orders[id]
	if !ok {
		return order.Order{}, order.ErrNotFound
	}
	o.Status = to
	r.orders[id] = o
	return o, nil
}

func newTestServer() *Server {
	return NewServer(order.NewService(newGRPCTestRepo(), nil, nil))
}

func TestGRPCCreateAndGetOrder(t *testing.T) {
	server := newTestServer()
	ctx := context.Background()

	created, err := server.CreateOrder(ctx, &order.CreateInput{
		CustomerID: "customer-1",
		Items: []order.Item{{
			ProductID: "product-1",
			Name:      "Shapan",
			Quantity:  1,
			PriceKZT:  1500,
		}},
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	found, err := server.GetOrder(ctx, &GetOrderRequest{OrderID: created.ID})
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if found.ID != created.ID {
		t.Fatalf("order id = %s, want %s", found.ID, created.ID)
	}
}

func TestGRPCListOrders(t *testing.T) {
	server := newTestServer()
	ctx := context.Background()

	if _, err := server.CreateOrder(ctx, &order.CreateInput{
		CustomerID: "customer-1",
		Items:      []order.Item{{ProductID: "p1", Name: "x", Quantity: 1, PriceKZT: 100}},
	}); err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	resp, err := server.ListOrders(ctx, &ListOrdersRequest{})
	if err != nil {
		t.Fatalf("ListOrders() error = %v", err)
	}
	if len(resp.Orders) != 1 {
		t.Fatalf("len(orders) = %d, want 1", len(resp.Orders))
	}
}

func TestGRPCUpdateAndCancelOrder(t *testing.T) {
	server := newTestServer()
	ctx := context.Background()

	created, err := server.CreateOrder(ctx, &order.CreateInput{
		CustomerID: "customer-1",
		Items:      []order.Item{{ProductID: "p1", Name: "x", Quantity: 1, PriceKZT: 100}},
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	updated, err := server.UpdateOrderStatus(ctx, &UpdateOrderStatusRequest{
		OrderID: created.ID,
		Status:  order.StatusShipped,
	})
	if err != nil {
		t.Fatalf("UpdateOrderStatus() error = %v", err)
	}
	if updated.Status != order.StatusShipped {
		t.Fatalf("status = %s, want %s", updated.Status, order.StatusShipped)
	}

	cancelled, err := server.CancelOrder(ctx, &CancelOrderRequest{
		OrderID: created.ID,
		Reason:  "grpc cancel",
	})
	if err != nil {
		t.Fatalf("CancelOrder() error = %v", err)
	}
	if cancelled.Status != order.StatusCanceled {
		t.Fatalf("status = %s, want %s", cancelled.Status, order.StatusCanceled)
	}
}

func TestGRPCGetOrderNotFound(t *testing.T) {
	server := newTestServer()
	_, err := server.GetOrder(context.Background(), &GetOrderRequest{OrderID: "missing"})
	if !errors.Is(err, order.ErrNotFound) {
		t.Fatalf("GetOrder() error = %v, want %v", err, order.ErrNotFound)
	}
}
