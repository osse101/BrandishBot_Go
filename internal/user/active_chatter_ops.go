package user

import (
	"fmt"
	"time"
)

// Track adds or updates a chatter's last message timestamp
func (t *ActiveChatterTracker) Track(platform, userID, username string) {
	key := makeKey(platform, userID)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.chatters[key] = &ChatterInfo{
		UserID:        userID,
		Username:      username,
		Platform:      platform,
		LastMessageAt: time.Now(),
	}
}

// Remove removes a chatter from the active list (e.g., when they've been hit)
func (t *ActiveChatterTracker) Remove(platform, userID string) {
	key := makeKey(platform, userID)

	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.chatters, key)
}

// cleanupLoop periodically removes expired chatters
func (t *ActiveChatterTracker) cleanupLoop() {
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
func (t *ActiveChatterTracker) cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	expiryThreshold := now.Add(-ChatterExpiryDuration)

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
func (t *ActiveChatterTracker) GetActiveCount(platform string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	expiryThreshold := now.Add(-ChatterExpiryDuration)
	count := 0

	for _, info := range t.chatters {
		if info.Platform == platform && info.LastMessageAt.After(expiryThreshold) {
			count++
		}
	}

	return count
}

// GetActiveChatters returns all currently active chatters across all platforms
func (t *ActiveChatterTracker) GetActiveChatters() []ChatterInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	expiryThreshold := now.Add(-ChatterExpiryDuration)

	var active []ChatterInfo
	for _, info := range t.chatters {
		if info.LastMessageAt.After(expiryThreshold) {
			active = append(active, *info)
		}
	}

	return active
}
