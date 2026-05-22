package userservice

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
	h := &Handler{client: client}

	auth := router.Group("/auth")
	auth.GET("/health", h.health)
	auth.POST("/register", h.register)
	auth.POST("/login", h.login)
	auth.POST("/refresh", h.refreshToken)
	auth.POST("/logout", h.logout)
	auth.POST("/forgot-password", h.forgotPassword)
	auth.POST("/reset-password", h.resetPassword)

	users := router.Group("/users")
	users.GET("/:id", h.getUser)
	users.GET("/me", h.getProfile)
	users.PUT("/me", h.updateProfile)
	users.PATCH("/me", h.updateProfile)
}

func (h *Handler) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.client.Health(ctx); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"service": "user",
			"status":  "unavailable",
			"error":   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"service": "user", "status": "ok"})
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

func (h *Handler) refreshToken(c *gin.Context) {
	var input RefreshTokenRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	result, err := h.client.RefreshToken(c.Request.Context(), input.RefreshToken)
	writeResult(c, http.StatusOK, result, err)
}

func (h *Handler) logout(c *gin.Context) {
	var input LogoutRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if err := h.client.Logout(c.Request.Context(), input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

func (h *Handler) forgotPassword(c *gin.Context) {
	var input ForgotPasswordRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if err := h.client.ForgotPassword(c.Request.Context(), input.Email); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "if the email exists, a reset link will be sent"})
}

func (h *Handler) resetPassword(c *gin.Context) {
	var input ResetPasswordRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if err := h.client.ResetPassword(c.Request.Context(), input.Token, input.NewPassword); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "password reset successfully"})
}

func (h *Handler) getUser(c *gin.Context) {
	user, err := h.client.GetUserByID(c.Request.Context(), c.Param("id"))
	writeResult(c, http.StatusOK, user, err)
}

func (h *Handler) getProfile(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		token := extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}
		resp, err := h.client.ValidateToken(c.Request.Context(), token)
		if err != nil || !resp.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		userID = resp.UserID
	}
	user, err := h.client.GetUser(c.Request.Context(), userID)
	writeResult(c, http.StatusOK, user, err)
}

func (h *Handler) updateProfile(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		token := extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}
		resp, err := h.client.ValidateToken(c.Request.Context(), token)
		if err != nil || !resp.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		userID = resp.UserID
	}

	var input UpdateProfileRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	user, err := h.client.UpdateProfile(c.Request.Context(), userID, input)
	writeResult(c, http.StatusOK, user, err)
}

func extractToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if len(header) > 7 && header[:7] == "Bearer " {
		return header[7:]
	}
	return ""
}

func writeResult(c *gin.Context, status int, value any, err error) {
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(status, value)
}
