package user

import (
	"sync/atomic"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// CacheSchemaVersion is the current version of the cache schema
// Increment this when the cached data structure changes to auto-invalidate old entries
const CacheSchemaVersion = "1.0"

// CacheConfig defines configuration for the user cache
type CacheConfig struct {
	Size int           // Maximum number of entries
	TTL  time.Duration // Time-to-live for entries
}

// DefaultCacheConfig returns the default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Size: 1000,
		TTL:  5 * time.Minute,
	}
}

// CacheStats tracks metrics for the user cache
type CacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
	Size      int
}

// cachedUserEntry wraps a user with version metadata for cache invalidation
type cachedUserEntry struct {
	Version  string       `json:"version"`
	User     *domain.User `json:"user"`
	CachedAt time.Time    `json:"cached_at"`
}

// userCache provides an in-memory LRU cache for user lookups
// with time-based expiration and version-based invalidation to prevent stale data.
type userCache struct {
	lru       *expirable.LRU[string, *cachedUserEntry]
	hits      atomic.Int64
	misses    atomic.Int64
	evictions atomic.Int64
}

// newUserCache creates a new user cache with the specified configuration.
func newUserCache(config CacheConfig) *userCache {
	c := &userCache{}
	// Use callback to track evictions if needed, currently lru/expirable doesn't expose explicit eviction callback easily
	// in the constructor for expirable, but we can track size.
	// expirable.LRU does typically have an EvictCallback in the underlying LRU but expirable wrapper simplifies it.
	// We will track hits/misses. Tracking evictions might need a wrapper or different library usage if strict.
	// For now, we will track what we can.

	onEvict := func(key string, value *cachedUserEntry) {
		c.evictions.Add(1)
	}

	c.lru = expirable.NewLRU[string, *cachedUserEntry](config.Size, onEvict, config.TTL)
	return c
}

// Get retrieves a user from the cache.
// Returns (user, true) if found and version matches.
// Returns (nil, false) if not in cache, expired, or version mismatch.
// Automatically invalidates entries with mismatched versions.
func (c *userCache) Get(platform, platformID string) (*domain.User, bool) {
	key := platform + ":" + platformID
	entry, found := c.lru.Get(key)
	if !found {
		c.misses.Add(1)
		return nil, false
	}

	// Check version - auto-invalidate if mismatch
	if entry.Version != CacheSchemaVersion {
		c.lru.Remove(key)
		c.misses.Add(1) // Treat version mismatch as a miss (and forced eviction)
		return nil, false
	}

	c.hits.Add(1)
	return entry.User, true
}

// Set stores a user in the cache with current schema version.
func (c *userCache) Set(platform, platformID string, user *domain.User) {
	key := platform + ":" + platformID
	entry := &cachedUserEntry{
		Version:  CacheSchemaVersion,
		User:     user,
		CachedAt: time.Now(),
	}
	c.lru.Add(key, entry)
}

// Invalidate removes a user from the cache.
// Useful when user data is updated.
func (c *userCache) Invalidate(platform, platformID string) {
	key := platform + ":" + platformID
	c.lru.Remove(key)
}

// Clear removes all entries from the cache.
func (c *userCache) Clear() {
	c.lru.Purge()
}

// GetStats returns the current cache statistics
func (c *userCache) GetStats() CacheStats {
	return CacheStats{
		Hits:      c.hits.Load(),
		Misses:    c.misses.Load(),
		Evictions: c.evictions.Load(),
		Size:      c.lru.Len(),
	}
}
