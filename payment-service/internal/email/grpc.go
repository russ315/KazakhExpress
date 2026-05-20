package email

import (
	"context"
	"fmt"

	smtpv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/smtp/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"kazakhexpress/payment-service/internal/payment"
)

type GRPCSender struct {
	conn   *grpc.ClientConn
	client smtpv1.SMTPServiceClient
}

func NewGRPCSender(target string) (*GRPCSender, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create smtp grpc client: %w", err)
	}
	return &GRPCSender{
		conn:   conn,
		client: smtpv1.NewSMTPServiceClient(conn),
	}, nil
}

func (s *GRPCSender) Close() error {
	return s.conn.Close()
}

func (s *GRPCSender) SendReceipt(ctx context.Context, email payment.ReceiptEmail) error {
	_, err := s.client.SendPaymentReceipt(ctx, &smtpv1.PaymentReceiptRequest{
		To:        email.To,
		PaymentId: email.PaymentID,
		OrderId:   email.OrderID,
		AmountKzt: email.AmountKZT,
	})
	return err
}

func (s *GRPCSender) SendRefund(ctx context.Context, email payment.RefundEmail) error {
	_, err := s.client.SendPaymentRefund(ctx, &smtpv1.PaymentRefundRequest{
		To:        email.To,
		PaymentId: email.PaymentID,
		OrderId:   email.OrderID,
		AmountKzt: email.AmountKZT,
		Reason:    email.Reason,
	})
	return err
}

func (s *GRPCSender) SendFailure(ctx context.Context, email payment.FailureEmail) error {
	_, err := s.client.SendPaymentFailure(ctx, &smtpv1.PaymentFailureRequest{
		To:        email.To,
		PaymentId: email.PaymentID,
		OrderId:   email.OrderID,
		AmountKzt: email.AmountKZT,
		Reason:    email.Reason,
	})
	return err
}
