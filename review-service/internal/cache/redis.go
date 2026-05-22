package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kazakhexpress/review-service/internal/review"

	"github.com/redis/go-redis/v9"
)

type RedisRatingCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisRatingCache(client *redis.Client, ttl time.Duration) *RedisRatingCache {
	return &RedisRatingCache{client: client, ttl: ttl}
}

func (c *RedisRatingCache) GetRating(ctx context.Context, productID string) (review.Rating, bool, error) {
	raw, err := c.client.Get(ctx, key(productID)).Bytes()
	if err == redis.Nil {
		return review.Rating{}, false, nil
	}
	if err != nil {
		return review.Rating{}, false, err
	}
	var rating review.Rating
	if err := json.Unmarshal(raw, &rating); err != nil {
		return review.Rating{}, false, err
	}
	return rating, true, nil
}

func (c *RedisRatingCache) SetRating(ctx context.Context, rating review.Rating) error {
	raw, err := json.Marshal(rating)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key(rating.ProductID), raw, c.ttl).Err()
}

func (c *RedisRatingCache) DeleteRating(ctx context.Context, productID string) error {
	return c.client.Del(ctx, key(productID)).Err()
}

func key(productID string) string {
	return fmt.Sprintf("product:%s:rating", productID)
}
