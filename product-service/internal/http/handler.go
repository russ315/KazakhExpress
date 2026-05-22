package http

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"kazakhexpress/product-service/internal/product"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *product.Service
}

func NewHandler(service *product.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() http.Handler {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.GET("/health", h.health)
	router.GET("/metrics", h.metrics)
	router.POST("/products", h.createProduct)
	router.GET("/products", h.listProducts)
	router.GET("/products/:id", h.getProduct)
	router.PATCH("/products/:id/stock", h.updateStock)
	router.POST("/products/:id/images", h.addImage)
	return router
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "product"})
}

func (h *Handler) metrics(c *gin.Context) {
	c.String(http.StatusOK, "kazakhexpress_service_up{service=\"product\"} 1\n")
}

func (h *Handler) createProduct(c *gin.Context) {
	var input product.CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	created, err := h.service.Create(c.Request.Context(), input)
	writeResult(c, http.StatusCreated, created, err)
}

func (h *Handler) listProducts(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	list, err := h.service.List(c.Request.Context(), product.ListFilter{
		Limit:  limit,
		Offset: offset,
		Query:  c.Query("q"),
	})
	writeResult(c, http.StatusOK, list, err)
}

func (h *Handler) getProduct(c *gin.Context) {
	found, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	writeResult(c, http.StatusOK, found, err)
}

func (h *Handler) updateStock(c *gin.Context) {
	var input product.UpdateStockInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	updated, err := h.service.UpdateStock(c.Request.Context(), c.Param("id"), input.Stock)
	writeResult(c, http.StatusOK, updated, err)
}

func (h *Handler) addImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image form file is required"})
		return
	}
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot open image"})
		return
	}
	defer opened.Close()
	content, err := io.ReadAll(io.LimitReader(opened, 10<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read image"})
		return
	}
	image, err := h.service.AddImage(c.Request.Context(), product.ImageInput{
		ProductID:   c.Param("id"),
		Filename:    file.Filename,
		ContentType: file.Header.Get("Content-Type"),
		Content:     content,
	})
	writeResult(c, http.StatusCreated, image, err)
}

func writeResult(c *gin.Context, status int, value any, err error) {
	if err != nil {
		switch {
		case errors.Is(err, product.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, product.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(status, value)
}
