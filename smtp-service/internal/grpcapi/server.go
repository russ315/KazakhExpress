package grpcapi

import (
	"context"

	smtpservice "kazakhexpress/smtp-service/internal/smtp"

	smtpv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/smtp/v1"
)

type Server struct {
	smtpv1.UnimplementedSMTPServiceServer
	service *smtpservice.Service
}

func NewServer(service *smtpservice.Service) *Server {
	return &Server{service: service}
}

func (s *Server) SendEmail(ctx context.Context, input *smtpv1.SendEmailRequest) (*smtpv1.SendEmailResponse, error) {
	if err := s.service.SendEmail(ctx, input.GetTo(), input.GetSubject(), input.GetBody()); err != nil {
		return nil, err
	}
	return &smtpv1.SendEmailResponse{Accepted: true}, nil
}

func (s *Server) SendPaymentReceipt(ctx context.Context, input *smtpv1.PaymentReceiptRequest) (*smtpv1.SendEmailResponse, error) {
	return s.SendEmail(ctx, &smtpv1.SendEmailRequest{
		To:      input.GetTo(),
		Subject: smtpservice.PaymentReceiptSubject(),
		Body:    smtpservice.PaymentReceiptBody(input.GetPaymentId(), input.GetOrderId(), input.GetAmountKzt()),
	})
}

func (s *Server) SendPaymentRefund(ctx context.Context, input *smtpv1.PaymentRefundRequest) (*smtpv1.SendEmailResponse, error) {
	return s.SendEmail(ctx, &smtpv1.SendEmailRequest{
		To:      input.GetTo(),
		Subject: smtpservice.PaymentRefundSubject(),
		Body:    smtpservice.PaymentRefundBody(input.GetPaymentId(), input.GetOrderId(), input.GetAmountKzt(), input.GetReason()),
	})
}

func (s *Server) SendPaymentFailure(ctx context.Context, input *smtpv1.PaymentFailureRequest) (*smtpv1.SendEmailResponse, error) {
	return s.SendEmail(ctx, &smtpv1.SendEmailRequest{
		To:      input.GetTo(),
		Subject: smtpservice.PaymentFailureSubject(),
		Body:    smtpservice.PaymentFailureBody(input.GetPaymentId(), input.GetOrderId(), input.GetAmountKzt(), input.GetReason()),
	})
}
