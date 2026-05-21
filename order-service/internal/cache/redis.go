package cache

import (
	"context"
	"fmt"
	"time"

	"kazakhexpress/order-service/internal/order"

	"github.com/redis/go-redis/v9"
)

type RedisStatusCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisStatusCache(client *redis.Client, ttl time.Duration) *RedisStatusCache {
	return &RedisStatusCache{client: client, ttl: ttl}
}

func (c *RedisStatusCache) SetStatus(ctx context.Context, orderID string, status order.Status) error {
	return c.client.Set(ctx, fmt.Sprintf("order:%s:status", orderID), string(status), c.ttl).Err()
}
