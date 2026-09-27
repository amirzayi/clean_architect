package synq

import (
	"context"
	"time"
)

// Get retrieves a value by key. If the key is not in the cache (or expired),
// it calls getFn to fetch the data, stores it in the cache, and returns it.
//
// Args:
//   - ctx: Context for cancellation/timeout.
//   - key: Cache key to retrieve.
//   - getFn: Function that fetches data if cache misses.
//   - ttl: time to live data in cache storage.
//
// Returns:
//   - The cached or freshly fetched value.
//   - Error if the operation fails (e.g., getFn fails or cache is unreachable).
func (cr CacheSync) Get[T any](ctx context.Context, key string, getFn func() (T, error), ttl time.Duration) (T, error) {
	data, err := cr.cache.Get[T](ctx, key)
	if err == nil {
		return data, nil
	}
	data, err = getFn()
	if err == nil {
		cr.cache.Set(ctx, key, data, ttl)
	}
	return data, err
}

// Set overwrite a key in the cache and optionally syncs it to the underlying storage via setFn.
// If setFn is provided, it is called to persist the data before updating the cache.
//
// Args:
//   - ctx: Context for cancellation/timeout.
//   - key: Cache key to update.
//   - value: Value to store.
//   - setFn: function to persist the data (e.g., DB write).
//
// Returns:
//   - Error if the operation fails (e.g., setFn fails or cache write fails).
func (cr CacheSync) Set[T any](ctx context.Context, key string, value T, setFn func() error, ttl time.Duration) error {
	if err := setFn(); err != nil {
		return err
	}
	return cr.cache.Set(ctx, key, value, ttl)
}

// Delete removes a key from the cache for cache invalidation after calling deleteFn.
//
// Args:
//   - ctx: Context for cancellation/timeout.
//   - key: Cache key to delete.
//   - deleteFn: function to delete data from storage (e.g., DB delete).
//
// Returns:
//   - Error if the operation fails (e.g., deleteFn fails or cache deletion fails).
func (cr CacheSync) Delete(ctx context.Context, key string, deleteFn func() error) error {
	if err := deleteFn(); err != nil {
		return err
	}
	return cr.cache.Delete(ctx, key)
}
