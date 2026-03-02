package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestActiveChatterTracker_GetRandomTargets(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// No users - should error
	_, err := tracker.GetRandomTargets(domain.PlatformDiscord, 5)
	require.Error(t, err, "Expected error when no active chatters")

	// Add 10 users
	for i := 1; i <= 10; i++ {
		tracker.Track(domain.PlatformDiscord, "user"+string(rune('0'+i)), "User"+string(rune('0'+i)))
	}

	// Request 5 targets
	targets, err := tracker.GetRandomTargets(domain.PlatformDiscord, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, len(targets))

	// Verify uniqueness (no duplicates)
	seen := make(map[string]bool)
	for _, target := range targets {
		assert.False(t, seen[target.UserID], "Duplicate target found: %s", target.UserID)
		seen[target.UserID] = true

		assert.NotEmpty(t, target.Username, "Expected non-empty username")
		assert.NotEmpty(t, target.UserID, "Expected non-empty userID")
	}

	// Request more than available (should return all available)
	targets, err = tracker.GetRandomTargets(domain.PlatformDiscord, 20)
	require.NoError(t, err)
	assert.Equal(t, 10, len(targets), "Expected 10 targets (all available)")

	// Request exact count
	targets, err = tracker.GetRandomTargets(domain.PlatformDiscord, 10)
	require.NoError(t, err)
	assert.Equal(t, 10, len(targets))

	// Request 1 target
	targets, err = tracker.GetRandomTargets(domain.PlatformDiscord, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, len(targets))
}

func TestActiveChatterTracker_GetRandomTargets_PlatformIsolation(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// Add users to different platforms
	tracker.Track(domain.PlatformDiscord, "user1", "Alice")
	tracker.Track(domain.PlatformDiscord, "user2", "Bob")
	tracker.Track(domain.PlatformTwitch, "user3", "Charlie")
	tracker.Track(domain.PlatformTwitch, "user4", "Dave")

	// Request from Discord (should only get Alice and Bob)
	targets, err := tracker.GetRandomTargets(domain.PlatformDiscord, 5)
	require.NoError(t, err)
	assert.Equal(t, 2, len(targets))

	// Verify only Discord users
	for _, target := range targets {
		assert.True(t, target.Username == "Alice" || target.Username == "Bob", "Got non-Discord user: %s", target.Username)
	}

	// Request from Twitch (should only get Charlie and Dave)
	targets, err = tracker.GetRandomTargets(domain.PlatformTwitch, 5)
	require.NoError(t, err)
	assert.Equal(t, 2, len(targets))

	// Verify only Twitch users
	for _, target := range targets {
		assert.True(t, target.Username == "Charlie" || target.Username == "Dave", "Got non-Twitch user: %s", target.Username)
	}
}

func TestActiveChatterTracker_GetRandomTargets_Randomness(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// Add 20 users
	for i := 1; i <= 20; i++ {
		username := "User" + string(rune('A'-1+i))
		userID := "user" + string(rune('0'+i))
		tracker.Track(domain.PlatformDiscord, userID, username)
	}

	// Request 5 targets multiple times and track which users we see
	usersSeen := make(map[string]int)
	iterations := 100

	for i := 0; i < iterations; i++ {
		targets, err := tracker.GetRandomTargets(domain.PlatformDiscord, 5)
		require.NoError(t, err)
		for _, target := range targets {
			usersSeen[target.UserID]++
		}
	}

	// Probabilistic check: verify at least 10 different users selected over 100 iterations.
	assert.GreaterOrEqual(t, len(usersSeen), 10, "Expected to see at least 10 different users over %d iterations, saw %d", iterations, len(usersSeen))
}
