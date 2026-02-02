package user

import (
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestActiveChatterTracker_Track(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// Track a user
	tracker.Track(domain.PlatformDiscord, "user1", "Alice")

	// Verify the user is in the active list
	count := tracker.GetActiveCount(domain.PlatformDiscord)
	if count != 1 {
		t.Errorf("Expected 1 active chatter, got %d", count)
	}

	// Update the same user (should not create duplicate)
	tracker.Track(domain.PlatformDiscord, "user1", "Alice")
	count = tracker.GetActiveCount(domain.PlatformDiscord)
	if count != 1 {
		t.Errorf("Expected 1 active chatter after update, got %d", count)
	}

	// Track different user
	tracker.Track("discord", "user2", "Bob")
	count = tracker.GetActiveCount("discord")
	if count != 2 {
		t.Errorf("Expected 2 active chatters, got %d", count)
	}
}

func TestActiveChatterTracker_GetRandomTarget(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// No users yet
	_, _, err := tracker.GetRandomTarget(domain.PlatformDiscord)
	if err == nil {
		t.Error("Expected error when no active chatters, got nil")
	}

	// Add users
	tracker.Track(domain.PlatformDiscord, "user1", "Alice")
	tracker.Track(domain.PlatformDiscord, "user2", "Bob")
	tracker.Track(domain.PlatformDiscord, "user3", "Charlie")

	// Get random target (multiple times to check randomness)
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		username, userID, err := tracker.GetRandomTarget(domain.PlatformDiscord)
		if err != nil {
			t.Fatalf("Unexpected error getting random target: %v", err)
		}
		if username == "" || userID == "" {
			t.Error("Expected non-empty username and userID")
		}
		seen[username] = true
	}

	// With 20 attempts and 3 users, we should see at least 2 different users (very high probability)
	if len(seen) < 2 {
		t.Errorf("Expected to see at least 2 different users in 20 random selections, saw %d", len(seen))
	}
}

func TestActiveChatterTracker_Remove(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// Add users
	tracker.Track(domain.PlatformDiscord, "user1", "Alice")
	tracker.Track(domain.PlatformDiscord, "user2", "Bob")

	count := tracker.GetActiveCount(domain.PlatformDiscord)
	if count != 2 {
		t.Errorf("Expected 2 active chatters, got %d", count)
	}

	// Remove one user
	tracker.Remove(domain.PlatformDiscord, "user1")
	count = tracker.GetActiveCount(domain.PlatformDiscord)
	if count != 1 {
		t.Errorf("Expected 1 active chatter after removal, got %d", count)
	}

	// Verify we can still get the remaining user
	username, userID, err := tracker.GetRandomTarget(domain.PlatformDiscord)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if username != "Bob" || userID != "user2" {
		t.Errorf("Expected Bob/user2, got %s/%s", username, userID)
	}

	// Remove the last user
	tracker.Remove(domain.PlatformDiscord, "user2")
	count = tracker.GetActiveCount(domain.PlatformDiscord)
	if count != 0 {
		t.Errorf("Expected 0 active chatters, got %d", count)
	}

	// No users left
	_, _, err = tracker.GetRandomTarget(domain.PlatformDiscord)
	if err == nil {
		t.Error("Expected error when no active chatters remain, got nil")
	}
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

	if discordCount != 1 || twitchCount != 1 || youtubeCount != 1 {
		t.Errorf("Expected 1 user per platform, got discord=%d, twitch=%d, youtube=%d",
			discordCount, twitchCount, youtubeCount)
	}

	// Get random target from each platform
	discordUser, _, err := tracker.GetRandomTarget(domain.PlatformDiscord)
	if err != nil || discordUser != "Alice" {
		t.Errorf("Expected Alice from discord, got %s (err: %v)", discordUser, err)
	}

	twitchUser, _, err := tracker.GetRandomTarget(domain.PlatformTwitch)
	if err != nil || twitchUser != "Bob" {
		t.Errorf("Expected Bob from twitch, got %s (err: %v)", twitchUser, err)
	}

	youtubeUser, _, err := tracker.GetRandomTarget(domain.PlatformYoutube)
	if err != nil || youtubeUser != "Charlie" {
		t.Errorf("Expected Charlie from youtube, got %s (err: %v)", youtubeUser, err)
	}

	// Empty platform
	_, _, err = tracker.GetRandomTarget("unknown")
	if err == nil {
		t.Error("Expected error for unknown platform, got nil")
	}
}

func TestActiveChatterTracker_ExpiryFiltering(t *testing.T) {
	// Test with very short expiry for faster testing
	// Note: This test directly manipulates the internal state for testing purposes
	tracker := &ActiveChatterTracker{
		chatters: make(map[string]*chatterInfo),
		stopCh:   make(chan struct{}),
	}
	defer tracker.Stop()

	// Add a user with old timestamp
	oldTime := time.Now().Add(-31 * time.Minute)
	tracker.chatters[makeKey(domain.PlatformDiscord, "user1")] = &chatterInfo{
		UserID:        "user1",
		Username:      "Alice",
		Platform:      domain.PlatformDiscord,
		LastMessageAt: oldTime,
	}

	// Add a user with recent timestamp
	tracker.Track(domain.PlatformDiscord, "user2", "Bob")

	// GetRandomTarget should only return the recent user
	username, userID, err := tracker.GetRandomTarget(domain.PlatformDiscord)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if username != "Bob" || userID != "user2" {
		t.Errorf("Expected Bob/user2 (fresh entry), got %s/%s", username, userID)
	}

	// GetActiveCount should only count the recent user
	count := tracker.GetActiveCount(domain.PlatformDiscord)
	if count != 1 {
		t.Errorf("Expected 1 active chatter (excluding expired), got %d", count)
	}
}

func TestActiveChatterTracker_Cleanup(t *testing.T) {
	tracker := &ActiveChatterTracker{
		chatters: make(map[string]*chatterInfo),
		stopCh:   make(chan struct{}),
	}
	defer tracker.Stop()

	// Add expired entries
	oldTime := time.Now().Add(-31 * time.Minute)
	tracker.chatters[makeKey(domain.PlatformDiscord, "user1")] = &chatterInfo{
		UserID:        "user1",
		Username:      "Alice",
		Platform:      domain.PlatformDiscord,
		LastMessageAt: oldTime,
	}
	tracker.chatters[makeKey(domain.PlatformDiscord, "user2")] = &chatterInfo{
		UserID:        "user2",
		Username:      "Bob",
		Platform:      domain.PlatformDiscord,
		LastMessageAt: oldTime,
	}

	// Add fresh entry
	tracker.Track(domain.PlatformDiscord, "user3", "Charlie")

	// Total entries before cleanup
	if len(tracker.chatters) != 3 {
		t.Errorf("Expected 3 total entries, got %d", len(tracker.chatters))
	}

	// Run cleanup
	tracker.cleanup()

	// Should have removed the 2 expired entries
	if len(tracker.chatters) != 1 {
		t.Errorf("Expected 1 entry after cleanup, got %d", len(tracker.chatters))
	}

	// Verify the remaining entry is Charlie
	username, userID, err := tracker.GetRandomTarget(domain.PlatformDiscord)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if username != "Charlie" || userID != "user3" {
		t.Errorf("Expected Charlie/user3, got %s/%s", username, userID)
	}
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
