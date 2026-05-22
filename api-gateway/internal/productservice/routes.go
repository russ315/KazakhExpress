package productservice

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct{ client Client }

func RegisterRoutes(router gin.IRouter, client Client) {
	h := &Handler{client: client}
	router.GET("/products/health", h.health)
	router.POST("/products", h.create)
	router.GET("/products", h.list)
	router.GET("/products/:productId", h.get)
	router.PATCH("/products/:productId/stock", h.updateStock)
	router.POST("/products/:productId/images", h.addImage)
}

func (h *Handler) health(c *gin.Context) {
	err := h.client.Health(c.Request.Context())
	writeResult(c, http.StatusOK, gin.H{"service": "product", "status": "ok"}, err)
}

func (h *Handler) create(c *gin.Context) {
	var input CreateProductRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	result, err := h.client.Create(c.Request.Context(), input)
	writeResult(c, http.StatusCreated, result, err)
}

func (h *Handler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	result, err := h.client.List(c.Request.Context(), limit, offset, c.Query("q"))
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) get(c *gin.Context) {
	result, err := h.client.Get(c.Request.Context(), c.Param("productId"))
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) updateStock(c *gin.Context) {
	var input UpdateStockRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	result, err := h.client.UpdateStock(c.Request.Context(), c.Param("productId"), input.Stock)
	writeResult(c, http.StatusOK, result, err)
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
	result, err := h.client.AddImage(c.Request.Context(), c.Param("productId"), file.Filename, file.Header.Get("Content-Type"), content)
	writeResult(c, http.StatusCreated, result, err)
}

func writeResult(c *gin.Context, status int, value any, err error) {
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(status, value)
}
