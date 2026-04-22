package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
	prefix string
}

func NewRedisDriver(client *redis.Client, prefix string) Driver {
	return redisCache{client: client, prefix: prefix}
}

func (r redisCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return r.client.Set(ctx, r.prefix+key, data, ttl).Err()
}

func (r redisCache) Get(ctx context.Context, key string) ([]byte, error) {
	cmd := r.client.Get(ctx, r.prefix+key)
	if err := cmd.Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMissed
		}
		return nil, err
	}
	return cmd.Bytes()
}

func (r redisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.prefix+key).Err()
}

func (r redisCache) Ping(ctx context.Context) error {
	status := r.client.Ping(ctx)
	return status.Err()
}

func (r redisCache) Close() error {
	return r.client.Close()
}
