package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kazakhexpress/payment-service/internal/payment"
)

type fakeRepository struct {
	payments map[string]payment.Payment
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{payments: make(map[string]payment.Payment)}
}

func (r *fakeRepository) Create(ctx context.Context, p payment.Payment) (payment.Payment, error) {
	r.payments[p.ID] = p
	return p, nil
}

func (r *fakeRepository) GetByID(ctx context.Context, id string) (payment.Payment, error) {
	p, ok := r.payments[id]
	if !ok {
		return payment.Payment{}, payment.ErrNotFound
	}
	return p, nil
}

func (r *fakeRepository) GetByOrderID(ctx context.Context, orderID string) (payment.Payment, error) {
	for _, p := range r.payments {
		if p.OrderID == orderID {
			return p, nil
		}
	}
	return payment.Payment{}, payment.ErrNotFound
}

func (r *fakeRepository) List(ctx context.Context, filter payment.ListFilter) ([]payment.Payment, error) {
	payments := make([]payment.Payment, 0, len(r.payments))
	for _, p := range r.payments {
		if filter.CustomerID != "" && p.CustomerID != filter.CustomerID {
			continue
		}
		payments = append(payments, p)
	}
	return payments, nil
}

func (r *fakeRepository) Update(ctx context.Context, p payment.Payment) (payment.Payment, error) {
	r.payments[p.ID] = p
	return p, nil
}

func (r *fakeRepository) AppendEvent(ctx context.Context, event payment.PaymentEvent) error {
	return nil
}

type fakePublisher struct{}

func (p fakePublisher) PublishPaymentCreated(ctx context.Context, event payment.PaymentEvent) error {
	return nil
}

func (p fakePublisher) PublishPaymentSucceeded(ctx context.Context, event payment.PaymentEvent) error {
	return nil
}

func (p fakePublisher) PublishPaymentFailed(ctx context.Context, event payment.PaymentEvent) error {
	return nil
}

func (p fakePublisher) PublishPaymentRefunded(ctx context.Context, event payment.PaymentEvent) error {
	return nil
}

func (p fakePublisher) PublishPaymentCancelled(ctx context.Context, event payment.PaymentEvent) error {
	return nil
}

type fakeEmailSender struct{}

func (s fakeEmailSender) SendReceipt(ctx context.Context, email payment.ReceiptEmail) error {
	return nil
}

func (s fakeEmailSender) SendRefund(ctx context.Context, email payment.RefundEmail) error {
	return nil
}

func (s fakeEmailSender) SendFailure(ctx context.Context, email payment.FailureEmail) error {
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

type fakeProvider struct{}

func (p fakeProvider) Charge(ctx context.Context, pay payment.Payment) (payment.ProviderResult, error) {
	return payment.ProviderResult{
		Status:                payment.StatusSucceeded,
		ProviderTransactionID: "txn-" + pay.ID,
	}, nil
}

func TestRoutesExposeSingularPaymentPrefix(t *testing.T) {
	service := payment.NewService(newFakeRepository(), fakePublisher{}, fakeEmailSender{}, newFakeIdempotencyStore(), fakeProvider{})
	handler := NewHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/payment", strings.NewReader(`{
		"order_id":"ord-1",
		"customer_id":"usr-1",
		"customer_email":"buyer@example.com",
		"amount_kzt":25000,
		"method":"kaspi",
		"idempotency_key":"idem-1"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		body, _ := io.ReadAll(rec.Body)
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, string(body))
	}
}
