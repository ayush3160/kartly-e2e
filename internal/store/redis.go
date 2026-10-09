package store

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is Redis: product cache, sessions and idempotency keys.
type Cache struct{ rdb *redis.Client }

// NewCache connects to Redis.
func NewCache(addr string) *Cache {
	return &Cache{rdb: redis.NewClient(&redis.Options{Addr: addr})}
}

// Get returns a cached value; ok is false on a miss.
func (c *Cache) Get(ctx context.Context, key string) (string, bool, error) {
	v, err := c.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	return v, err == nil, err
}

// Set caches a value.
func (c *Cache) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, val, ttl).Err()
}

// Claim sets key only when absent: the idempotency check. false means the
// key was already claimed.
func (c *Cache) Claim(ctx context.Context, key, val string, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, key, val, ttl).Result()
}

// Incr counts a rate-limit window.
func (c *Cache) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	n, err := c.rdb.Incr(ctx, key).Result()
	if err == nil && n == 1 {
		err = c.rdb.Expire(ctx, key, ttl).Err()
	}
	return n, err
}

// Del removes keys.
func (c *Cache) Del(ctx context.Context, keys ...string) error { return c.rdb.Del(ctx, keys...).Err() }

// Close disconnects.
func (c *Cache) Close() { _ = c.rdb.Close() }
