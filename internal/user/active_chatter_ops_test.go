package user

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestActiveChatterTracker_Track(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// Track a user
	tracker.Track(domain.PlatformDiscord, "user1", "Alice")

	// Verify the user is in the active list
	count := tracker.GetActiveCount(domain.PlatformDiscord)
	assert.Equal(t, 1, count, "Expected 1 active chatter")

	// Update the same user (should not create duplicate)
	tracker.Track(domain.PlatformDiscord, "user1", "Alice")
	count = tracker.GetActiveCount(domain.PlatformDiscord)
	assert.Equal(t, 1, count, "Expected 1 active chatter after update")

	// Track different user
	tracker.Track("discord", "user2", "Bob")
	count = tracker.GetActiveCount("discord")
	assert.Equal(t, 2, count, "Expected 2 active chatters")
}

func TestActiveChatterTracker_GetRandomTarget(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// No users yet
	_, _, err := tracker.GetRandomTarget(domain.PlatformDiscord)
	require.Error(t, err, "Expected error when no active chatters")

	// Add users
	tracker.Track(domain.PlatformDiscord, "user1", "Alice")
	tracker.Track(domain.PlatformDiscord, "user2", "Bob")
	tracker.Track(domain.PlatformDiscord, "user3", "Charlie")

	// Get random target (multiple times to check randomness)
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		username, userID, err := tracker.GetRandomTarget(domain.PlatformDiscord)
		require.NoError(t, err, "Unexpected error getting random target")
		assert.NotEmpty(t, username, "Expected non-empty username")
		assert.NotEmpty(t, userID, "Expected non-empty userID")
		seen[username] = true
	}

	// With 20 attempts and 3 users, we should see at least 2 different users (very high probability)
	assert.GreaterOrEqual(t, len(seen), 2, "Expected to see at least 2 different users in 20 random selections")
}

func TestActiveChatterTracker_Remove(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// Add users
	tracker.Track(domain.PlatformDiscord, "user1", "Alice")
	tracker.Track(domain.PlatformDiscord, "user2", "Bob")

	count := tracker.GetActiveCount(domain.PlatformDiscord)
	assert.Equal(t, 2, count, "Expected 2 active chatters")

	// Remove one user
	tracker.Remove(domain.PlatformDiscord, "user1")
	count = tracker.GetActiveCount(domain.PlatformDiscord)
	assert.Equal(t, 1, count, "Expected 1 active chatter after removal")

	// Verify we can still get the remaining user
	username, userID, err := tracker.GetRandomTarget(domain.PlatformDiscord)
	require.NoError(t, err, "Unexpected error getting remaining user")
	assert.Equal(t, "Bob", username)
	assert.Equal(t, "user2", userID)

	// Remove the last user
	tracker.Remove(domain.PlatformDiscord, "user2")
	count = tracker.GetActiveCount(domain.PlatformDiscord)
	assert.Equal(t, 0, count, "Expected 0 active chatters")

	// No users left
	_, _, err = tracker.GetRandomTarget(domain.PlatformDiscord)
	require.Error(t, err, "Expected error when no active chatters remain")
}

func TestActiveChatterTracker_PlatformIsolation(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// Add users to different platforms
	tracker.Track(domain.PlatformDiscord, "user1", "Alice")
	tracker.Track(domain.PlatformTwitch, "user2", "Bob")
	tracker.Track(domain.PlatformYoutube, "user3", "Charlie")

	// Verify platform isolation
	discordCount := tracker.GetActiveCount(domain.PlatformDiscord)
	twitchCount := tracker.GetActiveCount(domain.PlatformTwitch)
	youtubeCount := tracker.GetActiveCount(domain.PlatformYoutube)

	assert.Equal(t, 1, discordCount)
	assert.Equal(t, 1, twitchCount)
	assert.Equal(t, 1, youtubeCount)

	// Get random target from each platform
	discordUser, _, err := tracker.GetRandomTarget(domain.PlatformDiscord)
	require.NoError(t, err)
	assert.Equal(t, "Alice", discordUser)

	twitchUser, _, err := tracker.GetRandomTarget(domain.PlatformTwitch)
	require.NoError(t, err)
	assert.Equal(t, "Bob", twitchUser)

	youtubeUser, _, err := tracker.GetRandomTarget(domain.PlatformYoutube)
	require.NoError(t, err)
	assert.Equal(t, "Charlie", youtubeUser)

	// Empty platform
	_, _, err = tracker.GetRandomTarget("unknown")
	require.Error(t, err, "Expected error for unknown platform")
}

