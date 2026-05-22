package gateway

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (RateLimitResult, error)
}

type RateLimitResult struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}

type RouterOption func(*routerConfig)

type routerConfig struct {
	rateLimiter RateLimiter
	rateLimit   int
	rateWindow  time.Duration
}

func WithRateLimiter(limiter RateLimiter, limit int, window time.Duration) RouterOption {
	return func(config *routerConfig) {
		config.rateLimiter = limiter
		config.rateLimit = limit
		config.rateWindow = window
	}
}

func NewRouter(options ...RouterOption) *gin.Engine {
	config := routerConfig{
		rateLimit:  120,
		rateWindow: time.Minute,
	}
	for _, option := range options {
		option(&config)
	}
	router := gin.New()
	router.Use(requestLogger(), gin.Recovery(), cors())
	if config.rateLimiter != nil && config.rateLimit > 0 && config.rateWindow > 0 {
		router.Use(redisRateLimit(config.rateLimiter, config.rateLimit, config.rateWindow))
	}
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api-gateway"})
	})
	router.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, "kazakhexpress_service_up{service=\"api-gateway\"} 1\n")
	})
	return router
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = time.Now().UTC().Format("20060102150405.000000000")
		}
		c.Header("X-Request-ID", requestID)
		c.Next()
		slog.Info("http_request",
			"service", "api-gateway",
			"request_id", requestID,
			"method", c.Request.Method,
			"route", c.FullPath(),
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,Idempotency-Key")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func redisRateLimit(limiter RateLimiter, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "gateway:ratelimit:" + c.ClientIP() + ":" + c.Request.Method + ":" + c.Request.URL.Path
		result, err := limiter.Allow(c.Request.Context(), key, limit, window)
		if err != nil {
			slog.Warn("rate_limit_unavailable", "service", "api-gateway", "error", err)
			c.Next()
			return
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		if !result.Allowed {
			retryAfter := int(result.RetryAfter.Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter,
			})
			return
		}
		c.Next()
	}
}
