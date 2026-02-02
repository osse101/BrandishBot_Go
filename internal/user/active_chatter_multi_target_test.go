package user

import (
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestActiveChatterTracker_GetRandomTargets(t *testing.T) {
	tracker := NewActiveChatterTracker()
	defer tracker.Stop()

	// No users - should error
	_, err := tracker.GetRandomTargets(domain.PlatformDiscord, 5)
	if err == nil {
		t.Error("Expected error when no active chatters, got nil")
	}

	// Add 10 users
	for i := 1; i <= 10; i++ {
		tracker.Track(domain.PlatformDiscord, "user"+string(rune('0'+i)), "User"+string(rune('0'+i)))
	}

	// Request 5 targets
	targets, err := tracker.GetRandomTargets(domain.PlatformDiscord, 5)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(targets) != 5 {
		t.Errorf("Expected 5 targets, got %d", len(targets))
	}

	// Verify uniqueness (no duplicates)
	seen := make(map[string]bool)
	for _, target := range targets {
		if seen[target.UserID] {
			t.Errorf("Duplicate target found: %s", target.UserID)
		}
		seen[target.UserID] = true

		if target.Username == "" || target.UserID == "" {
			t.Error("Expected non-empty username and userID")
		}
	}

	// Request more than available (should return all available)
	targets, err = tracker.GetRandomTargets(domain.PlatformDiscord, 20)
	if err != nil {
		t.Fatalf("Unexpected error when requesting more than available: %v", err)
	}
	if len(targets) != 10 {
		t.Errorf("Expected 10 targets (all available), got %d", len(targets))
	}

	// Request exact count
	targets, err = tracker.GetRandomTargets(domain.PlatformDiscord, 10)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(targets) != 10 {
		t.Errorf("Expected 10 targets, got %d", len(targets))
	}

	// Request 1 target
	targets, err = tracker.GetRandomTargets(domain.PlatformDiscord, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(targets))
	}
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
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(targets) != 2 {
		t.Errorf("Expected 2 Discord targets, got %d", len(targets))
	}

	// Verify only Discord users
	for _, target := range targets {
		if target.Username != "Alice" && target.Username != "Bob" {
			t.Errorf("Got non-Discord user: %s", target.Username)
		}
	}

	// Request from Twitch (should only get Charlie and Dave)
	targets, err = tracker.GetRandomTargets(domain.PlatformTwitch, 5)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(targets) != 2 {
		t.Errorf("Expected 2 Twitch targets, got %d", len(targets))
	}

	// Verify only Twitch users
	for _, target := range targets {
		if target.Username != "Charlie" && target.Username != "Dave" {
			t.Errorf("Got non-Twitch user: %s", target.Username)
		}
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
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		for _, target := range targets {
			usersSeen[target.UserID]++
		}
	}

	// With 100 iterations selecting 5 from 20 users, we should see most users
	// Each user has 5/20 = 25% chance per iteration, so expected ~25 times
	// We should see at least 10 different users (conservative check)
	if len(usersSeen) < 10 {
		t.Errorf("Expected to see at least 10 different users over %d iterations, saw %d", iterations, len(usersSeen))
	}
}

func TestFormatTargetList(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "empty list",
			input:    []string{},
			expected: "",
		},
		{
			name:     "single user",
			input:    []string{"Alice"},
			expected: "Alice",
		},
		{
			name:     "two users",
			input:    []string{"Alice", "Bob"},
			expected: "Alice and Bob",
		},
		{
			name:     "three users",
			input:    []string{"Alice", "Bob", "Charlie"},
			expected: "Alice, Bob, and Charlie",
		},
		{
			name:     "five users",
			input:    []string{"Alice", "Bob", "Charlie", "Dave", "Eve"},
			expected: "Alice, Bob, Charlie, Dave, and Eve",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTargetList(tt.input)
			if result != tt.expected {
				t.Errorf("formatTargetList(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
