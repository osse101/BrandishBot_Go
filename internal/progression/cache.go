package progression

import (
	"sync"
	"time"
)

// ModifierCache provides in-memory caching for calculated modifier values
type ModifierCache struct {
	mu     sync.RWMutex
	values map[string]*CachedModifier
	ttl    time.Duration
}

// CachedModifier represents a cached final value with metadata
type CachedModifier struct {
	Value     float64
	NodeLevel int
	CachedAt  time.Time
	ExpiresAt time.Time
}

// NewModifierCache creates a new cache with the specified TTL
func NewModifierCache(ttl time.Duration) *ModifierCache {
	return &ModifierCache{
		values: make(map[string]*CachedModifier),
		ttl:    ttl,
	}
}

// Get retrieves a cached value if it exists and hasn't expired
func (c *ModifierCache) Get(featureKey string) (*CachedModifier, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.values[featureKey]
	if !ok || time.Now().After(cached.ExpiresAt) {
		return nil, false
	}
	return cached, true
}

// Set stores a calculated value in the cache
func (c *ModifierCache) Set(featureKey string, value float64, nodeLevel int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.values[featureKey] = &CachedModifier{
		Value:     value,
		NodeLevel: nodeLevel,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}
}

// Invalidate removes a specific feature key from the cache
func (c *ModifierCache) Invalidate(featureKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.values, featureKey)
}

// InvalidateAll clears the entire cache
// Called when any progression node is unlocked/leveled
func (c *ModifierCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values = make(map[string]*CachedModifier)
}

// Size returns the current number of cached entries
func (c *ModifierCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.values)
}
