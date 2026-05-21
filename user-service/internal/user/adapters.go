package user

import (
	"context"
	"time"

	redisclient "kazakhexpress/user-service/internal/redis"
)

type RedisCacheAdapter struct {
	client *redisclient.Client
}

func NewRedisCacheAdapter(client *redisclient.Client) *RedisCacheAdapter {
	return &RedisCacheAdapter{client: client}
}

func (a *RedisCacheAdapter) GetCachedUser(ctx context.Context, userID string, dest interface{}) error {
	return a.client.GetCachedUser(ctx, userID, dest)
}

func (a *RedisCacheAdapter) CacheUser(ctx context.Context, userID string, data interface{}, ttl time.Duration) error {
	return a.client.CacheUser(ctx, userID, data, ttl)
}

func (a *RedisCacheAdapter) InvalidateUserCache(ctx context.Context, userID string) error {
	return a.client.InvalidateUserCache(ctx, userID)
}

func (a *RedisCacheAdapter) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	return a.client.BlacklistToken(ctx, jti, ttl)
}

func (a *RedisCacheAdapter) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	return a.client.IsTokenBlacklisted(ctx, jti)
}

type RedisRateLimitAdapter struct {
	client *redisclient.Client
}

func NewRedisRateLimitAdapter(client *redisclient.Client) *RedisRateLimitAdapter {
	return &RedisRateLimitAdapter{client: client}
}

func (a *RedisRateLimitAdapter) CheckLoginRateLimit(ctx context.Context, identifier string, maxAttempts int, window time.Duration) (int, error) {
	return a.client.CheckLoginRateLimit(ctx, identifier, maxAttempts, window)
}

func (a *RedisRateLimitAdapter) ResetLoginRateLimit(ctx context.Context, identifier string) error {
	return a.client.ResetLoginRateLimit(ctx, identifier)
}
