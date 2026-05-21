package grpcapi

import (
	"context"

	"kazakhexpress/order-service/internal/order"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

const ServiceName = "kazakhexpress.order.v1.OrderService"

type Server struct {
	service *order.Service
}

type OrderServer interface {
	CreateOrder(context.Context, *order.CreateInput) (*order.Order, error)
	GetOrder(context.Context, *GetOrderRequest) (*order.Order, error)
	ListOrders(context.Context, *ListOrdersRequest) (*ListOrdersResponse, error)
	UpdateOrderStatus(context.Context, *UpdateOrderStatusRequest) (*order.Order, error)
	CancelOrder(context.Context, *CancelOrderRequest) (*order.Order, error)
}

type GetOrderRequest struct {
	OrderID string `json:"order_id"`
}

type ListOrdersRequest struct{}

type ListOrdersResponse struct {
	Orders []order.Order `json:"orders"`
}

type UpdateOrderStatusRequest struct {
	OrderID string       `json:"order_id"`
	Status  order.Status `json:"status"`
}

type CancelOrderRequest struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

func init() {
	encoding.RegisterCodec(jsonCodec{})
}

func NewServer(service *order.Service) *Server {
	return &Server{service: service}
}

func Register(server *grpc.Server, service *Server) {
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: ServiceName,
		HandlerType: (*OrderServer)(nil),
		Methods: []grpc.MethodDesc{
			{MethodName: "CreateOrder", Handler: createOrderHandler},
			{MethodName: "GetOrder", Handler: getOrderHandler},
			{MethodName: "ListOrders", Handler: listOrdersHandler},
			{MethodName: "UpdateOrderStatus", Handler: updateOrderStatusHandler},
			{MethodName: "CancelOrder", Handler: cancelOrderHandler},
		},
	}, service)
}

func (s *Server) CreateOrder(ctx context.Context, input *order.CreateInput) (*order.Order, error) {
	result, err := s.service.Create(ctx, *input)
	return &result, err
}

func (s *Server) GetOrder(ctx context.Context, input *GetOrderRequest) (*order.Order, error) {
	result, err := s.service.GetByID(ctx, input.OrderID)
	return &result, err
}

func (s *Server) ListOrders(ctx context.Context, input *ListOrdersRequest) (*ListOrdersResponse, error) {
	result, err := s.service.List(ctx)
	return &ListOrdersResponse{Orders: result}, err
}

func (s *Server) UpdateOrderStatus(ctx context.Context, input *UpdateOrderStatusRequest) (*order.Order, error) {
	result, err := s.service.UpdateStatus(ctx, input.OrderID, input.Status)
	return &result, err
}

func (s *Server) CancelOrder(ctx context.Context, input *CancelOrderRequest) (*order.Order, error) {
	result, err := s.service.Cancel(ctx, input.OrderID, input.Reason)
	return &result, err
}

func createOrderHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	input := new(order.CreateInput)
	if err := dec(input); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).CreateOrder(ctx, input)
	}
	return interceptor(ctx, input, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/" + ServiceName + "/CreateOrder"}, func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).CreateOrder(ctx, req.(*order.CreateInput))
	})
}

func getOrderHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	input := new(GetOrderRequest)
	if err := dec(input); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).GetOrder(ctx, input)
	}
	return interceptor(ctx, input, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/" + ServiceName + "/GetOrder"}, func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).GetOrder(ctx, req.(*GetOrderRequest))
	})
}

func listOrdersHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	input := new(ListOrdersRequest)
	if err := dec(input); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).ListOrders(ctx, input)
	}
	return interceptor(ctx, input, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/" + ServiceName + "/ListOrders"}, func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).ListOrders(ctx, req.(*ListOrdersRequest))
	})
}

func updateOrderStatusHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	input := new(UpdateOrderStatusRequest)
	if err := dec(input); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).UpdateOrderStatus(ctx, input)
	}
	return interceptor(ctx, input, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/" + ServiceName + "/UpdateOrderStatus"}, func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).UpdateOrderStatus(ctx, req.(*UpdateOrderStatusRequest))
	})
}

func cancelOrderHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	input := new(CancelOrderRequest)
	if err := dec(input); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).CancelOrder(ctx, input)
	}
	return interceptor(ctx, input, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/" + ServiceName + "/CancelOrder"}, func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).CancelOrder(ctx, req.(*CancelOrderRequest))
	})
}
