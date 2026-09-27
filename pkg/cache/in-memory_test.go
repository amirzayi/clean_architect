package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/amirzayi/clean_architect/pkg/cache"
	"github.com/stretchr/testify/require"
)

func TestMemCache(t *testing.T) {
	ctx := context.Background()

	drv := cache.NewInMemoryDriver()
	ch := cache.New(drv)

	for _, tc := range []struct {
		name, key, value, expectedValue string
		ttl, sleep                      time.Duration
		expectedError                   error
	}{
		{"cache hit", "some_key", "some_value", "some_value", 0, 0, nil},
		{"cache miss", "some_key", "some_value", "", time.Second, time.Second * 2, cache.ErrCacheMissed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ch.Set(ctx, tc.key, tc.value, tc.ttl)
			require.NoError(t, err)

			time.Sleep(tc.sleep)
			data, err := ch.Get[string](ctx, tc.key)
			require.Equal(t, tc.expectedError, err)
			require.Equal(t, tc.expectedValue, data)
		})
	}
}
