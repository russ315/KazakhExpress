package userservice

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct{ client Client }

func RegisterRoutes(router gin.IRouter, client Client) {
	h := &Handler{client: client}
	router.POST("/auth/register", h.register)
	router.POST("/auth/login", h.login)
	router.POST("/auth/refresh", h.refresh)
	router.POST("/auth/logout", h.logout)
	router.GET("/users/me", h.me)
	router.PUT("/users/me", h.updateMe)
	router.GET("/users/:id", h.getUser)
}

func (h *Handler) register(c *gin.Context) {
	var input RegisterRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	result, err := h.client.Register(c.Request.Context(), input)
	writeResult(c, http.StatusCreated, result, err)
}

func (h *Handler) login(c *gin.Context) {
	var input LoginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	result, err := h.client.Login(c.Request.Context(), input)
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) refresh(c *gin.Context) {
	var input RefreshRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	result, err := h.client.Refresh(c.Request.Context(), input.RefreshToken)
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) logout(c *gin.Context) {
	var input LogoutRequest
	_ = c.ShouldBindJSON(&input)
	if input.AccessToken == "" {
		input.AccessToken = bearer(c)
	}
	err := h.client.Logout(c.Request.Context(), input)
	writeResult(c, http.StatusOK, gin.H{"ok": true}, err)
}

func (h *Handler) me(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	result, err := h.client.GetUser(c.Request.Context(), userID)
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) updateMe(c *gin.Context) {
	var input UpdateProfileRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	input.UserID = c.GetHeader("X-User-ID")
	result, err := h.client.UpdateProfile(c.Request.Context(), input)
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) getUser(c *gin.Context) {
	result, err := h.client.GetUser(c.Request.Context(), c.Param("id"))
	writeResult(c, http.StatusOK, result, err)
}

func bearer(c *gin.Context) string {
	value := c.GetHeader("Authorization")
	return strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
}

func writeResult(c *gin.Context, status int, value any, err error) {
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(status, value)
}
