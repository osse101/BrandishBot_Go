package user

import (
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestCacheInvalidation(t *testing.T) {
	// Setup
	config := CacheConfig{Size: 10, TTL: 1 * time.Minute}
	cache := newUserCache(config)

	user := &domain.User{
		ID:       "user-1",
		Username: "testuser",
		TwitchID: "twitch-123",
	}

	// 1. Set user in cache
	cache.Set(domain.PlatformTwitch, "twitch-123", user)

	// 2. Verify retrieval
	retrieved, found := cache.Get(domain.PlatformTwitch, "twitch-123")
	assert.True(t, found)
	assert.Equal(t, user, retrieved)

	// 3. Invalidate
	cache.Invalidate(domain.PlatformTwitch, "twitch-123")

	// 4. Verify miss
	retrieved, found = cache.Get(domain.PlatformTwitch, "twitch-123")
	assert.False(t, found)
	assert.Nil(t, retrieved)
}

func TestCacheStats(t *testing.T) {
	config := CacheConfig{Size: 10, TTL: 1 * time.Minute}
	cache := newUserCache(config)

	user := &domain.User{
		ID:       "user-1",
		Username: "testuser",
	}

	// Initial stats
	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, 0, stats.Size)

	// Miss
	cache.Get("platform", "id")
	stats = cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)

	// Set and Hit
	cache.Set("platform", "id", user)
	cache.Get("platform", "id")
	stats = cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, 1, stats.Size)
}

func TestCacheConfig(t *testing.T) {
	// Test Default
	cfg := DefaultCacheConfig()
	assert.Equal(t, 1000, cfg.Size)
	assert.Equal(t, 5*time.Minute, cfg.TTL)
}
