package progression

import (
	"testing"
)

func TestUnlockCache_GetSet(t *testing.T) {
	cache := NewUnlockCache()

	// Test cache miss
	_, found := cache.Get("feature_search", 1)
	if found {
		t.Error("Expected cache miss for new key")
	}

	// Test cache set and hit
	cache.Set("feature_search", 1, true)
	unlocked, found := cache.Get("feature_search", 1)
	if !found {
		t.Error("Expected cache hit after set")
	}
	if !unlocked {
		t.Error("Expected unlocked=true")
	}

	// Test different levels are separate
	cache.Set("feature_search", 2, false)
	unlocked1, _ := cache.Get("feature_search", 1)
	unlocked2, _ := cache.Get("feature_search", 2)
	if unlocked1 != true || unlocked2 != false {
		t.Error("Different levels should have separate cache entries")
	}
}

func TestUnlockCache_InvalidateAll(t *testing.T) {
	cache := NewUnlockCache()

	// Populate cache
	cache.Set("feature_search", 1, true)
	cache.Set("feature_gamble", 1, true)
	cache.Set("item_money", 1, false)

	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	// Invalidate all
	cache.InvalidateAll()

	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after invalidation, got %d", cache.Size())
	}

	// Verify cache misses
	_, found := cache.Get("feature_search", 1)
	if found {
		t.Error("Expected cache miss after invalidation")
	}
}

func TestUnlockCache_KeyFormat(t *testing.T) {
	cache := NewUnlockCache()

	// Test that key format properly separates node and level
	cache.Set("feature", 1, true)
	cache.Set("feature", 2, false)

	val1, found1 := cache.Get("feature", 1)
	val2, found2 := cache.Get("feature", 2)

	if !found1 || !found2 {
		t.Error("Both entries should be found")
	}

	if val1 != true || val2 != false {
		t.Error("Values should be stored independently")
	}
}

func TestUnlockCache_ConcurrentAccess(t *testing.T) {
	cache := NewUnlockCache()

	// Test concurrent writes don't panic
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			cache.Set("concurrent", n, true)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have entries without panicking
	if cache.Size() == 0 {
		t.Error("Expected some cache entries after concurrent writes")
	}
}
