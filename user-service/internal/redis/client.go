package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	blacklistPrefix = "blacklist:"
	rateLimitPrefix = "ratelimit:login:"
	cachePrefix     = "user:"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(addr string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	return c.rdb.Set(ctx, blacklistPrefix+jti, "1", ttl).Err()
}

func (c *Client) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	val, err := c.rdb.Exists(ctx, blacklistPrefix+jti).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

func (c *Client) CheckLoginRateLimit(ctx context.Context, identifier string, maxAttempts int, window time.Duration) (int, error) {
	key := rateLimitPrefix + identifier
	now := time.Now().UnixNano()

	pipe := c.rdb.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", now-window.Nanoseconds()))
	count := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	attempts, err := count.Result()
	if err != nil {
		return 0, err
	}

	return int(attempts), nil
}

func (c *Client) ResetLoginRateLimit(ctx context.Context, identifier string) error {
	return c.rdb.Del(ctx, rateLimitPrefix+identifier).Err()
}

func (c *Client) CacheUser(ctx context.Context, userID string, data interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, cachePrefix+userID, bytes, ttl).Err()
}

func (c *Client) GetCachedUser(ctx context.Context, userID string, dest interface{}) error {
	val, err := c.rdb.Get(ctx, cachePrefix+userID).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache miss")
		}
		return err
	}
	return json.Unmarshal(val, dest)
}

func (c *Client) InvalidateUserCache(ctx context.Context, userID string) error {
	return c.rdb.Del(ctx, cachePrefix+userID).Err()
}

func (c *Client) ExtractJTI(tokenString string) string {
	return tokenString
}
