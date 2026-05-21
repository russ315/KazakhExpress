package orderservice

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
	group := router.Group("/orders")
	group.GET("/health", handler.health)
	group.POST("", handler.createOrder)
	group.GET("", handler.listOrders)
	group.GET("/:id", handler.getOrder)
	group.PATCH("/:id/status", handler.updateStatus)
	group.POST("/:id/cancel", handler.cancelOrder)
}

func (h *Handler) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.client.Health(ctx); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"service": "order", "status": "unavailable", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"service": "order", "status": "ok"})
}

func (h *Handler) createOrder(c *gin.Context) {
	var input CreateOrderRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	created, err := h.client.CreateOrder(c.Request.Context(), input)
	writeResult(c, http.StatusCreated, created, err)
}

func (h *Handler) listOrders(c *gin.Context) {
	orders, err := h.client.ListOrders(c.Request.Context())
	writeResult(c, http.StatusOK, orders, err)
}

func (h *Handler) getOrder(c *gin.Context) {
	found, err := h.client.GetOrder(c.Request.Context(), c.Param("id"))
	writeResult(c, http.StatusOK, found, err)
}

func (h *Handler) updateStatus(c *gin.Context) {
	var input UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	input.OrderID = c.Param("id")
	updated, err := h.client.UpdateOrderStatus(c.Request.Context(), input)
	writeResult(c, http.StatusOK, updated, err)
}

func (h *Handler) cancelOrder(c *gin.Context) {
	var input CancelOrderRequest
	_ = c.ShouldBindJSON(&input)
	input.OrderID = c.Param("id")
	cancelled, err := h.client.CancelOrder(c.Request.Context(), input)
	writeResult(c, http.StatusOK, cancelled, err)
}

func writeResult(c *gin.Context, status int, value any, err error) {
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(status, value)
}
