package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	client *redis.Client
}

func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

func (l *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (RateLimitResult, error) {
	if l == nil || l.client == nil {
		return RateLimitResult{}, fmt.Errorf("redis rate limiter is not configured")
	}
	pipe := l.client.TxPipeline()
	countCmd := pipe.Incr(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return RateLimitResult{}, fmt.Errorf("redis rate limit increment: %w", err)
	}
	count := int(countCmd.Val())
	ttl := ttlCmd.Val()
	if count == 1 || ttl < 0 {
		if err := l.client.Expire(ctx, key, window).Err(); err != nil {
			return RateLimitResult{}, fmt.Errorf("redis rate limit expire: %w", err)
		}
		ttl = window
	}
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}
	return RateLimitResult{
		Allowed:    count <= limit,
		Remaining:  remaining,
		RetryAfter: ttl,
	}, nil
}
