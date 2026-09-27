// Package synq provides a seamless way to synchronize a cache with a database
// in a database-first manner. It ensures consistency by always prioritizing the database
// as the source of truth, while keeping the cache updated for performance.
//
// Key Features:
//   - **Database-First**: All operations (Get/Set/Delete) first modify the database
//     (via provided callbacks) before updating the cache.
//   - **Sync & Async Modes**: Supports both blocking (strong consistency) and
//     non-blocking (eventual consistency) operations.
//   - **CRUD Integration**: Wraps database calls with automatic cache management,
//     reducing boilerplate for read-through/write-through patterns.
//
// Usage Example:
//
//	store := synq.New(redisClient, slog.Default())
//	user, err := store.Get(ctx, "user123", func() (User, error) {
//	    return db.GetUserByID(ctx, "user123") // Database fetch
//	})
//
// Use Async methods (e.g., SetAsync) for write-heavy workloads where
// latency matters more than immediate consistency.
package synq

import (
	"log/slog"

	"github.com/amirzayi/clean_architect/pkg/cache"
)

// CacheSync provides synchronized CRUD operations with caching support.
// It ensures cache consistency by invoking the given functions (getFn, setFn, deleteFn)
// to sync data between the cache and the underlying storage (e.g., DB, API).
type CacheSync struct {
	cache  *cache.Cache
	logger *slog.Logger
}

func New(cache *cache.Cache, logger *slog.Logger) CacheSync {
	return CacheSync{
		cache:  cache,
		logger: logger,
	}
}
