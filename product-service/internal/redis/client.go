package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kazakhexpress/product-service/internal/product"

	"github.com/redis/go-redis/v9"
)

const productPrefix = "product:"

type Client struct {
	rdb *redis.Client
}

func NewClient(addr string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 0})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

type CacheAdapter struct {
	client *Client
}

func NewCacheAdapter(client *Client) *CacheAdapter {
	return &CacheAdapter{client: client}
}

func (a *CacheAdapter) GetProduct(ctx context.Context, id string) (product.Product, bool, error) {
	val, err := a.client.rdb.Get(ctx, productPrefix+id).Bytes()
	if err == redis.Nil {
		return product.Product{}, false, nil
	}
	if err != nil {
		return product.Product{}, false, err
	}
	var p product.Product
	if err := json.Unmarshal(val, &p); err != nil {
		return product.Product{}, false, err
	}
	return p, true, nil
}

func (a *CacheAdapter) SetProduct(ctx context.Context, p product.Product, ttl time.Duration) error {
	bytes, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return a.client.rdb.Set(ctx, productPrefix+p.ID, bytes, ttl).Err()
}

func (a *CacheAdapter) InvalidateProduct(ctx context.Context, id string) error {
	return a.client.rdb.Del(ctx, productPrefix+id).Err()
}
