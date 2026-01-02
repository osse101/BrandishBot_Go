package user

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// userCache provides an in-memory LRU cache for user lookups
// with time-based expiration to prevent stale data.
type userCache struct {
	lru *expirable.LRU[string, *domain.User]
}

// newUserCache creates a new user cache with the specified size and TTL.
// size: maximum number of cached users
// ttl: time-to-live for cached entries
func newUserCache(size int, ttl time.Duration) *userCache {
	return &userCache{
		lru: expirable.NewLRU[string, *domain.User](size, nil, ttl),
	}
}

// Get retrieves a user from the cache.
// Returns (user, true) if found, (nil, false) if not in cache or expired.
func (c *userCache) Get(platform, platformID string) (*domain.User, bool) {
	key := platform + ":" + platformID
	return c.lru.Get(key)
}

// Set stores a user in the cache.
func (c *userCache) Set(platform, platformID string, user *domain.User) {
	key := platform + ":" + platformID
	c.lru.Add(key, user)
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
