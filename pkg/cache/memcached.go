package cache

import (
	"context"
	"errors"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

type memCache struct {
	Client *memcache.Client
	Prefix string
}

func NewMemCachedDriver(client *memcache.Client, prefix string) Driver {
	return memCache{Client: client, Prefix: prefix}
}

func (m memCache) Set(_ context.Context, key string, data []byte, ttl time.Duration) error {
	return m.Client.Set(&memcache.Item{Key: m.Prefix + key, Value: data, Expiration: int32(ttl.Seconds())})
}

func (m memCache) Get(_ context.Context, key string) (data []byte, err error) {
	v, err := m.Client.Get(m.Prefix + key)
	if err != nil {
		if errors.Is(err, memcache.ErrCacheMiss) {
			return nil, ErrCacheMissed
		}
		return nil, err
	}
	return v.Value, nil
}

func (m memCache) Delete(_ context.Context, key string) error {
	return m.Client.Delete(m.Prefix + key)
}
