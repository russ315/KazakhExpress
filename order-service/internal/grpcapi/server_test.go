package grpcapi

import (
	"context"
	"errors"
	"testing"

	"kazakhexpress/order-service/internal/order"

	orderv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/order/v1"
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

	created, err := server.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		CustomerId: "customer-1",
		Items: []*orderv1.OrderItem{{
			ProductId: "product-1",
			Name:      "Shapan",
			Quantity:  1,
			PriceKzt:  1500,
		}},
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	found, err := server.GetOrder(ctx, &orderv1.GetOrderRequest{OrderId: created.Id})
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if found.Id != created.Id {
		t.Fatalf("order id = %s, want %s", found.Id, created.Id)
	}
}

func TestGRPCListOrders(t *testing.T) {
	server := newTestServer()
	ctx := context.Background()

	if _, err := server.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		CustomerId: "customer-1",
		Items:      []*orderv1.OrderItem{{ProductId: "p1", Name: "x", Quantity: 1, PriceKzt: 100}},
	}); err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	resp, err := server.ListOrders(ctx, &orderv1.ListOrdersRequest{})
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

	created, err := server.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		CustomerId: "customer-1",
		Items:      []*orderv1.OrderItem{{ProductId: "p1", Name: "x", Quantity: 1, PriceKzt: 100}},
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	updated, err := server.UpdateOrderStatus(ctx, &orderv1.UpdateOrderStatusRequest{
		OrderId: created.Id,
		Status:  orderv1.OrderStatus_ORDER_STATUS_SHIPPED,
	})
	if err != nil {
		t.Fatalf("UpdateOrderStatus() error = %v", err)
	}
	if updated.Status != orderv1.OrderStatus_ORDER_STATUS_SHIPPED {
		t.Fatalf("status = %s, want %s", updated.Status, orderv1.OrderStatus_ORDER_STATUS_SHIPPED)
	}

	cancelled, err := server.CancelOrder(ctx, &orderv1.CancelOrderRequest{
		OrderId: created.Id,
		Reason:  "grpc cancel",
	})
	if err != nil {
		t.Fatalf("CancelOrder() error = %v", err)
	}
	if cancelled.Status != orderv1.OrderStatus_ORDER_STATUS_CANCELED {
		t.Fatalf("status = %s, want %s", cancelled.Status, orderv1.OrderStatus_ORDER_STATUS_CANCELED)
	}
}

func TestGRPCGetOrderNotFound(t *testing.T) {
	server := newTestServer()
	_, err := server.GetOrder(context.Background(), &orderv1.GetOrderRequest{OrderId: "missing"})
	if !errors.Is(err, order.ErrNotFound) {
		t.Fatalf("GetOrder() error = %v, want %v", err, order.ErrNotFound)
	}
}
