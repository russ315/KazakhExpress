package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"kazakhexpress/product-service/internal/product"
)

func newTestHandler() *Handler {
	svc := product.NewService(product.NewMemoryRepository(), product.NoopCache{}, product.NoopPublisher{})
	return NewHandler(svc)
}

func TestHandlerProductFlow(t *testing.T) {
	h := newTestHandler()
	mux := h.Routes()

	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString(`{"name":"Rug","price_kzt":1000,"stock":4}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status: %d", rec.Code)
	}

	var created product.Product
	_ = json.NewDecoder(rec.Body).Decode(&created)

	req = httptest.NewRequest(http.MethodPost, "/products/"+created.ID+"/stock/reserve", bytes.NewBufferString(`{"quantity":2,"reservation_id":"res-a"}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("reserve status: %d body=%s", rec.Code, rec.Body.String())
	}
}
