package activechatter

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestTracker_Track(t *testing.T) {
	tracker := NewTracker()
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
	tracker.Track(domain.PlatformDiscord, "user2", "Bob")
	count = tracker.GetActiveCount(domain.PlatformDiscord)
	assert.Equal(t, 2, count, "Expected 2 active chatters")
}

func TestTracker_GetRandomTarget(t *testing.T) {
	tracker := NewTracker()
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

func TestTracker_Remove(t *testing.T) {
	tracker := NewTracker()
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

func TestTracker_PlatformIsolation(t *testing.T) {
	tracker := NewTracker()
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

func TestTracker_Cleanup(t *testing.T) {
	tracker := &Tracker{
		chatters: make(map[string]*Chatter),
		stopCh:   make(chan struct{}),
	}
	defer tracker.Stop()

	// Add expired entries
	oldTime := time.Now().Add(-31 * time.Minute)
	tracker.chatters[makeKey(domain.PlatformDiscord, "user1")] = &Chatter{
		UserID:        "user1",
		Username:      "Alice",
		Platform:      domain.PlatformDiscord,
		LastMessageAt: oldTime,
	}
	tracker.chatters[makeKey(domain.PlatformDiscord, "user2")] = &Chatter{
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

func TestTracker_GetRandomTargets(t *testing.T) {
	tracker := NewTracker()
	defer tracker.Stop()

	// No users - should error
	_, err := tracker.GetRandomTargets(domain.PlatformDiscord, 5)
	require.Error(t, err)

	// Add 10 users
	for i := 1; i <= 10; i++ {
		tracker.Track(domain.PlatformDiscord, fmt.Sprintf("user%d", i), fmt.Sprintf("User%d", i))
	}

	// Request 5 targets
	targets, err := tracker.GetRandomTargets(domain.PlatformDiscord, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, len(targets))

	// Verify uniqueness
	seen := make(map[string]bool)
	for _, target := range targets {
		assert.False(t, seen[target.UserID])
		seen[target.UserID] = true
	}

	// Request more than available
	targets, err = tracker.GetRandomTargets(domain.PlatformDiscord, 20)
	require.NoError(t, err)
	assert.Equal(t, 10, len(targets))
}
