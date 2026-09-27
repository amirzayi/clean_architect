package synq

import (
	"context"
	"log/slog"
	"time"
)

// GetAsync is similar to Get but executes cache fulfillment asynchronously.
//
// Args:
//   - ctx: Context for cancellation/timeout.
//   - key: Cache key to retrieve.
//   - getFn: Function that fetches data if cache misses (executed in background).
//   - ttl: time to live data in cache storage.
//
// Returns:
//   - The cached or freshly fetched value.
//   - Error only if the getFn fails (cache fetch errors are logged).
func (cr CacheSync) GetAsync[T any](ctx context.Context, key string, getFn func() (T, error), ttl time.Duration) (T, error) {
	data, err := cr.cache.Get[T](ctx, key)
	if err == nil {
		return data, nil
	}
	data, err = getFn()
	if err != nil {
		return data, err
	}
	go func() {
		if err = cr.cache.Set(context.Background(), key, data, ttl); err != nil {
			cr.logger.Error("failed to set cache", slog.String("key", key), slog.Any("error", err))
		}
	}()
	return data, nil
}

// SetAsync is similar to Set but executes cache fulfillment asynchronously.
//
// Args:
//   - key: Cache key to update.
//   - value: Value to store.
//   - setFn: function to persist the data (e.g., DB write).
//   - ttl: time to live data in cache storage.
//
// Returns:
//   - Error only if the cache update fails (cache fulfillment errors are logged).
func (cr CacheSync) SetAsync[T any](key string, value T, setFn func() error, ttl time.Duration) error {
	if err := setFn(); err != nil {
		return err
	}
	go func() {
		if err := cr.cache.Set(context.Background(), key, value, ttl); err != nil {
			cr.logger.Error("failed to set cache", slog.String("key", key), slog.Any("error", err))
		}
	}()
	return nil
}

// DeleteAsync is similar to Delete but executes cache removal asynchronously.
//
// Args:
//   - key: Cache key to delete.
//   - deleteFn: Function to delete data from storage.
//
// Returns:
//   - Error only if the cache deletion fails (cache removal errors are logged).
func (cr CacheSync) DeleteAsync(key string, deleteFn func() error) error {
	if err := deleteFn(); err != nil {
		return err
	}
	go func() {
		if err := cr.cache.Delete(context.Background(), key); err != nil {
			cr.logger.Error("failed to delete cache", slog.String("key", key), slog.Any("error", err))
		}
	}()
	return nil
}
