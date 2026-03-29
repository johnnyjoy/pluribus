package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store is a small cache interface. Redis is the only implementation; do not make it authoritative.
type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// DeleteByPrefix removes all keys with the given prefix (e.g. for invalidation). No-op if unsupported.
	DeleteByPrefix(ctx context.Context, prefix string) error
}

// RedisStore implements Store using Redis.
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedis creates a Redis-backed cache store. defaultTTL is used when Set is called with ttl <= 0.
func NewRedis(addr, password string, db int, defaultTTL time.Duration) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &RedisStore{client: client, ttl: defaultTTL}, nil
}

// Get returns the value for key. Returns (nil, nil) when key does not exist (cache miss).
func (r *RedisStore) Get(ctx context.Context, key string) ([]byte, error) {
	b, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Set stores value at key with the given ttl. If ttl <= 0, the store's default TTL is used.
func (r *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = r.ttl
	}
	return r.client.Set(ctx, key, value, ttl).Err()
}

// DeleteByPrefix removes all keys matching prefix+"*". Used to invalidate memory search cache for a project.
func (r *RedisStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	keys, err := r.client.Keys(ctx, prefix+"*").Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

// Close closes the Redis connection. Call when shutting down.
func (r *RedisStore) Close() error {
	return r.client.Close()
}
