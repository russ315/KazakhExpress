package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"kazakhexpress/payment-service/internal/payment"
)

type Handler struct {
	service *payment.Service
}

func NewHandler(service *payment.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/payments", h.payments)
	mux.HandleFunc("/payments/", h.paymentByID)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "payment"})
}

func (h *Handler) payments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createPayment(w, r)
	case http.MethodGet:
		h.listPayments(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) paymentByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/payments/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}

	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		h.getPayment(w, r, id)
		return
	}

	if len(parts) == 2 && parts[1] == "refund" && r.Method == http.MethodPost {
		h.refundPayment(w, r, id)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (h *Handler) createPayment(w http.ResponseWriter, r *http.Request) {
	var input payment.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	created, err := h.service.Create(r.Context(), input)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) listPayments(w http.ResponseWriter, r *http.Request) {
	payments, err := h.service.List(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payments)
}

func (h *Handler) getPayment(w http.ResponseWriter, r *http.Request, id string) {
	found, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func (h *Handler) refundPayment(w http.ResponseWriter, r *http.Request, id string) {
	var input payment.RefundInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	input.PaymentID = id

	refunded, err := h.service.Refund(r.Context(), input)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, refunded)
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, payment.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, payment.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
