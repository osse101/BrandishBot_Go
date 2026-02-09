package user

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ActiveChatterTracker tracks users who have recently sent messages
// and are eligible to be targeted by random weapons like grenades and TNT.
type ActiveChatterTracker struct {
	mu       sync.RWMutex
	chatters map[string]*chatterInfo
	stopCh   chan struct{}
}

// chatterInfo holds information about an active chatter
type chatterInfo struct {
	UserID        string
	Username      string
	Platform      string
	LastMessageAt time.Time
}

const (
	// ChatterExpiryDuration is how long a user remains targetable after their last message
	ChatterExpiryDuration = 30 * time.Minute
	// CleanupInterval is how often we clean up expired chatters
	CleanupInterval = 5 * time.Minute
)

// NewActiveChatterTracker creates a new tracker and starts the cleanup goroutine
func NewActiveChatterTracker() *ActiveChatterTracker {
	tracker := &ActiveChatterTracker{
		chatters: make(map[string]*chatterInfo),
		stopCh:   make(chan struct{}),
	}
	go tracker.cleanupLoop()
	return tracker
}

// Track adds or updates a chatter's last message timestamp
func (t *ActiveChatterTracker) Track(platform, userID, username string) {
	key := makeKey(platform, userID)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.chatters[key] = &chatterInfo{
		UserID:        userID,
		Username:      username,
		Platform:      platform,
		LastMessageAt: time.Now(),
	}
}

// GetRandomTarget returns a random active chatter for the given platform
// Returns username and userID, or an error if no active chatters are available
func (t *ActiveChatterTracker) GetRandomTarget(platform string) (username string, userID string, err error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	expiryThreshold := now.Add(-ChatterExpiryDuration)

	// Collect all active (non-expired) chatters for this platform
	var activeChatters []*chatterInfo
	for _, info := range t.chatters {
		if info.Platform == platform && info.LastMessageAt.After(expiryThreshold) {
			activeChatters = append(activeChatters, info)
		}
	}

	if len(activeChatters) == 0 {
		return "", "", fmt.Errorf("no active targets available")
	}

	// Select a random chatter
	selected := activeChatters[rand.Intn(len(activeChatters))] //nolint:gosec // weak random is fine for games
	return selected.Username, selected.UserID, nil
}

// TargetInfo holds information about a selected target
type TargetInfo struct {
	Username string
	UserID   string
}

// GetRandomTargets returns multiple random active chatters for the given platform
// count specifies how many targets to select (will return fewer if not enough active chatters)
// Returns slice of TargetInfo or an error if no active chatters are available
func (t *ActiveChatterTracker) GetRandomTargets(platform string, count int) ([]TargetInfo, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	expiryThreshold := now.Add(-ChatterExpiryDuration)

	// Collect all active (non-expired) chatters for this platform
	var activeChatters []*chatterInfo
	for _, info := range t.chatters {
		if info.Platform == platform && info.LastMessageAt.After(expiryThreshold) {
			activeChatters = append(activeChatters, info)
		}
	}

	if len(activeChatters) == 0 {
		return nil, fmt.Errorf("no active targets available")
	}

	// Determine how many targets we can actually select
	numToSelect := count
	if numToSelect > len(activeChatters) {
		numToSelect = len(activeChatters)
	}

	// Shuffle and select first N (Fisher-Yates shuffle for first N elements)
	targets := make([]TargetInfo, numToSelect)
	selectedIndices := make([]int, len(activeChatters))
	for i := range selectedIndices {
		selectedIndices[i] = i
	}

	// Partial Fisher-Yates shuffle (only shuffle first numToSelect positions)
	for i := 0; i < numToSelect; i++ {
		j := i + rand.Intn(len(selectedIndices)-i) //nolint:gosec // weak random is fine for games
		selectedIndices[i], selectedIndices[j] = selectedIndices[j], selectedIndices[i]

		selectedChatter := activeChatters[selectedIndices[i]]
		targets[i] = TargetInfo{
			Username: selectedChatter.Username,
			UserID:   selectedChatter.UserID,
		}
	}

	return targets, nil
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

// Stop stops the cleanup goroutine (useful for testing and shutdown)
func (t *ActiveChatterTracker) Stop() {
	close(t.stopCh)
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
func (t *ActiveChatterTracker) GetActiveChatters() []chatterInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	expiryThreshold := now.Add(-ChatterExpiryDuration)

	var active []chatterInfo
	for _, info := range t.chatters {
		if info.LastMessageAt.After(expiryThreshold) {
			active = append(active, *info)
		}
	}

	return active
}
