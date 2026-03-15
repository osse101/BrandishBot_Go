package activechatter

import (
	"fmt"
	"sync"
	"time"
)

// Tracker tracks users who have recently sent messages
// and are eligible to be targeted by random weapons like grenades and TNT.
type Tracker struct {
	mu       sync.RWMutex
	chatters map[string]*Chatter
	stopCh   chan struct{}
}

// NewTracker creates a new tracker and starts the cleanup goroutine
func NewTracker() *Tracker {
	tracker := &Tracker{
		chatters: make(map[string]*Chatter),
		stopCh:   make(chan struct{}),
	}
	go tracker.cleanupLoop()
	return tracker
}

// Stop stops the cleanup goroutine (useful for testing and shutdown)
func (t *Tracker) Stop() {
	close(t.stopCh)
}

// Track adds or updates a chatter's last message timestamp
func (t *Tracker) Track(platform, userID, username string) {
	key := makeKey(platform, userID)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.chatters[key] = &Chatter{
		UserID:        userID,
		Username:      username,
		Platform:      platform,
		LastMessageAt: time.Now(),
	}
}

// Remove removes a chatter from the active list (e.g., when they've been hit)
func (t *Tracker) Remove(platform, userID string) {
	key := makeKey(platform, userID)

	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.chatters, key)
}

// cleanupLoop periodically removes expired chatters
func (t *Tracker) cleanupLoop() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.cleanup()
		case <-t.stopCh:
			return
		}
	}
}

// cleanup removes all expired chatters
func (t *Tracker) cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	expiryThreshold := now.Add(-ExpiryDuration)

	for key, info := range t.chatters {
		if info.LastMessageAt.Before(expiryThreshold) {
			delete(t.chatters, key)
		}
	}
}

// makeKey creates a composite key for the chatters map
func makeKey(platform, userID string) string {
	return fmt.Sprintf("%s:%s", platform, userID)
}

// GetActiveCount returns the number of active chatters for a platform (for testing/debugging)
func (t *Tracker) GetActiveCount(platform string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	expiryThreshold := now.Add(-ExpiryDuration)
	count := 0

	for _, info := range t.chatters {
		if info.Platform == platform && info.LastMessageAt.After(expiryThreshold) {
			count++
		}
	}

	return count
}

// GetActiveChatters returns all currently active chatters across all platforms
func (t *Tracker) GetActiveChatters() []Chatter {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	expiryThreshold := now.Add(-ExpiryDuration)

	var active []Chatter
	for _, info := range t.chatters {
		if info.LastMessageAt.After(expiryThreshold) {
			active = append(active, *info)
		}
	}

	return active
}
