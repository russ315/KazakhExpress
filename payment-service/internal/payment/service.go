package payment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid payment input")
var ErrInvalidState = errors.New("invalid payment state")

type EventPublisher interface {
	PublishPaymentCreated(ctx context.Context, event PaymentEvent) error
	PublishPaymentSucceeded(ctx context.Context, event PaymentEvent) error
	PublishPaymentFailed(ctx context.Context, event PaymentEvent) error
	PublishPaymentRefunded(ctx context.Context, event PaymentEvent) error
	PublishPaymentCancelled(ctx context.Context, event PaymentEvent) error
}

type EmailSender interface {
	SendReceipt(ctx context.Context, email ReceiptEmail) error
	SendRefund(ctx context.Context, email RefundEmail) error
	SendFailure(ctx context.Context, email FailureEmail) error
}

type IdempotencyStore interface {
	GetPaymentID(ctx context.Context, key string) (string, bool, error)
	SavePaymentID(ctx context.Context, key string, paymentID string) error
}

type PaymentProvider interface {
	Charge(ctx context.Context, payment Payment) (ProviderResult, error)
}

type Service struct {
	repo        Repository
	publisher   EventPublisher
	emailer     EmailSender
	idempotency IdempotencyStore
	provider    PaymentProvider
}

func NewService(repo Repository, publisher EventPublisher, emailer EmailSender, idempotency IdempotencyStore, provider PaymentProvider) *Service {
	return &Service{
		repo:        repo,
		publisher:   publisher,
		emailer:     emailer,
		idempotency: idempotency,
		provider:    provider,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Payment, error) {
	if input.OrderID == "" || input.CustomerID == "" || input.CustomerEmail == "" || input.AmountKZT <= 0 || !isAllowedMethod(input.Method) || input.IdempotencyKey == "" {
		return Payment{}, ErrInvalidInput
	}

	if paymentID, ok, err := s.idempotency.GetPaymentID(ctx, input.IdempotencyKey); err != nil {
		return Payment{}, err
	} else if ok {
		return s.repo.GetByID(ctx, paymentID)
	}

	now := time.Now().UTC()
	p := Payment{
		ID:             "pay-" + uuid.NewString(),
		OrderID:        input.OrderID,
		CustomerID:     input.CustomerID,
		CustomerEmail:  input.CustomerEmail,
		AmountKZT:      input.AmountKZT,
		Method:         input.Method,
		Status:         StatusPending,
		IdempotencyKey: input.IdempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return Payment{}, err
	}
	if err := s.idempotency.SavePaymentID(ctx, input.IdempotencyKey, created.ID); err != nil {
		return Payment{}, err
	}

	event := paymentEvent(created)
	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return Payment{}, err
	}
	if err := s.publisher.PublishPaymentCreated(ctx, event); err != nil {
		return Payment{}, err
	}

	result, err := s.provider.Charge(ctx, created)
	if err != nil {
		return Payment{}, err
	}
	created.ProviderTransactionID = result.ProviderTransactionID
	created.FailureReason = result.FailureReason
	switch result.Status {
	case StatusSucceeded:
		created.Status = StatusSucceeded
	case StatusFailed:
		created.Status = StatusFailed
	default:
		return Payment{}, ErrInvalidState
	}
	created.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.Update(ctx, created)
	if err != nil {
		return Payment{}, err
	}
	event = paymentEvent(updated)
	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return Payment{}, err
	}
	if updated.Status == StatusFailed {
		if err := s.publisher.PublishPaymentFailed(ctx, event); err != nil {
			return Payment{}, err
		}
		if err := s.emailer.SendFailure(ctx, FailureEmail{
			To:        updated.CustomerEmail,
			PaymentID: updated.ID,
			OrderID:   updated.OrderID,
			AmountKZT: updated.AmountKZT,
			Reason:    updated.FailureReason,
		}); err != nil {
			return Payment{}, err
		}
		return updated, nil
	}
	if err := s.publisher.PublishPaymentSucceeded(ctx, event); err != nil {
		return Payment{}, err
	}
	if err := s.emailer.SendReceipt(ctx, ReceiptEmail{
		To:        updated.CustomerEmail,
		PaymentID: updated.ID,
		OrderID:   updated.OrderID,
		AmountKZT: updated.AmountKZT,
	}); err != nil {
		return Payment{}, err
	}

	return updated, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (Payment, error) {
	if id == "" {
		return Payment{}, ErrInvalidInput
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Payment, error) {
	return s.repo.List(ctx, ListFilter{})
}

func (s *Service) ListByCustomerID(ctx context.Context, customerID string) ([]Payment, error) {
	if customerID == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.List(ctx, ListFilter{CustomerID: customerID})
}

func (s *Service) GetByOrderID(ctx context.Context, orderID string) (Payment, error) {
	if orderID == "" {
		return Payment{}, ErrInvalidInput
	}
	return s.repo.GetByOrderID(ctx, orderID)
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
		return Payment{}, ErrInvalidState
	}
	if p.Status != StatusSucceeded {
		return Payment{}, ErrInvalidState
	}

	p.Status = StatusRefunded
	p.RefundReason = input.Reason
	p.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		return Payment{}, err
	}

	event := paymentEvent(updated)
	event.Reason = input.Reason
	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return Payment{}, err
	}
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

