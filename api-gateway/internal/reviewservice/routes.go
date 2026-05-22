package reviewservice

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct{ client Client }

func RegisterRoutes(router gin.IRouter, client Client) {
	h := &Handler{client: client}
	router.GET("/reviews/health", h.health)
	router.POST("/products/:productId/reviews", h.create)
	router.GET("/products/:productId/reviews", h.list)
	router.GET("/products/:productId/rating", h.rating)
	router.GET("/reviews/:id", h.get)
	router.PUT("/reviews/:id", h.update)
	router.DELETE("/reviews/:id", h.delete)
}

func (h *Handler) health(c *gin.Context) {
	err := h.client.Health(c.Request.Context())
	writeResult(c, http.StatusOK, gin.H{"service": "review", "status": "ok"}, err)
}

func (h *Handler) create(c *gin.Context) {
	var input CreateReviewRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	result, err := h.client.Create(c.Request.Context(), c.Param("productId"), input)
	writeResult(c, http.StatusCreated, result, err)
}

func (h *Handler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	result, err := h.client.List(c.Request.Context(), c.Param("productId"), limit, offset)
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) rating(c *gin.Context) {
	result, err := h.client.Rating(c.Request.Context(), c.Param("productId"))
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) get(c *gin.Context) {
	result, err := h.client.Get(c.Request.Context(), c.Param("id"))
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) update(c *gin.Context) {
	var input UpdateReviewRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	result, err := h.client.Update(c.Request.Context(), c.Param("id"), input)
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) delete(c *gin.Context) {
	err := h.client.Delete(c.Request.Context(), c.Param("id"))
	writeResult(c, http.StatusOK, gin.H{"deleted": true}, err)
}

func writeResult(c *gin.Context, status int, value any, err error) {
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(status, value)
}
