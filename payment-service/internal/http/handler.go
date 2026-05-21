package http

import (
	"errors"
	nethttp "net/http"

	"kazakhexpress/payment-service/internal/payment"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *payment.Service
}

func NewHandler(service *payment.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() nethttp.Handler {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/health", h.health)
	router.GET("/metrics", h.metrics)

	paymentGroup := router.Group("/payment")
	h.registerPaymentRoutes(paymentGroup)

	return router
}

func (h *Handler) registerPaymentRoutes(group *gin.RouterGroup) {
	group.POST("", h.createPayment)
	group.GET("", h.listPayments)
	group.GET("/order/:orderID", h.getPaymentByOrderID)
	group.POST("/webhook/mock", h.mockWebhook)
	group.GET("/:id", h.getPayment)
	group.POST("/:id/refund", h.refundPayment)
	group.POST("/:id/confirm", h.confirmPayment)
	group.POST("/:id/cancel", h.cancelPayment)
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(nethttp.StatusOK, gin.H{"status": "ok", "service": "payment"})
}

func (h *Handler) metrics(c *gin.Context) {
	c.Header("Content-Type", "text/plain; version=0.0.4")
	c.String(nethttp.StatusOK, "# HELP payment_service_up Payment service process health.\n# TYPE payment_service_up gauge\npayment_service_up 1\n")
}

func (h *Handler) createPayment(c *gin.Context) {
	var input payment.CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, nethttp.StatusBadRequest, "invalid json")
		return
	}

	created, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(nethttp.StatusCreated, created)
}

func (h *Handler) listPayments(c *gin.Context) {
	customerID := c.Query("customer_id")
	if customerID != "" {
		payments, err := h.service.ListByCustomerID(c.Request.Context(), customerID)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(nethttp.StatusOK, payments)
		return
	}

	payments, err := h.service.List(c.Request.Context())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, payments)
}

func (h *Handler) getPaymentByOrderID(c *gin.Context) {
	found, err := h.service.GetByOrderID(c.Request.Context(), c.Param("orderID"))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, found)
}

func (h *Handler) getPayment(c *gin.Context) {
	found, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, found)
}

func (h *Handler) refundPayment(c *gin.Context) {
	var input payment.RefundInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, nethttp.StatusBadRequest, "invalid json")
		return
	}
	input.PaymentID = c.Param("id")

	refunded, err := h.service.Refund(c.Request.Context(), input)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, refunded)
}

func (h *Handler) confirmPayment(c *gin.Context) {
	var input payment.ConfirmInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, nethttp.StatusBadRequest, "invalid json")
		return
	}
	input.PaymentID = c.Param("id")

	confirmed, err := h.service.Confirm(c.Request.Context(), input)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, confirmed)
}

func (h *Handler) cancelPayment(c *gin.Context) {
	var input payment.CancelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, nethttp.StatusBadRequest, "invalid json")
		return
	}
	input.PaymentID = c.Param("id")

	cancelled, err := h.service.Cancel(c.Request.Context(), input)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, cancelled)
}

func (h *Handler) mockWebhook(c *gin.Context) {
	var input struct {
		PaymentID             string `json:"payment_id"`
		Status                string `json:"status"`
		ProviderTransactionID string `json:"provider_transaction_id"`
		Reason                string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, nethttp.StatusBadRequest, "invalid json")
		return
	}

	switch payment.Status(input.Status) {
	case payment.StatusSucceeded:
		confirmed, err := h.service.Confirm(c.Request.Context(), payment.ConfirmInput{
			PaymentID:             input.PaymentID,
			ProviderTransactionID: input.ProviderTransactionID,
		})
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(nethttp.StatusOK, confirmed)
	case payment.StatusCancelled, payment.StatusFailed:
		cancelled, err := h.service.Cancel(c.Request.Context(), payment.CancelInput{
			PaymentID: input.PaymentID,
			Reason:    input.Reason,
		})
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(nethttp.StatusOK, cancelled)
	default:
		writeError(c, nethttp.StatusBadRequest, "unsupported webhook status")
	}
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, payment.ErrInvalidInput):
		writeError(c, nethttp.StatusBadRequest, err.Error())
	case errors.Is(err, payment.ErrNotFound):
		writeError(c, nethttp.StatusNotFound, err.Error())
	case errors.Is(err, payment.ErrInvalidState):
		writeError(c, nethttp.StatusConflict, err.Error())
	default:
		writeError(c, nethttp.StatusInternalServerError, "internal server error")
	}
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
