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

	dbStartGet := time.Now()
	if paymentID, ok, err := s.idempotency.GetPaymentID(ctx, input.IdempotencyKey); err != nil {
		PaymentDBOperationsTotal.WithLabelValues("get_idempotency", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("get_idempotency").Observe(time.Since(dbStartGet).Seconds())
		return Payment{}, err
	} else if ok {
		PaymentDBOperationsTotal.WithLabelValues("get_idempotency", "success").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("get_idempotency").Observe(time.Since(dbStartGet).Seconds())
		dbStartGetByID := time.Now()
		res, err := s.repo.GetByID(ctx, paymentID)
		if err != nil {
			PaymentDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
			PaymentDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGetByID).Seconds())
			return Payment{}, err
		}
		PaymentDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGetByID).Seconds())
		return res, nil
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

	dbStartCreate := time.Now()
	created, err := s.repo.Create(ctx, p)
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("create", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("create").Observe(time.Since(dbStartCreate).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("create", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("create").Observe(time.Since(dbStartCreate).Seconds())

	dbStartSaveIdem := time.Now()
	if err := s.idempotency.SavePaymentID(ctx, input.IdempotencyKey, created.ID); err != nil {
		PaymentDBOperationsTotal.WithLabelValues("save_idempotency", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("save_idempotency").Observe(time.Since(dbStartSaveIdem).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("save_idempotency", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("save_idempotency").Observe(time.Since(dbStartSaveIdem).Seconds())

	event := paymentEvent(created)
	dbStartAppend := time.Now()
	if err := s.repo.AppendEvent(ctx, event); err != nil {
		PaymentDBOperationsTotal.WithLabelValues("append_event", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("append_event").Observe(time.Since(dbStartAppend).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("append_event", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("append_event").Observe(time.Since(dbStartAppend).Seconds())

	if err := s.publisher.PublishPaymentCreated(ctx, event); err != nil {
		return Payment{}, err
	}

	PaymentAttemptsTotal.WithLabelValues(string(created.Method)).Inc()

	startCharge := time.Now()
	result, err := s.provider.Charge(ctx, created)
	if err != nil {
		PaymentFailuresTotal.WithLabelValues(string(created.Method), "provider_error").Inc()
		PaymentProcessingDurationSeconds.WithLabelValues(string(created.Method), "provider_error").Observe(time.Since(startCharge).Seconds())
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

	dbStartUpdate := time.Now()
	updated, err := s.repo.Update(ctx, created)
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("update", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("update").Observe(time.Since(dbStartUpdate).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("update", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("update").Observe(time.Since(dbStartUpdate).Seconds())

	event = paymentEvent(updated)
	dbStartAppend2 := time.Now()
	if err := s.repo.AppendEvent(ctx, event); err != nil {
		PaymentDBOperationsTotal.WithLabelValues("append_event_result", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("append_event_result").Observe(time.Since(dbStartAppend2).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("append_event_result", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("append_event_result").Observe(time.Since(dbStartAppend2).Seconds())

	if updated.Status == StatusFailed {
		PaymentFailuresTotal.WithLabelValues(string(updated.Method), updated.FailureReason).Inc()
		PaymentProcessingDurationSeconds.WithLabelValues(string(updated.Method), "failed").Observe(time.Since(startCharge).Seconds())

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

	PaymentSuccessTotal.WithLabelValues(string(updated.Method)).Inc()
	PaymentAmountKZTTotal.Add(float64(updated.AmountKZT))
	PaymentProcessingDurationSeconds.WithLabelValues(string(updated.Method), "success").Observe(time.Since(startCharge).Seconds())

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
	dbStart := time.Now()
	res, err := s.repo.GetByID(ctx, id)
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStart).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStart).Seconds())
	return res, nil
}

func (s *Service) List(ctx context.Context) ([]Payment, error) {
	dbStart := time.Now()
	res, err := s.repo.List(ctx, ListFilter{})
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("list", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("list").Observe(time.Since(dbStart).Seconds())
		return nil, err
	}
	PaymentDBOperationsTotal.WithLabelValues("list", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("list").Observe(time.Since(dbStart).Seconds())
	return res, nil
}

func (s *Service) ListByCustomerID(ctx context.Context, customerID string) ([]Payment, error) {
	if customerID == "" {
		return nil, ErrInvalidInput
	}
	dbStart := time.Now()
	res, err := s.repo.List(ctx, ListFilter{CustomerID: customerID})
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("list_by_customer", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("list_by_customer").Observe(time.Since(dbStart).Seconds())
		return nil, err
	}
	PaymentDBOperationsTotal.WithLabelValues("list_by_customer", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("list_by_customer").Observe(time.Since(dbStart).Seconds())
	return res, nil
}

func (s *Service) GetByOrderID(ctx context.Context, orderID string) (Payment, error) {
	if orderID == "" {
		return Payment{}, ErrInvalidInput
	}
	dbStart := time.Now()
	res, err := s.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("get_by_order", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("get_by_order").Observe(time.Since(dbStart).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("get_by_order", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("get_by_order").Observe(time.Since(dbStart).Seconds())
	return res, nil
}

func (s *Service) Refund(ctx context.Context, input RefundInput) (Payment, error) {
	if input.PaymentID == "" || input.Reason == "" {
		return Payment{}, ErrInvalidInput
	}

	dbStartGet := time.Now()
	p, err := s.repo.GetByID(ctx, input.PaymentID)
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())

	if p.Status == StatusRefunded {
		return Payment{}, ErrInvalidState
	}
	if p.Status != StatusSucceeded {
		return Payment{}, ErrInvalidState
	}

	p.Status = StatusRefunded
	p.RefundReason = input.Reason
	p.UpdatedAt = time.Now().UTC()

	dbStartUpdate := time.Now()
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("update", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("update").Observe(time.Since(dbStartUpdate).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("update", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("update").Observe(time.Since(dbStartUpdate).Seconds())

	event := paymentEvent(updated)
	event.Reason = input.Reason
	dbStartAppend := time.Now()
	if err := s.repo.AppendEvent(ctx, event); err != nil {
		PaymentDBOperationsTotal.WithLabelValues("append_event", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("append_event").Observe(time.Since(dbStartAppend).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("append_event", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("append_event").Observe(time.Since(dbStartAppend).Seconds())

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
	dbStartGet := time.Now()
	p, err := s.repo.GetByID(ctx, input.PaymentID)
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())

	if p.Status == StatusSucceeded {
		return p, nil
	}
	if p.Status != StatusPending {
		return Payment{}, ErrInvalidState
	}
	p.Status = StatusSucceeded
	p.ProviderTransactionID = input.ProviderTransactionID
	p.UpdatedAt = time.Now().UTC()

	dbStartUpdate := time.Now()
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("update", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("update").Observe(time.Since(dbStartUpdate).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("update", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("update").Observe(time.Since(dbStartUpdate).Seconds())

	event := paymentEvent(updated)
	dbStartAppend := time.Now()
	if err := s.repo.AppendEvent(ctx, event); err != nil {
		PaymentDBOperationsTotal.WithLabelValues("append_event", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("append_event").Observe(time.Since(dbStartAppend).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("append_event", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("append_event").Observe(time.Since(dbStartAppend).Seconds())

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
	dbStartGet := time.Now()
	p, err := s.repo.GetByID(ctx, input.PaymentID)
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())

	if p.Status == StatusRefunded || p.Status == StatusCancelled {
		return Payment{}, ErrInvalidState
	}
	p.Status = StatusCancelled
	p.FailureReason = input.Reason
	p.UpdatedAt = time.Now().UTC()

	dbStartUpdate := time.Now()
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		PaymentDBOperationsTotal.WithLabelValues("update", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("update").Observe(time.Since(dbStartUpdate).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("update", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("update").Observe(time.Since(dbStartUpdate).Seconds())

	event := paymentEvent(updated)
	event.Reason = input.Reason
	dbStartAppend := time.Now()
	if err := s.repo.AppendEvent(ctx, event); err != nil {
		PaymentDBOperationsTotal.WithLabelValues("append_event", "error").Inc()
		PaymentDBOperationDurationSeconds.WithLabelValues("append_event").Observe(time.Since(dbStartAppend).Seconds())
		return Payment{}, err
	}
	PaymentDBOperationsTotal.WithLabelValues("append_event", "success").Inc()
	PaymentDBOperationDurationSeconds.WithLabelValues("append_event").Observe(time.Since(dbStartAppend).Seconds())

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
