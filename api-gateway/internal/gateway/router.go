package gateway

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

var (
	metricsOnce sync.Once

	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_http_requests_total",
			Help: "Total number of HTTP requests by service, route, status and method.",
		},
		[]string{"service", "method", "route", "status"},
	)
	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kazakhexpress_http_request_duration_seconds",
			Help:    "HTTP request latency by service, route, status and method.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "route", "status"},
	)
	httpInflight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kazakhexpress_http_in_flight_requests",
			Help: "Current in-flight HTTP requests by service.",
		},
		[]string{"service"},
	)
	gatewayRateLimitBlocks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_gateway_rate_limit_blocks_total",
			Help: "Total rate limit blocks at the API gateway.",
		},
		[]string{"client_ip", "path"},
	)
)

func initMetrics() {
	metricsOnce.Do(func() {
		prometheus.MustRegister(httpRequests, httpDuration, httpInflight, gatewayRateLimitBlocks)
	})
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
	initMetrics()
	router := gin.New()
	router.Use(metricsMiddleware("api-gateway"), requestLogger(), gin.Recovery(), cors())
	if config.rateLimiter != nil && config.rateLimit > 0 && config.rateWindow > 0 {
		router.Use(redisRateLimit(config.rateLimiter, config.rateLimit, config.rateWindow))
	}
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api-gateway"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	return router
}

func metricsMiddleware(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		httpInflight.WithLabelValues(service).Inc()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		status := strconv.Itoa(c.Writer.Status())
		httpRequests.WithLabelValues(service, c.Request.Method, route, status).Inc()
		httpDuration.WithLabelValues(service, c.Request.Method, route, status).Observe(time.Since(start).Seconds())
		httpInflight.WithLabelValues(service).Dec()
	}
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
			gatewayRateLimitBlocks.WithLabelValues(c.ClientIP(), c.Request.URL.Path).Inc()
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
