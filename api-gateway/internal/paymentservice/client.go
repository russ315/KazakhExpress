package paymentservice

import (
	"context"
	"fmt"

	paymentv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client interface {
	CreatePayment(ctx context.Context, input CreatePaymentRequest) (Payment, error)
	GetPayment(ctx context.Context, paymentID string) (Payment, error)
	GetPaymentByOrderID(ctx context.Context, orderID string) (Payment, error)
	ListPayments(ctx context.Context, customerID string) ([]Payment, error)
	RefundPayment(ctx context.Context, input RefundPaymentRequest) (Payment, error)
	ConfirmPayment(ctx context.Context, input ConfirmPaymentRequest) (Payment, error)
	CancelPayment(ctx context.Context, input CancelPaymentRequest) (Payment, error)
}

type GRPCClient struct {
	conn   *grpc.ClientConn
	client paymentv1.PaymentServiceClient
}

func NewGRPCClient(target string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create payment grpc client: %w", err)
	}
	return &GRPCClient{
		conn:   conn,
		client: paymentv1.NewPaymentServiceClient(conn),
	}, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

func (c *GRPCClient) CreatePayment(ctx context.Context, input CreatePaymentRequest) (Payment, error) {
	output, err := c.client.CreatePayment(ctx, &paymentv1.CreatePaymentRequest{
		OrderId:        input.OrderID,
		CustomerId:     input.CustomerID,
		CustomerEmail:  input.CustomerEmail,
		AmountKzt:      input.AmountKZT,
		Method:         methodToProto(input.Method),
		IdempotencyKey: input.IdempotencyKey,
	})
	return paymentFromProto(output), err
}

func (c *GRPCClient) GetPayment(ctx context.Context, paymentID string) (Payment, error) {
	output, err := c.client.GetPayment(ctx, &paymentv1.GetPaymentRequest{PaymentId: paymentID})
	return paymentFromProto(output), err
}

func (c *GRPCClient) GetPaymentByOrderID(ctx context.Context, orderID string) (Payment, error) {
	output, err := c.client.GetPaymentByOrderID(ctx, &paymentv1.GetPaymentByOrderIDRequest{OrderId: orderID})
	return paymentFromProto(output), err
}

func (c *GRPCClient) ListPayments(ctx context.Context, customerID string) ([]Payment, error) {
	output, err := c.client.ListPayments(ctx, &paymentv1.ListPaymentsRequest{CustomerId: customerID})
	if err != nil {
		return nil, err
	}
	payments := make([]Payment, 0, len(output.GetPayments()))
	for _, item := range output.GetPayments() {
		payments = append(payments, paymentFromProto(item))
	}
	return payments, nil
}

func (c *GRPCClient) RefundPayment(ctx context.Context, input RefundPaymentRequest) (Payment, error) {
	output, err := c.client.RefundPayment(ctx, &paymentv1.RefundPaymentRequest{
		PaymentId: input.PaymentID,
		Reason:    input.Reason,
	})
	return paymentFromProto(output), err
}

func (c *GRPCClient) ConfirmPayment(ctx context.Context, input ConfirmPaymentRequest) (Payment, error) {
	output, err := c.client.ConfirmPayment(ctx, &paymentv1.ConfirmPaymentRequest{
		PaymentId:             input.PaymentID,
		ProviderTransactionId: input.ProviderTransactionID,
	})
	return paymentFromProto(output), err
}

func (c *GRPCClient) CancelPayment(ctx context.Context, input CancelPaymentRequest) (Payment, error) {
	output, err := c.client.CancelPayment(ctx, &paymentv1.CancelPaymentRequest{
		PaymentId: input.PaymentID,
		Reason:    input.Reason,
	})
	return paymentFromProto(output), err
}

func methodToProto(method string) paymentv1.PaymentMethod {
	switch method {
	case "card":
		return paymentv1.PaymentMethod_PAYMENT_METHOD_CARD
	case "kaspi":
		return paymentv1.PaymentMethod_PAYMENT_METHOD_KASPI
	case "wallet":
		return paymentv1.PaymentMethod_PAYMENT_METHOD_WALLET
	default:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

func methodFromProto(method paymentv1.PaymentMethod) string {
	switch method {
	case paymentv1.PaymentMethod_PAYMENT_METHOD_CARD:
		return "card"
	case paymentv1.PaymentMethod_PAYMENT_METHOD_KASPI:
		return "kaspi"
	case paymentv1.PaymentMethod_PAYMENT_METHOD_WALLET:
		return "wallet"
	default:
		return ""
	}
}

func statusFromProto(status paymentv1.PaymentStatus) string {
	switch status {
	case paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING:
		return "pending"
	case paymentv1.PaymentStatus_PAYMENT_STATUS_SUCCEEDED:
		return "succeeded"
	case paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED:
		return "failed"
	case paymentv1.PaymentStatus_PAYMENT_STATUS_REFUNDED:
		return "refunded"
	case paymentv1.PaymentStatus_PAYMENT_STATUS_CANCELLED:
		return "cancelled"
	default:
		return ""
	}
}

func paymentFromProto(item *paymentv1.Payment) Payment {
	if item == nil {
		return Payment{}
	}
	return Payment{
		ID:                    item.GetId(),
		OrderID:               item.GetOrderId(),
		CustomerID:            item.GetCustomerId(),
		CustomerEmail:         item.GetCustomerEmail(),
		AmountKZT:             item.GetAmountKzt(),
		Method:                methodFromProto(item.GetMethod()),
		Status:                statusFromProto(item.GetStatus()),
		ProviderTransactionID: item.GetProviderTransactionId(),
		IdempotencyKey:        item.GetIdempotencyKey(),
		RefundReason:          item.GetRefundReason(),
		FailureReason:         item.GetFailureReason(),
		CreatedAt:             item.GetCreatedAt(),
		UpdatedAt:             item.GetUpdatedAt(),
	}
}
