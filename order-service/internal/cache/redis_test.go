package cache

import (
	"context"
	"testing"
	"time"

	"kazakhexpress/order-service/internal/order"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisStatusCacheSetStatus(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisStatusCache(client, time.Minute)

	ctx := context.Background()
	if err := cache.SetStatus(ctx, "ord-1", order.StatusPaid); err != nil {
		t.Fatalf("SetStatus() error = %v", err)
	}

	got, err := client.Get(ctx, "order:ord-1:status").Result()
	if err != nil {
		t.Fatalf("redis get: %v", err)
	}
	if got != string(order.StatusPaid) {
		t.Fatalf("cached value = %q, want %q", got, order.StatusPaid)
	}
}
