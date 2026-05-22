package paymentservice

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	client Client
}

func RegisterRoutes(router gin.IRouter, client Client) {
	handler := &Handler{client: client}
	group := router.Group("/payment")
	group.GET("/health", handler.health)
	group.POST("", handler.createPayment)
	group.GET("", handler.listPayments)
	group.GET("/order/:orderID", handler.getPaymentByOrderID)
	group.GET("/:id", handler.getPayment)
	group.POST("/:id/refund", handler.refundPayment)
	group.POST("/:id/confirm", handler.confirmPayment)
	group.POST("/:id/cancel", handler.cancelPayment)
}

func (h *Handler) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.client.Health(ctx); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"service": "payment",
			"status":  "unavailable",
			"error":   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"service": "payment", "status": "ok"})
}

func (h *Handler) createPayment(c *gin.Context) {
	var input CreatePaymentRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	payment, err := h.client.CreatePayment(c.Request.Context(), input)
	writeResult(c, http.StatusCreated, payment, err)
}

func (h *Handler) listPayments(c *gin.Context) {
	payments, err := h.client.ListPayments(c.Request.Context(), c.Query("customer_id"))
	writeResult(c, http.StatusOK, payments, err)
}

func (h *Handler) getPayment(c *gin.Context) {
	payment, err := h.client.GetPayment(c.Request.Context(), c.Param("id"))
	writeResult(c, http.StatusOK, payment, err)
}

func (h *Handler) getPaymentByOrderID(c *gin.Context) {
	payment, err := h.client.GetPaymentByOrderID(c.Request.Context(), c.Param("orderID"))
	writeResult(c, http.StatusOK, payment, err)
}

func (h *Handler) refundPayment(c *gin.Context) {
	var input RefundPaymentRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	input.PaymentID = c.Param("id")
	payment, err := h.client.RefundPayment(c.Request.Context(), input)
	writeResult(c, http.StatusOK, payment, err)
}

func (h *Handler) confirmPayment(c *gin.Context) {
	var input ConfirmPaymentRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	input.PaymentID = c.Param("id")
	payment, err := h.client.ConfirmPayment(c.Request.Context(), input)
	writeResult(c, http.StatusOK, payment, err)
}

func (h *Handler) cancelPayment(c *gin.Context) {
	var input CancelPaymentRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	input.PaymentID = c.Param("id")
	payment, err := h.client.CancelPayment(c.Request.Context(), input)
	writeResult(c, http.StatusOK, payment, err)
}

func writeResult(c *gin.Context, status int, value any, err error) {
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(status, value)
}