func TestActiveChatterTracker_ExpiryFiltering(t *testing.T) {
	// Test with very short expiry for faster testing
	// Note: This test directly manipulates the internal state for testing purposes
	tracker := &ActiveChatterTracker{
		chatters: make(map[string]*ChatterInfo),
		stopCh:   make(chan struct{}),
	}
	defer tracker.Stop()

	// Add a user with old timestamp
	oldTime := time.Now().Add(-31 * time.Minute)
	tracker.chatters[makeKey(domain.PlatformDiscord, "user1")] = &ChatterInfo{
		UserID:        "user1",
		Username:      "Alice",
		Platform:      domain.PlatformDiscord,
		LastMessageAt: oldTime,
	}

	// Add a user with recent timestamp
	tracker.Track(domain.PlatformDiscord, "user2", "Bob")

	// GetRandomTarget should only return the recent user
	username, userID, err := tracker.GetRandomTarget(domain.PlatformDiscord)
	require.NoError(t, err)
	assert.Equal(t, "Bob", username)
	assert.Equal(t, "user2", userID)

	// GetActiveCount should only count the recent user
	count := tracker.GetActiveCount(domain.PlatformDiscord)
	assert.Equal(t, 1, count, "Expected 1 active chatter (excluding expired)")
}

func TestActiveChatterTracker_Cleanup(t *testing.T) {
	tracker := &ActiveChatterTracker{
		chatters: make(map[string]*ChatterInfo),
		stopCh:   make(chan struct{}),
	}
	defer tracker.Stop()

	// Add expired entries
	oldTime := time.Now().Add(-31 * time.Minute)
	tracker.chatters[makeKey(domain.PlatformDiscord, "user1")] = &ChatterInfo{
		UserID:        "user1",
		Username:      "Alice",
		Platform:      domain.PlatformDiscord,
		LastMessageAt: oldTime,
	}
	tracker.chatters[makeKey(domain.PlatformDiscord, "user2")] = &ChatterInfo{
		UserID:        "user2",
		Username:      "Bob",
		Platform:      domain.PlatformDiscord,
		LastMessageAt: oldTime,
	}

	// Add fresh entry
	tracker.Track(domain.PlatformDiscord, "user3", "Charlie")

	// Total entries before cleanup
	assert.Equal(t, 3, len(tracker.chatters), "Expected 3 total entries")

	// Run cleanup
	tracker.cleanup()

	// Should have removed the 2 expired entries
	assert.Equal(t, 1, len(tracker.chatters), "Expected 1 entry after cleanup")

	// Verify the remaining entry is Charlie
	username, userID, err := tracker.GetRandomTarget(domain.PlatformDiscord)
	require.NoError(t, err)
	assert.Equal(t, "Charlie", username)
	assert.Equal(t, "user3", userID)
}

func TestActiveChatterTracker_Concurrency(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// Run concurrent operations
	done := make(chan bool)
	numGoroutines := 10
	operationsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func(_ int) {
			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix of operations
				switch j % 4 {
				case 0:
					tracker.Track(domain.PlatformDiscord, "user1", "Alice")
				case 1:
					tracker.GetRandomTarget(domain.PlatformDiscord)
				case 2:
					tracker.Remove(domain.PlatformDiscord, "user1")
				case 3:
					tracker.GetActiveCount(domain.PlatformDiscord)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// If we get here without panicking or deadlocking, the test passes
	t.Log("Concurrency test completed successfully")
}
