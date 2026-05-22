package reviewservice

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	client Client
}

func RegisterRoutes(router gin.IRouter, client Client) {
	handler := &Handler{client: client}
	router.GET("/reviews/health", handler.health)
	router.POST("/products/:productId/reviews", handler.createReview)
	router.GET("/products/:productId/reviews", handler.listReviews)
	router.GET("/products/:productId/rating", handler.getProductRating)
	router.GET("/reviews/:id", handler.getReview)
	router.PUT("/reviews/:id", handler.updateReview)
	router.DELETE("/reviews/:id", handler.deleteReview)
}

func (h *Handler) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := h.client.Health(ctx); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"service": "review", "status": "unavailable", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"service": "review", "status": "ok"})
}

func (h *Handler) createReview(c *gin.Context) {
	var input CreateReviewRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	review, err := h.client.CreateReview(c.Request.Context(), c.Param("productId"), input)
	writeResult(c, http.StatusCreated, review, err)
}

func (h *Handler) listReviews(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := h.client.ListProductReviews(c.Request.Context(), c.Param("productId"), page, pageSize)
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) getProductRating(c *gin.Context) {
	rating, err := h.client.GetProductRating(c.Request.Context(), c.Param("productId"))
	writeResult(c, http.StatusOK, rating, err)
}

func (h *Handler) getReview(c *gin.Context) {
	review, err := h.client.GetReview(c.Request.Context(), c.Param("id"))
	writeResult(c, http.StatusOK, review, err)
}

func (h *Handler) updateReview(c *gin.Context) {
	var input UpdateReviewRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	review, err := h.client.UpdateReview(c.Request.Context(), c.Param("id"), input)
	writeResult(c, http.StatusOK, review, err)
}

func (h *Handler) deleteReview(c *gin.Context) {
	err := h.client.DeleteReview(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeResult(c, http.StatusNoContent, nil, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func writeResult(c *gin.Context, status int, value any, err error) {
	if err != nil {
		code := http.StatusBadGateway
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument, codes.FailedPrecondition:
				code = http.StatusBadRequest
			case codes.NotFound:
				code = http.StatusNotFound
			case codes.AlreadyExists:
				code = http.StatusConflict
			}
		}
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}
	if value == nil {
		c.Status(status)
		return
	}
	c.JSON(status, value)
}
