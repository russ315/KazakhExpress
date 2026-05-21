package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisIdempotencyStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisIdempotencyStore(client *redis.Client, ttl time.Duration) *RedisIdempotencyStore {
	return &RedisIdempotencyStore{client: client, ttl: ttl}
}

func (s *RedisIdempotencyStore) GetPaymentID(ctx context.Context, key string) (string, bool, error) {
	value, err := s.client.Get(ctx, s.redisKey(key)).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get idempotency key: %w", err)
	}
	return value, true, nil
}

func (s *RedisIdempotencyStore) SavePaymentID(ctx context.Context, key string, paymentID string) error {
	if err := s.client.Set(ctx, s.redisKey(key), paymentID, s.ttl).Err(); err != nil {
		return fmt.Errorf("save idempotency key: %w", err)
	}
	return nil
}

func (s *RedisIdempotencyStore) redisKey(key string) string {
	return "payment:idempotency:" + key
}