func (s *Service) Confirm(ctx context.Context, input ConfirmInput) (Payment, error) {
	if input.PaymentID == "" || input.ProviderTransactionID == "" {
		return Payment{}, ErrInvalidInput
	}
	p, err := s.repo.GetByID(ctx, input.PaymentID)
	if err != nil {
		return Payment{}, err
	}
	if p.Status == StatusSucceeded {
		return p, nil
	}
	if p.Status != StatusPending {
		return Payment{}, ErrInvalidState
	}
	p.Status = StatusSucceeded
	p.ProviderTransactionID = input.ProviderTransactionID
	p.UpdatedAt = time.Now().UTC()
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		return Payment{}, err
	}
	event := paymentEvent(updated)
	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return Payment{}, err
	}
	if err := s.publisher.PublishPaymentSucceeded(ctx, event); err != nil {
		return Payment{}, err
	}
	return updated, s.emailer.SendReceipt(ctx, ReceiptEmail{
		To:        updated.CustomerEmail,
		PaymentID: updated.ID,
		OrderID:   updated.OrderID,
		AmountKZT: updated.AmountKZT,
	})
}

func (s *Service) Cancel(ctx context.Context, input CancelInput) (Payment, error) {
	if input.PaymentID == "" || input.Reason == "" {
		return Payment{}, ErrInvalidInput
	}
	p, err := s.repo.GetByID(ctx, input.PaymentID)
	if err != nil {
		return Payment{}, err
	}
	if p.Status == StatusRefunded || p.Status == StatusCancelled {
		return Payment{}, ErrInvalidState
	}
	p.Status = StatusCancelled
	p.FailureReason = input.Reason
	p.UpdatedAt = time.Now().UTC()
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		return Payment{}, err
	}
	event := paymentEvent(updated)
	event.Reason = input.Reason
	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return Payment{}, err
	}
	if err := s.publisher.PublishPaymentCancelled(ctx, event); err != nil {
		return Payment{}, err
	}
	return updated, nil
}

func paymentEvent(p Payment) PaymentEvent {
	return PaymentEvent{
		PaymentID:             p.ID,
		OrderID:               p.OrderID,
		CustomerID:            p.CustomerID,
		AmountKZT:             p.AmountKZT,
		Status:                p.Status,
		Reason:                firstNonEmpty(p.RefundReason, p.FailureReason),
		ProviderTransactionID: p.ProviderTransactionID,
		OccurredAt:            time.Now().UTC(),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func isAllowedMethod(method Method) bool {
	switch method {
	case MethodCard, MethodKaspi, MethodWallet:
		return true
	default:
		return false
	}
}
