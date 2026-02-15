package subscription

import (
	"fmt"
	"sync"
	"time"
)

// StatusCache provides in-memory caching for subscription status lookups
type StatusCache struct {
	mu     sync.RWMutex
	values map[string]*CachedStatus
	ttl    time.Duration
}

// CachedStatus represents a cached subscription status
type CachedStatus struct {
	IsActive  bool
	TierName  string
	TierLevel int
	CachedAt  time.Time
	ExpiresAt time.Time
}

// NewStatusCache creates a new cache with the specified TTL
func NewStatusCache(ttl time.Duration) *StatusCache {
	return &StatusCache{
		values: make(map[string]*CachedStatus),
		ttl:    ttl,
	}
}

// Get retrieves a cached subscription status if it exists and hasn't expired
func (c *StatusCache) Get(userID, platform string) (*CachedStatus, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := makeKey(userID, platform)
	cached, ok := c.values[key]
	if !ok || time.Now().After(cached.ExpiresAt) {
		return nil, false
	}
	return cached, true
}

// Set stores a subscription status in the cache
func (c *StatusCache) Set(userID, platform string, isActive bool, tierName string, tierLevel int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	key := makeKey(userID, platform)
	c.values[key] = &CachedStatus{
		IsActive:  isActive,
		TierName:  tierName,
		TierLevel: tierLevel,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}
}

// Invalidate removes a specific user's subscription status from cache
func (c *StatusCache) Invalidate(userID, platform string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := makeKey(userID, platform)
	delete(c.values, key)
}

// InvalidateAll clears the entire cache
func (c *StatusCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values = make(map[string]*CachedStatus)
}

// Size returns the current number of cached entries
func (c *StatusCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.values)
}

// makeKey creates a cache key from userID and platform
func makeKey(userID, platform string) string {
	return fmt.Sprintf("%s:%s", platform, userID)
}
