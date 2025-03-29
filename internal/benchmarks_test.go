package internal

import (
	"fmt"
	"testing"
	"time"

	"github.com/littleironwaltz/bluesky-mcp/internal/cache"
)

// BenchmarkCacheOperations tests the performance of cache operations
func BenchmarkCacheOperations(b *testing.B) {
	cacheService := cache.New()
	defer cacheService.Stop()

	// Setup some initial data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cacheService.Set(key, i, 5*time.Minute)
	}

	b.Run("Get-Existing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i%1000)
			cacheService.Get(key)
		}
	})

	b.Run("Get-Nonexistent", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("nonexistent-key-%d", i)
			cacheService.Get(key)
		}
	})

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bench-key-%d", i)
			cacheService.Set(key, i, 5*time.Minute)
		}
	})

	b.Run("Delete", func(b *testing.B) {
		// Setup data for deletion
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("delete-key-%d", i)
			cacheService.Set(key, i, 5*time.Minute)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("delete-key-%d", i)
			cacheService.Delete(key)
		}
	})
}

// For now, we'll focus on just the cache benchmarks
// The other benchmarks require implementing several feed module functions
// which would be better created as part of the feed package development