package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisRateLimiter(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	limiter := NewRedisRateLimiter(client)
	ctx := context.Background()
	key := "gateway:ratelimit:test"

	first, err := limiter.Allow(ctx, key, 2, time.Minute)
	if err != nil {
		t.Fatalf("first allow: %v", err)
	}
	if !first.Allowed || first.Remaining != 1 {
		t.Fatalf("first result = %+v, want allowed with 1 remaining", first)
	}

	second, err := limiter.Allow(ctx, key, 2, time.Minute)
	if err != nil {
		t.Fatalf("second allow: %v", err)
	}
	if !second.Allowed || second.Remaining != 0 {
		t.Fatalf("second result = %+v, want allowed with 0 remaining", second)
	}

	third, err := limiter.Allow(ctx, key, 2, time.Minute)
	if err != nil {
		t.Fatalf("third allow: %v", err)
	}
	if third.Allowed || third.RetryAfter <= 0 {
		t.Fatalf("third result = %+v, want rejected with retry after", third)
	}
}
