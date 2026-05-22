package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kazakhexpress/review-service/internal/review"

	"github.com/redis/go-redis/v9"
)

const defaultTTL = 5 * time.Minute

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

func ratingKey(productID string) string {
	return "product:" + productID + ":rating"
}

func reviewsPageKey(productID string, page int) string {
	return fmt.Sprintf("product:%s:reviews:page:%d", productID, page)
}

func (a *CacheAdapter) GetRating(ctx context.Context, productID string) (review.ProductRating, bool, error) {
	val, err := a.client.rdb.Get(ctx, ratingKey(productID)).Bytes()
	if err == redis.Nil {
		return review.ProductRating{}, false, nil
	}
	if err != nil {
		return review.ProductRating{}, false, err
	}
	var rating review.ProductRating
	if err := json.Unmarshal(val, &rating); err != nil {
		return review.ProductRating{}, false, err
	}
	return rating, true, nil
}

func (a *CacheAdapter) SetRating(ctx context.Context, rating review.ProductRating, ttl time.Duration) error {
	if ttl == 0 {
		ttl = defaultTTL
	}
	bytes, err := json.Marshal(rating)
	if err != nil {
		return err
	}
	return a.client.rdb.Set(ctx, ratingKey(rating.ProductID), bytes, ttl).Err()
}

func (a *CacheAdapter) InvalidateRating(ctx context.Context, productID string) error {
	return a.client.rdb.Del(ctx, ratingKey(productID)).Err()
}

func (a *CacheAdapter) GetReviewsPage(ctx context.Context, productID string, page int) (review.ListPage, bool, error) {
	val, err := a.client.rdb.Get(ctx, reviewsPageKey(productID, page)).Bytes()
	if err == redis.Nil {
		return review.ListPage{}, false, nil
	}
	if err != nil {
		return review.ListPage{}, false, err
	}
	var result review.ListPage
	if err := json.Unmarshal(val, &result); err != nil {
		return review.ListPage{}, false, err
	}
	return result, true, nil
}

func (a *CacheAdapter) SetReviewsPage(ctx context.Context, productID string, page review.ListPage, ttl time.Duration) error {
	if ttl == 0 {
		ttl = defaultTTL
	}
	bytes, err := json.Marshal(page)
	if err != nil {
		return err
	}
	return a.client.rdb.Set(ctx, reviewsPageKey(productID, page.Page), bytes, ttl).Err()
}

func (a *CacheAdapter) InvalidateReviews(ctx context.Context, productID string) error {
	iter := a.client.rdb.Scan(ctx, 0, "product:"+productID+":reviews:page:*", 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return a.client.rdb.Del(ctx, keys...).Err()
}
