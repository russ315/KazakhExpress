package grpcapi

import (
	"context"
	"time"

	"kazakhexpress/order-service/internal/order"

	orderv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/order/v1"
)

type Server struct {
	orderv1.UnimplementedOrderServiceServer
	service *order.Service
}

func NewServer(service *order.Service) *Server {
	return &Server{service: service}
}

func (s *Server) HealthCheck(ctx context.Context, req *orderv1.HealthCheckRequest) (*orderv1.HealthCheckResponse, error) {
	return &orderv1.HealthCheckResponse{Status: "ok"}, nil
}

func (s *Server) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.Order, error) {
	created, err := s.service.Create(ctx, order.CreateInput{CustomerID: req.GetCustomerId(), Items: itemsFromProto(req.GetItems())})
	if err != nil {
		return nil, err
	}
	return toProto(created), nil
}

func (s *Server) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.Order, error) {
	found, err := s.service.GetByID(ctx, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	return toProto(found), nil
}

func (s *Server) ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	orders, err := s.service.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*orderv1.Order, 0, len(orders))
	for _, item := range orders {
		out = append(out, toProto(item))
	}
	return &orderv1.ListOrdersResponse{Orders: out}, nil
}

func (s *Server) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.Order, error) {
	updated, err := s.service.UpdateStatus(ctx, req.GetOrderId(), statusFromProto(req.GetStatus()))
	if err != nil {
		return nil, err
	}
	return toProto(updated), nil
}

func (s *Server) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.Order, error) {
	cancelled, err := s.service.Cancel(ctx, req.GetOrderId(), req.GetReason())
	if err != nil {
		return nil, err
	}
	return toProto(cancelled), nil
}

func toProto(o order.Order) *orderv1.Order {
	return &orderv1.Order{
		Id:         o.ID,
		CustomerId: o.CustomerID,
		Items:      itemsToProto(o.Items),
		Status:     statusToProto(o.Status),
		TotalKzt:   o.TotalKZT,
		CreatedAt:  formatTime(o.CreatedAt),
		UpdatedAt:  formatTime(o.UpdatedAt),
	}
}

func itemsToProto(items []order.Item) []*orderv1.OrderItem {
	out := make([]*orderv1.OrderItem, 0, len(items))
	for _, item := range items {
		out = append(out, &orderv1.OrderItem{ProductId: item.ProductID, Name: item.Name, Quantity: int32(item.Quantity), PriceKzt: item.PriceKZT})
	}
	return out
}

func itemsFromProto(items []*orderv1.OrderItem) []order.Item {
	out := make([]order.Item, 0, len(items))
	for _, item := range items {
		out = append(out, order.Item{ProductID: item.GetProductId(), Name: item.GetName(), Quantity: int(item.GetQuantity()), PriceKZT: item.GetPriceKzt()})
	}
	return out
}

func statusToProto(status order.Status) orderv1.OrderStatus {
	switch status {
	case order.StatusCreated:
		return orderv1.OrderStatus_ORDER_STATUS_CREATED
	case order.StatusPaid:
		return orderv1.OrderStatus_ORDER_STATUS_PAID
	case order.StatusPaymentFailed:
		return orderv1.OrderStatus_ORDER_STATUS_PAYMENT_FAILED
	case order.StatusShipped:
		return orderv1.OrderStatus_ORDER_STATUS_SHIPPED
	case order.StatusCompleted:
		return orderv1.OrderStatus_ORDER_STATUS_COMPLETED
	case order.StatusCanceled:
		return orderv1.OrderStatus_ORDER_STATUS_CANCELED
	default:
		return orderv1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func statusFromProto(status orderv1.OrderStatus) order.Status {
	switch status {
	case orderv1.OrderStatus_ORDER_STATUS_PAID:
		return order.StatusPaid
	case orderv1.OrderStatus_ORDER_STATUS_PAYMENT_FAILED:
		return order.StatusPaymentFailed
	case orderv1.OrderStatus_ORDER_STATUS_SHIPPED:
		return order.StatusShipped
	case orderv1.OrderStatus_ORDER_STATUS_COMPLETED:
		return order.StatusCompleted
	case orderv1.OrderStatus_ORDER_STATUS_CANCELED:
		return order.StatusCanceled
	default:
		return order.StatusCreated
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
