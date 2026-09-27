package cache

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"time"
)

var (
	ErrCacheMissed = errors.New("cache missed")
)

type Driver interface {
	Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) (data []byte, err error)
	Delete(ctx context.Context, key string) error
}

type Cache struct {
	drv Driver
}

func New(drv Driver) *Cache {
	return &Cache{
		drv: drv,
	}
}

func (c Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return err
	}
	return c.drv.Set(ctx, key, buf.Bytes(), ttl)
}

func (c Cache) Get[T any](ctx context.Context, key string) (T, error) {
	var v T

	b, err := c.drv.Get(ctx, key)
	if err != nil {
		return v, err
	}

	if err = gob.NewDecoder(bytes.NewReader(b)).Decode(&v); err != nil {
		return v, err
	}
	return v, nil
}

func (c Cache) Delete(ctx context.Context, key string) error {
	return c.drv.Delete(ctx, key)
}
