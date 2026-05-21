package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"kazakhexpress/order-service/internal/order"
)

func newTestHandler() *Handler {
	return NewHandler(order.NewService(newTestRepo(), nil, nil))
}

type testRepo struct {
	orders map[string]order.Order
}

func newTestRepo() *testRepo {
	return &testRepo{orders: make(map[string]order.Order)}
}

func (r *testRepo) Create(ctx context.Context, o order.Order) (order.Order, error) {
	r.orders[o.ID] = o
	return o, nil
}

func (r *testRepo) List(ctx context.Context) ([]order.Order, error) {
	orders := make([]order.Order, 0, len(r.orders))
	for _, o := range r.orders {
		orders = append(orders, o)
	}
	return orders, nil
}

func (r *testRepo) GetByID(ctx context.Context, id string) (order.Order, error) {
	o, ok := r.orders[id]
	if !ok {
		return order.Order{}, order.ErrNotFound
	}
	return o, nil
}

func (r *testRepo) UpdateStatus(ctx context.Context, id string, from order.Status, to order.Status, reason string) (order.Order, error) {
	o, ok := r.orders[id]
	if !ok {
		return order.Order{}, order.ErrNotFound
	}
	o.Status = to
	r.orders[id] = o
	return o, nil
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	newTestHandler().Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCreateOrderHTTP(t *testing.T) {
	body := []byte(`{"customer_id":"customer-1","items":[{"product_id":"p1","name":"Shapan","quantity":1,"price_kzt":1000}]}`)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newTestHandler().Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var created order.Order
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.Status != order.StatusCreated {
		t.Fatalf("status = %s, want %s", created.Status, order.StatusCreated)
	}
}

func TestCreateOrderInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte("{")))
	rec := httptest.NewRecorder()

	newTestHandler().Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetOrderHTTP(t *testing.T) {
	handler := newTestHandler()
	createReq := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(`{"customer_id":"c1","items":[{"product_id":"p1","name":"x","quantity":1,"price_kzt":100}]}`)))
	createRec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(createRec, createReq)

	var created order.Order
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+created.ID, nil)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGetOrderNotFoundHTTP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/orders/missing", nil)
	rec := httptest.NewRecorder()

	newTestHandler().Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCancelOrderHTTP(t *testing.T) {
	handler := newTestHandler()
	createReq := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(`{"customer_id":"c1","items":[{"product_id":"p1","name":"x","quantity":1,"price_kzt":100}]}`)))
	createRec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(createRec, createReq)

	var created order.Order
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	body := []byte(`{"reason":"changed mind"}`)
	req := httptest.NewRequest(http.MethodPost, "/orders/"+created.ID+"/cancel", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var cancelled order.Order
	if err := json.NewDecoder(rec.Body).Decode(&cancelled); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if cancelled.Status != order.StatusCanceled {
		t.Fatalf("status = %s, want %s", cancelled.Status, order.StatusCanceled)
	}
}

func TestUpdateStatusHTTP(t *testing.T) {
	handler := newTestHandler()
	createReq := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(`{"customer_id":"c1","items":[{"product_id":"p1","name":"x","quantity":1,"price_kzt":100}]}`)))
	createRec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(createRec, createReq)

	var created order.Order
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	body := []byte(`{"status":"shipped"}`)
	req := httptest.NewRequest(http.MethodPatch, "/orders/"+created.ID+"/status", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestMethodNotAllowedHTTP(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/orders", nil)
	rec := httptest.NewRecorder()

	newTestHandler().Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
