package orderservice

import (
	"context"
	"fmt"

	orderv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client interface {
	Health(ctx context.Context) error
	CreateOrder(ctx context.Context, input CreateOrderRequest) (Order, error)
	GetOrder(ctx context.Context, orderID string) (Order, error)
	ListOrders(ctx context.Context) ([]Order, error)
	UpdateOrderStatus(ctx context.Context, input UpdateOrderStatusRequest) (Order, error)
	CancelOrder(ctx context.Context, input CancelOrderRequest) (Order, error)
}

type GRPCClient struct {
	conn   *grpc.ClientConn
	client orderv1.OrderServiceClient
}

func NewGRPCClient(target string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create order grpc client: %w", err)
	}
	return &GRPCClient{conn: conn, client: orderv1.NewOrderServiceClient(conn)}, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

func (c *GRPCClient) Health(ctx context.Context) error {
	_, err := c.client.HealthCheck(ctx, &orderv1.HealthCheckRequest{})
	return err
}

func (c *GRPCClient) CreateOrder(ctx context.Context, input CreateOrderRequest) (Order, error) {
	output, err := c.client.CreateOrder(ctx, &orderv1.CreateOrderRequest{CustomerId: input.CustomerID, Items: itemsToProto(input.Items)})
	return orderFromProto(output), err
}

func (c *GRPCClient) GetOrder(ctx context.Context, orderID string) (Order, error) {
	output, err := c.client.GetOrder(ctx, &orderv1.GetOrderRequest{OrderId: orderID})
	return orderFromProto(output), err
}

func (c *GRPCClient) ListOrders(ctx context.Context) ([]Order, error) {
	output, err := c.client.ListOrders(ctx, &orderv1.ListOrdersRequest{})
	if err != nil {
		return nil, err
	}
	orders := make([]Order, 0, len(output.GetOrders()))
	for _, item := range output.GetOrders() {
		orders = append(orders, orderFromProto(item))
	}
	return orders, nil
}

func (c *GRPCClient) UpdateOrderStatus(ctx context.Context, input UpdateOrderStatusRequest) (Order, error) {
	output, err := c.client.UpdateOrderStatus(ctx, &orderv1.UpdateOrderStatusRequest{OrderId: input.OrderID, Status: statusToProto(input.Status)})
	return orderFromProto(output), err
}

func (c *GRPCClient) CancelOrder(ctx context.Context, input CancelOrderRequest) (Order, error) {
	output, err := c.client.CancelOrder(ctx, &orderv1.CancelOrderRequest{OrderId: input.OrderID, Reason: input.Reason})
	return orderFromProto(output), err
}

func itemsToProto(items []Item) []*orderv1.OrderItem {
	out := make([]*orderv1.OrderItem, 0, len(items))
	for _, item := range items {
		out = append(out, &orderv1.OrderItem{ProductId: item.ProductID, Name: item.Name, Quantity: int32(item.Quantity), PriceKzt: item.PriceKZT})
	}
	return out
}

func itemsFromProto(items []*orderv1.OrderItem) []Item {
	out := make([]Item, 0, len(items))
	for _, item := range items {
		out = append(out, Item{ProductID: item.GetProductId(), Name: item.GetName(), Quantity: int(item.GetQuantity()), PriceKZT: item.GetPriceKzt()})
	}
	return out
}

func orderFromProto(item *orderv1.Order) Order {
	if item == nil {
		return Order{}
	}
	return Order{ID: item.GetId(), CustomerID: item.GetCustomerId(), Items: itemsFromProto(item.GetItems()), Status: statusFromProto(item.GetStatus()), TotalKZT: item.GetTotalKzt(), CreatedAt: item.GetCreatedAt(), UpdatedAt: item.GetUpdatedAt()}
}

func statusToProto(status Status) orderv1.OrderStatus {
	switch status {
	case "paid":
		return orderv1.OrderStatus_ORDER_STATUS_PAID
	case "payment_failed":
		return orderv1.OrderStatus_ORDER_STATUS_PAYMENT_FAILED
	case "shipped":
		return orderv1.OrderStatus_ORDER_STATUS_SHIPPED
	case "completed":
		return orderv1.OrderStatus_ORDER_STATUS_COMPLETED
	case "canceled":
		return orderv1.OrderStatus_ORDER_STATUS_CANCELED
	default:
		return orderv1.OrderStatus_ORDER_STATUS_CREATED
	}
}

func statusFromProto(status orderv1.OrderStatus) Status {
	switch status {
	case orderv1.OrderStatus_ORDER_STATUS_PAID:
		return "paid"
	case orderv1.OrderStatus_ORDER_STATUS_PAYMENT_FAILED:
		return "payment_failed"
	case orderv1.OrderStatus_ORDER_STATUS_SHIPPED:
		return "shipped"
	case orderv1.OrderStatus_ORDER_STATUS_COMPLETED:
		return "completed"
	case orderv1.OrderStatus_ORDER_STATUS_CANCELED:
		return "canceled"
	default:
		return "created"
	}
}
