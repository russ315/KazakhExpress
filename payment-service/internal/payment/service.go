package payment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid payment input")

type EventPublisher interface {
	PublishPaymentCreated(ctx context.Context, event PaymentEvent) error
	PublishPaymentRefunded(ctx context.Context, event PaymentEvent) error
}

type EmailSender interface {
	SendReceipt(ctx context.Context, email ReceiptEmail) error
	SendRefund(ctx context.Context, email RefundEmail) error
}

type Service struct {
	repo      Repository
	publisher EventPublisher
	emailer   EmailSender
}

func NewService(repo Repository, publisher EventPublisher, emailer EmailSender) *Service {
	return &Service{repo: repo, publisher: publisher, emailer: emailer}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Payment, error) {
	if input.OrderID == "" || input.CustomerID == "" || input.CustomerEmail == "" || input.AmountKZT <= 0 || !isAllowedMethod(input.Method) {
		return Payment{}, ErrInvalidInput
	}

	now := time.Now().UTC()
	p := Payment{
		ID:            "pay-" + uuid.NewString(),
		OrderID:       input.OrderID,
		CustomerID:    input.CustomerID,
		CustomerEmail: input.CustomerEmail,
		AmountKZT:     input.AmountKZT,
		Method:        input.Method,
		Status:        StatusPaid,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return Payment{}, err
	}

	event := paymentEvent(created)
	if err := s.publisher.PublishPaymentCreated(ctx, event); err != nil {
		return Payment{}, err
	}
	if err := s.emailer.SendReceipt(ctx, ReceiptEmail{
		To:        created.CustomerEmail,
		PaymentID: created.ID,
		OrderID:   created.OrderID,
		AmountKZT: created.AmountKZT,
	}); err != nil {
		return Payment{}, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (Payment, error) {
	if id == "" {
		return Payment{}, ErrInvalidInput
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Payment, error) {
	return s.repo.List(ctx)
}

func (s *Service) Refund(ctx context.Context, input RefundInput) (Payment, error) {
	if input.PaymentID == "" || input.Reason == "" {
		return Payment{}, ErrInvalidInput
	}

	p, err := s.repo.GetByID(ctx, input.PaymentID)
	if err != nil {
		return Payment{}, err
	}
	if p.Status == StatusRefunded {
		return Payment{}, ErrInvalidInput
	}

	p.Status = StatusRefunded
	p.RefundReason = input.Reason
	p.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		return Payment{}, err
	}

	event := paymentEvent(updated)
	if err := s.publisher.PublishPaymentRefunded(ctx, event); err != nil {
		return Payment{}, err
	}
	if err := s.emailer.SendRefund(ctx, RefundEmail{
		To:        updated.CustomerEmail,
		PaymentID: updated.ID,
		OrderID:   updated.OrderID,
		AmountKZT: updated.AmountKZT,
		Reason:    updated.RefundReason,
	}); err != nil {
		return Payment{}, err
	}

	return updated, nil
}

func paymentEvent(p Payment) PaymentEvent {
	return PaymentEvent{
		PaymentID:  p.ID,
		OrderID:    p.OrderID,
		CustomerID: p.CustomerID,
		AmountKZT:  p.AmountKZT,
		Status:     p.Status,
	}
}

func isAllowedMethod(method Method) bool {
	switch method {
	case MethodCard, MethodKaspi, MethodWallet:
		return true
	default:
		return false
	}
}
