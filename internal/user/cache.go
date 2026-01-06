package user

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// CacheSchemaVersion is the current version of the cache schema
// Increment this when the cached data structure changes to auto-invalidate old entries
const CacheSchemaVersion = "1.0"

// cachedUserEntry wraps a user with version metadata for cache invalidation
type cachedUserEntry struct {
	Version string       `json:"version"`
	User    *domain.User `json:"user"`
	CachedAt time.Time   `json:"cached_at"`
}

// userCache provides an in-memory LRU cache for user lookups
// with time-based expiration and version-based invalidation to prevent stale data.
type userCache struct {
	lru *expirable.LRU[string, *cachedUserEntry]
}

// newUserCache creates a new user cache with the specified size and TTL.
// size: maximum number of cached users
// ttl: time-to-live for cached entries
func newUserCache(size int, ttl time.Duration) *userCache {
	return &userCache{
		lru: expirable.NewLRU[string, *cachedUserEntry](size, nil, ttl),
	}
}

// Get retrieves a user from the cache.
// Returns (user, true) if found and version matches.
// Returns (nil, false) if not in cache, expired, or version mismatch.
// Automatically invalidates entries with mismatched versions.
func (c *userCache) Get(platform, platformID string) (*domain.User, bool) {
	key := platform + ":" + platformID
	entry, found := c.lru.Get(key)
	if !found {
		return nil, false
	}
	
	// Check version - auto-invalidate if mismatch
	if entry.Version != CacheSchemaVersion {
		c.lru.Remove(key)
		return nil, false
	}
	
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
