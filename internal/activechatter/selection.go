package activechatter

import (
	"fmt"
	"math/rand"
	"time"
)

// GetRandomTarget returns a random active chatter for the given platform
// Returns username and userID, or an error if no active chatters are available
func (t *Tracker) GetRandomTarget(platform string) (username string, userID string, err error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	expiryThreshold := now.Add(-ExpiryDuration)

	// Collect all active (non-expired) chatters for this platform
	var activeChatters []*Chatter
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

// GetRandomTargets returns multiple random active chatters for the given platform
// count specifies how many targets to select (will return fewer if not enough active chatters)
// Returns slice of TargetInfo or an error if no active chatters are available
func (t *Tracker) GetRandomTargets(platform string, count int) ([]TargetInfo, error) {
	if count <= 0 {
		return []TargetInfo{}, nil
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	expiryThreshold := now.Add(-ExpiryDuration)

	// Collect all active (non-expired) chatters for this platform
	var activeChatters []*Chatter
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
