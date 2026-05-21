package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"kazakhexpress/order-service/internal/order"
)

type Handler struct {
	service *order.Service
}

func NewHandler(service *order.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/orders", h.orders)
	mux.HandleFunc("/orders/", h.orderByID)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "order"})
}

func (h *Handler) orders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createOrder(w, r)
	case http.MethodGet:
		h.listOrders(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) orderByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/orders/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		h.getOrder(w, r, id)
		return
	}

	if len(parts) == 2 && parts[1] == "status" && r.Method == http.MethodPatch {
		h.updateStatus(w, r, id)
		return
	}

	if len(parts) == 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
		h.cancelOrder(w, r, id)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var input order.CreateInput
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

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.service.List(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request, id string) {
	found, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, found)
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request, id string) {
	var input order.UpdateStatusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	updated, err := h.service.UpdateStatus(r.Context(), id, input.Status)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request, id string) {
	var input order.CancelInput
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&input)
	}

	cancelled, err := h.service.Cancel(r.Context(), id, input.Reason)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, cancelled)
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, order.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, order.ErrNotFound):
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
