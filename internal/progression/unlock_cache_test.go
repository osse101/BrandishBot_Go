package progression

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnlockCache_GetSet(t *testing.T) {
	t.Parallel()

	t.Run("cache miss for new key", func(t *testing.T) {
		t.Parallel()
		cache := NewUnlockCache()
		_, found := cache.Get("feature_search", 1)
		assert.False(t, found, "Expected cache miss for new key")
	})

	t.Run("cache set and hit", func(t *testing.T) {
		t.Parallel()
		cache := NewUnlockCache()
		cache.Set("feature_search", 1, true)
		unlocked, found := cache.Get("feature_search", 1)
		assert.True(t, found, "Expected cache hit after set")
		assert.True(t, unlocked, "Expected unlocked=true")
	})

	t.Run("different levels are separate", func(t *testing.T) {
		t.Parallel()
		cache := NewUnlockCache()
		cache.Set("feature_search", 1, true)
		cache.Set("feature_search", 2, false)

		unlocked1, found1 := cache.Get("feature_search", 1)
		assert.True(t, found1)
		assert.True(t, unlocked1)

		unlocked2, found2 := cache.Get("feature_search", 2)
		assert.True(t, found2)
		assert.False(t, unlocked2)
	})
}

func TestUnlockCache_InvalidateAll(t *testing.T) {
	t.Parallel()

	cache := NewUnlockCache()

	// Populate cache
	cache.Set("feature_search", 1, true)
	cache.Set("feature_gamble", 1, true)
	cache.Set("item_money", 1, false)

	assert.Equal(t, 3, cache.Size(), "Expected size 3 before invalidation")

	// Invalidate all
	cache.InvalidateAll()

	assert.Equal(t, 0, cache.Size(), "Expected size 0 after invalidation")

	// Verify cache misses
	_, found := cache.Get("feature_search", 1)
	assert.False(t, found, "Expected cache miss after invalidation")
}

func TestUnlockCache_KeyFormat(t *testing.T) {
	t.Parallel()

	cache := NewUnlockCache()

	// Test that key format properly separates node and level
	cache.Set("feature", 1, true)
	cache.Set("feature", 2, false)

	val1, found1 := cache.Get("feature", 1)
	val2, found2 := cache.Get("feature", 2)

	assert.True(t, found1, "Entry 1 should be found")
	assert.True(t, found2, "Entry 2 should be found")
	assert.True(t, val1, "Value 1 should be true")
	assert.False(t, val2, "Value 2 should be false")
}

func TestUnlockCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	cache := NewUnlockCache()
	var wg sync.WaitGroup

	// Test concurrent writes, reads, size checks, and invalidations don't panic
	numGoroutines := 50

	wg.Add(numGoroutines * 4) // 4 operations per index

	for i := 0; i < numGoroutines; i++ {
		// Concurrent Set
		go func(n int) {
			defer wg.Done()
			cache.Set("concurrent", n, n%2 == 0)
		}(i)

		// Concurrent Get
		go func(n int) {
			defer wg.Done()
			cache.Get("concurrent", n)
		}(i)

		// Concurrent Size Check
		go func() {
			defer wg.Done()
			cache.Size()
		}()

		// Periodic InvalidateAll (less frequent so some entries remain)
		go func(n int) {
			defer wg.Done()
			if n%10 == 0 {
				cache.InvalidateAll()
			}
		}(i)
	}

	wg.Wait()

	// Test completes without panicking - assert that we can still interact with cache
	cache.Set("final_test", 1, true)
	val, found := cache.Get("final_test", 1)
	assert.True(t, found)
	assert.True(t, val)
}
