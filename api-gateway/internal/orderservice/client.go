package orderservice

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
)

const serviceName = "kazakhexpress.order.v1.OrderService"

type Client interface {
	Health(ctx context.Context) error
	CreateOrder(ctx context.Context, input CreateOrderRequest) (Order, error)
	GetOrder(ctx context.Context, orderID string) (Order, error)
	ListOrders(ctx context.Context) ([]Order, error)
	UpdateOrderStatus(ctx context.Context, input UpdateOrderStatusRequest) (Order, error)
	CancelOrder(ctx context.Context, input CancelOrderRequest) (Order, error)
}

type GRPCClient struct {
	conn *grpc.ClientConn
}

func init() {
	encoding.RegisterCodec(jsonCodec{})
}

func NewGRPCClient(target string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(jsonCodec{})),
	)
	if err != nil {
		return nil, fmt.Errorf("create order grpc client: %w", err)
	}
	return &GRPCClient{conn: conn}, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

func (c *GRPCClient) Health(ctx context.Context) error {
	_, err := c.ListOrders(ctx)
	return err
}

func (c *GRPCClient) CreateOrder(ctx context.Context, input CreateOrderRequest) (Order, error) {
	var output Order
	err := c.conn.Invoke(ctx, fullMethod("CreateOrder"), input, &output)
	return output, err
}

func (c *GRPCClient) GetOrder(ctx context.Context, orderID string) (Order, error) {
	var output Order
	err := c.conn.Invoke(ctx, fullMethod("GetOrder"), GetOrderRequest{OrderID: orderID}, &output)
	return output, err
}

func (c *GRPCClient) ListOrders(ctx context.Context) ([]Order, error) {
	var output ListOrdersResponse
	err := c.conn.Invoke(ctx, fullMethod("ListOrders"), ListOrdersRequest{}, &output)
	return output.Orders, err
}

func (c *GRPCClient) UpdateOrderStatus(ctx context.Context, input UpdateOrderStatusRequest) (Order, error) {
	var output Order
	err := c.conn.Invoke(ctx, fullMethod("UpdateOrderStatus"), input, &output)
	return output, err
}

func (c *GRPCClient) CancelOrder(ctx context.Context, input CancelOrderRequest) (Order, error) {
	var output Order
	err := c.conn.Invoke(ctx, fullMethod("CancelOrder"), input, &output)
	return output, err
}

func fullMethod(name string) string {
	return "/" + serviceName + "/" + name
}
