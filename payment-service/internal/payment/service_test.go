package payment

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct {
	payments map[string]Payment
	events   []PaymentEvent
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{payments: make(map[string]Payment)}
}

func (r *fakeRepository) Create(ctx context.Context, p Payment) (Payment, error) {
	r.payments[p.ID] = p
	return p, nil
}

func (r *fakeRepository) GetByID(ctx context.Context, id string) (Payment, error) {
	p, ok := r.payments[id]
	if !ok {
		return Payment{}, ErrNotFound
	}
	return p, nil
}

func (r *fakeRepository) List(ctx context.Context, filter ListFilter) ([]Payment, error) {
	payments := make([]Payment, 0, len(r.payments))
	for _, p := range r.payments {
		if filter.CustomerID != "" && p.CustomerID != filter.CustomerID {
			continue
		}
		payments = append(payments, p)
	}
	return payments, nil
}

func (r *fakeRepository) GetByOrderID(ctx context.Context, orderID string) (Payment, error) {
	for _, p := range r.payments {
		if p.OrderID == orderID {
			return p, nil
		}
	}
	return Payment{}, ErrNotFound
}

func (r *fakeRepository) Update(ctx context.Context, p Payment) (Payment, error) {
	if _, ok := r.payments[p.ID]; !ok {
		return Payment{}, ErrNotFound
	}
	r.payments[p.ID] = p
	return p, nil
}

func (r *fakeRepository) AppendEvent(ctx context.Context, event PaymentEvent) error {
	r.events = append(r.events, event)
	return nil
}

type fakePublisher struct {
	created   []PaymentEvent
	succeeded []PaymentEvent
	failed    []PaymentEvent
	refunded  []PaymentEvent
	cancelled []PaymentEvent
}

func (p *fakePublisher) PublishPaymentCreated(ctx context.Context, event PaymentEvent) error {
	p.created = append(p.created, event)
	return nil
}

func (p *fakePublisher) PublishPaymentSucceeded(ctx context.Context, event PaymentEvent) error {
	p.succeeded = append(p.succeeded, event)
	return nil
}

func (p *fakePublisher) PublishPaymentFailed(ctx context.Context, event PaymentEvent) error {
	p.failed = append(p.failed, event)
	return nil
}

func (p *fakePublisher) PublishPaymentRefunded(ctx context.Context, event PaymentEvent) error {
	p.refunded = append(p.refunded, event)
	return nil
}

func (p *fakePublisher) PublishPaymentCancelled(ctx context.Context, event PaymentEvent) error {
	p.cancelled = append(p.cancelled, event)
	return nil
}

type fakeEmailSender struct {
	receipts []ReceiptEmail
	refunds  []RefundEmail
	failures []FailureEmail
}

func (s *fakeEmailSender) SendReceipt(ctx context.Context, email ReceiptEmail) error {
	s.receipts = append(s.receipts, email)
	return nil
}

func (s *fakeEmailSender) SendRefund(ctx context.Context, email RefundEmail) error {
	s.refunds = append(s.refunds, email)
	return nil
}

func (s *fakeEmailSender) SendFailure(ctx context.Context, email FailureEmail) error {
	s.failures = append(s.failures, email)
	return nil
}

type fakeIdempotencyStore struct {
	values map[string]string
}

func newFakeIdempotencyStore() *fakeIdempotencyStore {
	return &fakeIdempotencyStore{values: make(map[string]string)}
}

func (s *fakeIdempotencyStore) GetPaymentID(ctx context.Context, key string) (string, bool, error) {
	value, ok := s.values[key]
	return value, ok, nil
}

func (s *fakeIdempotencyStore) SavePaymentID(ctx context.Context, key string, paymentID string) error {
	s.values[key] = paymentID
	return nil
}

type fakeProvider struct {
	status Status
	calls  int
}

func (p *fakeProvider) Charge(ctx context.Context, payment Payment) (ProviderResult, error) {
	p.calls++
	if p.status == "" {
		p.status = StatusSucceeded
	}
	return ProviderResult{
		Status:                p.status,
		ProviderTransactionID: "txn-" + payment.ID,
		FailureReason:         "declined",
	}, nil
}

