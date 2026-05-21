package paymentservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeClient struct {
	created CreatePaymentRequest
}

func (c *fakeClient) Health(ctx context.Context) error {
	return nil
}

func (c *fakeClient) CreatePayment(ctx context.Context, input CreatePaymentRequest) (Payment, error) {
	c.created = input
	return Payment{ID: "pay-1", OrderID: input.OrderID, Status: "succeeded"}, nil
}

func TestRegisterRoutesExposesPaymentHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, &fakeClient{})
	req := httptest.NewRequest(http.MethodGet, "/payment/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func (c *fakeClient) GetPayment(ctx context.Context, paymentID string) (Payment, error) {
	return Payment{ID: paymentID}, nil
}

func (c *fakeClient) GetPaymentByOrderID(ctx context.Context, orderID string) (Payment, error) {
	return Payment{ID: "pay-1", OrderID: orderID}, nil
}

func (c *fakeClient) ListPayments(ctx context.Context, customerID string) ([]Payment, error) {
	return []Payment{{ID: "pay-1", CustomerID: customerID}}, nil
}

func (c *fakeClient) RefundPayment(ctx context.Context, input RefundPaymentRequest) (Payment, error) {
	return Payment{ID: input.PaymentID, RefundReason: input.Reason}, nil
}

func (c *fakeClient) ConfirmPayment(ctx context.Context, input ConfirmPaymentRequest) (Payment, error) {
	return Payment{ID: input.PaymentID, ProviderTransactionID: input.ProviderTransactionID}, nil
}

func (c *fakeClient) CancelPayment(ctx context.Context, input CancelPaymentRequest) (Payment, error) {
	return Payment{ID: input.PaymentID, FailureReason: input.Reason}, nil
}

func TestRegisterRoutesCreatesPaymentThroughClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	client := &fakeClient{}
	RegisterRoutes(router, client)

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

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if client.created.OrderID != "ord-1" {
		t.Fatalf("client order id = %q, want ord-1", client.created.OrderID)
	}
	var response Payment
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.ID != "pay-1" {
		t.Fatalf("response id = %q, want pay-1", response.ID)
	}
}
