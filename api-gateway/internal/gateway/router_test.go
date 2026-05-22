package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRouterHealth(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRouterRateLimitAllowsRequests(t *testing.T) {
	router := NewRouter(WithRateLimiter(fakeLimiter{result: RateLimitResult{Allowed: true, Remaining: 1}}, 2, time.Minute))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-RateLimit-Limit"); got != "2" {
		t.Fatalf("rate limit header = %q, want 2", got)
	}
}

func TestRouterRateLimitRejectsRequests(t *testing.T) {
	router := NewRouter(WithRateLimiter(fakeLimiter{result: RateLimitResult{Allowed: false, Remaining: 0, RetryAfter: 30 * time.Second}}, 2, time.Minute))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if got := rec.Header().Get("Retry-After"); got != "30" {
		t.Fatalf("retry after = %q, want 30", got)
	}
}

func TestRouterRateLimitFailsOpen(t *testing.T) {
	router := NewRouter(WithRateLimiter(fakeLimiter{err: context.DeadlineExceeded}, 2, time.Minute))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

type fakeLimiter struct {
	result RateLimitResult
	err    error
}

func (f fakeLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (RateLimitResult, error) {
	return f.result, f.err
}