func TestCreatePaymentStoresPaymentAndSendsSideEffects(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	emailer := &fakeEmailSender{}
	provider := &fakeProvider{}
	service := NewService(repo, publisher, emailer, newFakeIdempotencyStore(), provider)

	payment, err := service.Create(context.Background(), CreateInput{
		OrderID:        "ord-1",
		CustomerID:     "usr-1",
		CustomerEmail:  "buyer@example.com",
		AmountKZT:      25000,
		Method:         MethodCard,
		IdempotencyKey: "idem-1",
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if payment.ID == "" {
		t.Fatal("Create() returned empty payment ID")
	}
	if payment.Status != StatusSucceeded {
		t.Fatalf("Create() status = %q, want %q", payment.Status, StatusSucceeded)
	}
	if payment.ProviderTransactionID == "" {
		t.Fatal("Create() returned empty provider transaction id")
	}
	if len(publisher.created) != 1 {
		t.Fatalf("created events = %d, want 1", len(publisher.created))
	}
	if len(publisher.succeeded) != 1 {
		t.Fatalf("succeeded events = %d, want 1", len(publisher.succeeded))
	}
	if len(emailer.receipts) != 1 {
		t.Fatalf("receipt emails = %d, want 1", len(emailer.receipts))
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
}

func TestCreatePaymentRejectsInvalidAmount(t *testing.T) {
	service := NewService(newFakeRepository(), &fakePublisher{}, &fakeEmailSender{}, newFakeIdempotencyStore(), &fakeProvider{})

	_, err := service.Create(context.Background(), CreateInput{
		OrderID:        "ord-1",
		CustomerID:     "usr-1",
		CustomerEmail:  "buyer@example.com",
		AmountKZT:      0,
		Method:         MethodCard,
		IdempotencyKey: "idem-1",
	})

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Create() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestCreatePaymentReturnsExistingPaymentForDuplicateIdempotencyKey(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	emailer := &fakeEmailSender{}
	cache := newFakeIdempotencyStore()
	provider := &fakeProvider{}
	service := NewService(repo, publisher, emailer, cache, provider)

	first, err := service.Create(context.Background(), CreateInput{
		OrderID:        "ord-1",
		CustomerID:     "usr-1",
		CustomerEmail:  "buyer@example.com",
		AmountKZT:      25000,
		Method:         MethodKaspi,
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	second, err := service.Create(context.Background(), CreateInput{
		OrderID:        "ord-1",
		CustomerID:     "usr-1",
		CustomerEmail:  "buyer@example.com",
		AmountKZT:      25000,
		Method:         MethodKaspi,
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}

	if second.ID != first.ID {
		t.Fatalf("duplicate Create() ID = %q, want %q", second.ID, first.ID)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
	if len(publisher.created) != 1 || len(publisher.succeeded) != 1 {
		t.Fatalf("published created/succeeded = %d/%d, want 1/1", len(publisher.created), len(publisher.succeeded))
	}
}

func TestCreatePaymentStoresFailureAndSendsFailureEmail(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	emailer := &fakeEmailSender{}
	service := NewService(repo, publisher, emailer, newFakeIdempotencyStore(), &fakeProvider{status: StatusFailed})

	payment, err := service.Create(context.Background(), CreateInput{
		OrderID:        "ord-1",
		CustomerID:     "usr-1",
		CustomerEmail:  "buyer@example.com",
		AmountKZT:      25000,
		Method:         MethodCard,
		IdempotencyKey: "idem-1",
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if payment.Status != StatusFailed {
		t.Fatalf("Create() status = %q, want %q", payment.Status, StatusFailed)
	}
	if payment.FailureReason == "" {
		t.Fatal("Create() returned empty failure reason")
	}
	if len(publisher.failed) != 1 {
		t.Fatalf("failed events = %d, want 1", len(publisher.failed))
	}
	if len(emailer.failures) != 1 {
		t.Fatalf("failure emails = %d, want 1", len(emailer.failures))
	}
	if len(emailer.receipts) != 0 {
		t.Fatalf("receipt emails = %d, want 0", len(emailer.receipts))
	}
}

func TestGetPaymentByOrderIDReturnsPayment(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, &fakePublisher{}, &fakeEmailSender{}, newFakeIdempotencyStore(), &fakeProvider{})

	created, err := service.Create(context.Background(), CreateInput{
		OrderID:        "ord-42",
		CustomerID:     "usr-1",
		CustomerEmail:  "buyer@example.com",
		AmountKZT:      25000,
		Method:         MethodWallet,
		IdempotencyKey: "idem-42",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := service.GetByOrderID(context.Background(), "ord-42")
	if err != nil {
		t.Fatalf("GetByOrderID() error = %v", err)
	}
	if found.ID != created.ID {
		t.Fatalf("GetByOrderID() ID = %q, want %q", found.ID, created.ID)
	}
}

func TestRefundPaymentMarksPaymentRefunded(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	emailer := &fakeEmailSender{}
	service := NewService(repo, publisher, emailer, newFakeIdempotencyStore(), &fakeProvider{})

	payment, err := service.Create(context.Background(), CreateInput{
		OrderID:        "ord-1",
		CustomerID:     "usr-1",
		CustomerEmail:  "buyer@example.com",
		AmountKZT:      25000,
		Method:         MethodCard,
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	refunded, err := service.Refund(context.Background(), RefundInput{
		PaymentID: payment.ID,
		Reason:    "customer request",
	})

	if err != nil {
		t.Fatalf("Refund() error = %v", err)
	}
	if refunded.Status != StatusRefunded {
		t.Fatalf("Refund() status = %q, want %q", refunded.Status, StatusRefunded)
	}
	if refunded.RefundReason != "customer request" {
		t.Fatalf("Refund() reason = %q", refunded.RefundReason)
	}
	if len(publisher.refunded) != 1 {
		t.Fatalf("refunded events = %d, want 1", len(publisher.refunded))
	}
	if len(emailer.refunds) != 1 {
		t.Fatalf("refund emails = %d, want 1", len(emailer.refunds))
	}
}

func TestRefundPaymentRejectsAlreadyRefundedPayment(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, &fakePublisher{}, &fakeEmailSender{}, newFakeIdempotencyStore(), &fakeProvider{})

	payment, err := service.Create(context.Background(), CreateInput{
		OrderID:        "ord-1",
		CustomerID:     "usr-1",
		CustomerEmail:  "buyer@example.com",
		AmountKZT:      25000,
		Method:         MethodCard,
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.Refund(context.Background(), RefundInput{PaymentID: payment.ID, Reason: "first"}); err != nil {
		t.Fatalf("first Refund() error = %v", err)
	}

	_, err = service.Refund(context.Background(), RefundInput{PaymentID: payment.ID, Reason: "second"})
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("second Refund() error = %v, want %v", err, ErrInvalidState)
	}
}

func TestCancelPaymentMarksPaymentCancelled(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	service := NewService(repo, publisher, &fakeEmailSender{}, newFakeIdempotencyStore(), &fakeProvider{})

	payment, err := service.Create(context.Background(), CreateInput{
		OrderID:        "ord-1",
		CustomerID:     "usr-1",
		CustomerEmail:  "buyer@example.com",
		AmountKZT:      25000,
		Method:         MethodCard,
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	cancelled, err := service.Cancel(context.Background(), CancelInput{PaymentID: payment.ID, Reason: "order cancelled"})
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if cancelled.Status != StatusCancelled {
		t.Fatalf("Cancel() status = %q, want %q", cancelled.Status, StatusCancelled)
	}
	if len(publisher.cancelled) != 1 {
		t.Fatalf("cancelled events = %d, want 1", len(publisher.cancelled))
	}
}

func TestConfirmPaymentReturnsSucceededPaymentWithoutDuplicateSideEffects(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	emailer := &fakeEmailSender{}
	service := NewService(repo, publisher, emailer, newFakeIdempotencyStore(), &fakeProvider{})

	payment, err := service.Create(context.Background(), CreateInput{
		OrderID:        "ord-1",
		CustomerID:     "usr-1",
		CustomerEmail:  "buyer@example.com",
		AmountKZT:      25000,
		Method:         MethodCard,
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	confirmed, err := service.Confirm(context.Background(), ConfirmInput{
		PaymentID:             payment.ID,
		ProviderTransactionID: "txn-replay",
	})
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if confirmed.ID != payment.ID {
		t.Fatalf("Confirm() ID = %q, want %q", confirmed.ID, payment.ID)
	}
	if len(publisher.succeeded) != 1 {
		t.Fatalf("succeeded events = %d, want 1", len(publisher.succeeded))
	}
	if len(emailer.receipts) != 1 {
		t.Fatalf("receipt emails = %d, want 1", len(emailer.receipts))
	}
}
