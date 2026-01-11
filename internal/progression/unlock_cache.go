package progression

import (
	"fmt"
	"sync"
)

// UnlockCache provides in-memory caching for node unlock status checks
// This dramatically reduces database queries for feature unlock checks which are
// the hottest queries in the system (every !search, !gamble, !craft, etc.)
type UnlockCache struct {
	mu      sync.RWMutex
	unlocks map[string]bool // "nodeKey:level" -> unlocked status
}

// NewUnlockCache creates a new unlock status cache
func NewUnlockCache() *UnlockCache {
	return &UnlockCache{
		unlocks: make(map[string]bool),
	}
}

// Get retrieves cached unlock status for a node at a specific level
// Returns (unlocked, found). If found is false, caller should query DB.
func (c *UnlockCache) Get(nodeKey string, level int) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", nodeKey, level)
	unlocked, found := c.unlocks[key]
	return unlocked, found
}

// Set stores unlock status for a node at a specific level
func (c *UnlockCache) Set(nodeKey string, level int, unlocked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s:%d", nodeKey, level)
	c.unlocks[key] = unlocked
}

// InvalidateAll clears the entire cache
// Called when any node is unlocked or relocked to ensure consistency
func (c *UnlockCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.unlocks = make(map[string]bool)
}

// Size returns the current number of cached entries
func (c *UnlockCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.unlocks)
}
