package payment

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct {
	payments map[string]Payment
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

func (r *fakeRepository) List(ctx context.Context) ([]Payment, error) {
	payments := make([]Payment, 0, len(r.payments))
	for _, p := range r.payments {
		payments = append(payments, p)
	}
	return payments, nil
}

func (r *fakeRepository) Update(ctx context.Context, p Payment) (Payment, error) {
	if _, ok := r.payments[p.ID]; !ok {
		return Payment{}, ErrNotFound
	}
	r.payments[p.ID] = p
	return p, nil
}

type fakePublisher struct {
	events []PaymentEvent
}

func (p *fakePublisher) PublishPaymentCreated(ctx context.Context, event PaymentEvent) error {
	p.events = append(p.events, event)
	return nil
}

func (p *fakePublisher) PublishPaymentRefunded(ctx context.Context, event PaymentEvent) error {
	p.events = append(p.events, event)
	return nil
}

type fakeEmailSender struct {
	receipts []ReceiptEmail
	refunds  []RefundEmail
}

func (s *fakeEmailSender) SendReceipt(ctx context.Context, email ReceiptEmail) error {
	s.receipts = append(s.receipts, email)
	return nil
}

func (s *fakeEmailSender) SendRefund(ctx context.Context, email RefundEmail) error {
	s.refunds = append(s.refunds, email)
	return nil
}

func TestCreatePaymentStoresPaymentAndSendsSideEffects(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	emailer := &fakeEmailSender{}
	service := NewService(repo, publisher, emailer)

	payment, err := service.Create(context.Background(), CreateInput{
		OrderID:       "ord-1",
		CustomerID:    "usr-1",
		CustomerEmail: "buyer@example.com",
		AmountKZT:     25000,
		Method:        MethodCard,
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if payment.ID == "" {
		t.Fatal("Create() returned empty payment ID")
	}
	if payment.Status != StatusPaid {
		t.Fatalf("Create() status = %q, want %q", payment.Status, StatusPaid)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.events))
	}
	if len(emailer.receipts) != 1 {
		t.Fatalf("receipt emails = %d, want 1", len(emailer.receipts))
	}
}

func TestCreatePaymentRejectsInvalidAmount(t *testing.T) {
	service := NewService(newFakeRepository(), &fakePublisher{}, &fakeEmailSender{})

	_, err := service.Create(context.Background(), CreateInput{
		OrderID:       "ord-1",
		CustomerID:    "usr-1",
		CustomerEmail: "buyer@example.com",
		AmountKZT:     0,
		Method:        MethodCard,
	})

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Create() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestRefundPaymentMarksPaymentRefunded(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	emailer := &fakeEmailSender{}
	service := NewService(repo, publisher, emailer)

	payment, err := service.Create(context.Background(), CreateInput{
		OrderID:       "ord-1",
		CustomerID:    "usr-1",
		CustomerEmail: "buyer@example.com",
		AmountKZT:     25000,
		Method:        MethodCard,
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
	if len(publisher.events) != 2 {
		t.Fatalf("published events = %d, want 2", len(publisher.events))
	}
	if len(emailer.refunds) != 1 {
		t.Fatalf("refund emails = %d, want 1", len(emailer.refunds))
	}
}
