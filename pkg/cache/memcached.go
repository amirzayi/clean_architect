package cache

import (
	"context"
	"errors"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

type memCache struct {
	client *memcache.Client
	prefix string
}

func NewMemCachedDriver(client *memcache.Client, prefix string) Driver {
	return memCache{client: client, prefix: prefix}
}

func (m memCache) Set(_ context.Context, key string, data []byte, ttl time.Duration) error {
	return m.client.Set(&memcache.Item{Key: m.prefix + key, Value: data, Expiration: int32(ttl.Seconds())})
}

func (m memCache) Get(_ context.Context, key string) (data []byte, err error) {
	v, err := m.client.Get(m.prefix + key)
	if err != nil {
		if errors.Is(err, memcache.ErrCacheMiss) {
			return nil, ErrCacheMissed
		}
		return nil, err
	}
	return v.Value, nil
}

func (m memCache) Delete(_ context.Context, key string) error {
	return m.client.Delete(m.prefix + key)
}
