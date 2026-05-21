package grpcapi

import (
	"context"
	"time"

	paymentv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/payment/v1"
	"kazakhexpress/payment-service/internal/payment"
)

type Server struct {
	paymentv1.UnimplementedPaymentServiceServer
	service *payment.Service
}

func NewServer(service *payment.Service) *Server {
	return &Server{service: service}
}

func (s *Server) CreatePayment(ctx context.Context, input *paymentv1.CreatePaymentRequest) (*paymentv1.Payment, error) {
	result, err := s.service.Create(ctx, payment.CreateInput{
		OrderID:        input.GetOrderId(),
		CustomerID:     input.GetCustomerId(),
		CustomerEmail:  input.GetCustomerEmail(),
		AmountKZT:      input.GetAmountKzt(),
		Method:         methodFromProto(input.GetMethod()),
		IdempotencyKey: input.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, err
	}
	return paymentToProto(result), nil
}

func (s *Server) GetPayment(ctx context.Context, input *paymentv1.GetPaymentRequest) (*paymentv1.Payment, error) {
	result, err := s.service.GetByID(ctx, input.GetPaymentId())
	if err != nil {
		return nil, err
	}
	return paymentToProto(result), nil
}

func (s *Server) GetPaymentByOrderID(ctx context.Context, input *paymentv1.GetPaymentByOrderIDRequest) (*paymentv1.Payment, error) {
	result, err := s.service.GetByOrderID(ctx, input.GetOrderId())
	if err != nil {
		return nil, err
	}
	return paymentToProto(result), nil
}

func (s *Server) ListPayments(ctx context.Context, input *paymentv1.ListPaymentsRequest) (*paymentv1.ListPaymentsResponse, error) {
	var (
		result []payment.Payment
		err    error
	)
	if input.GetCustomerId() != "" {
		result, err = s.service.ListByCustomerID(ctx, input.GetCustomerId())
	} else {
		result, err = s.service.List(ctx)
	}
	if err != nil {
		return nil, err
	}
	payments := make([]*paymentv1.Payment, 0, len(result))
	for _, item := range result {
		payments = append(payments, paymentToProto(item))
	}
	return &paymentv1.ListPaymentsResponse{Payments: payments}, nil
}

func (s *Server) RefundPayment(ctx context.Context, input *paymentv1.RefundPaymentRequest) (*paymentv1.Payment, error) {
	result, err := s.service.Refund(ctx, payment.RefundInput{
		PaymentID: input.GetPaymentId(),
		Reason:    input.GetReason(),
	})
	if err != nil {
		return nil, err
	}
	return paymentToProto(result), nil
}

func (s *Server) ConfirmPayment(ctx context.Context, input *paymentv1.ConfirmPaymentRequest) (*paymentv1.Payment, error) {
	result, err := s.service.Confirm(ctx, payment.ConfirmInput{
		PaymentID:             input.GetPaymentId(),
		ProviderTransactionID: input.GetProviderTransactionId(),
	})
	if err != nil {
		return nil, err
	}
	return paymentToProto(result), nil
}

func (s *Server) CancelPayment(ctx context.Context, input *paymentv1.CancelPaymentRequest) (*paymentv1.Payment, error) {
	result, err := s.service.Cancel(ctx, payment.CancelInput{
		PaymentID: input.GetPaymentId(),
		Reason:    input.GetReason(),
	})
	if err != nil {
		return nil, err
	}
	return paymentToProto(result), nil
}

func paymentToProto(item payment.Payment) *paymentv1.Payment {
	return &paymentv1.Payment{
		Id:                    item.ID,
		OrderId:               item.OrderID,
		CustomerId:            item.CustomerID,
		CustomerEmail:         item.CustomerEmail,
		AmountKzt:             item.AmountKZT,
		Method:                methodToProto(item.Method),
		Status:                statusToProto(item.Status),
		ProviderTransactionId: item.ProviderTransactionID,
		IdempotencyKey:        item.IdempotencyKey,
		RefundReason:          item.RefundReason,
		FailureReason:         item.FailureReason,
		CreatedAt:             formatTime(item.CreatedAt),
		UpdatedAt:             formatTime(item.UpdatedAt),
	}
}

func methodFromProto(method paymentv1.PaymentMethod) payment.Method {
	switch method {
	case paymentv1.PaymentMethod_PAYMENT_METHOD_CARD:
		return payment.MethodCard
	case paymentv1.PaymentMethod_PAYMENT_METHOD_KASPI:
		return payment.MethodKaspi
	case paymentv1.PaymentMethod_PAYMENT_METHOD_WALLET:
		return payment.MethodWallet
	default:
		return ""
	}
}

func methodToProto(method payment.Method) paymentv1.PaymentMethod {
	switch method {
	case payment.MethodCard:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_CARD
	case payment.MethodKaspi:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_KASPI
	case payment.MethodWallet:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_WALLET
	default:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

func statusToProto(status payment.Status) paymentv1.PaymentStatus {
	switch status {
	case payment.StatusPending:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING
	case payment.StatusSucceeded:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_SUCCEEDED
	case payment.StatusFailed:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED
	case payment.StatusRefunded:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_REFUNDED
	case payment.StatusCancelled:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_CANCELLED
	default:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
