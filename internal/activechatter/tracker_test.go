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
	t.Parallel()
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
	t.Parallel()

	tests := []struct {
		name          string
		setupTracker  func(*Tracker)
		platform      string
		expectError   bool
		expectedErr   string
		validateUsers func(t *testing.T, tracker *Tracker)
	}{
		{
			name: "Best Case - Valid Target Available",
			setupTracker: func(tr *Tracker) {
				tr.Track(domain.PlatformDiscord, "user1", "Alice")
			},
			platform:    domain.PlatformDiscord,
			expectError: false,
			validateUsers: func(t *testing.T, tr *Tracker) {
				username, userID, err := tr.GetRandomTarget(domain.PlatformDiscord)
				require.NoError(t, err)
				assert.Equal(t, "Alice", username)
				assert.Equal(t, "user1", userID)
			},
		},
		{
			name: "Invalid Case - No Users Yet",
			setupTracker: func(tr *Tracker) {
			},
			platform:    domain.PlatformDiscord,
			expectError: true,
			expectedErr: "no active targets available",
			validateUsers: func(t *testing.T, tr *Tracker) {
			},
		},
		{
			name: "Boundary Case - Multiple Users Randomness",
			setupTracker: func(tr *Tracker) {
				tr.Track(domain.PlatformDiscord, "user1", "Alice")
				tr.Track(domain.PlatformDiscord, "user2", "Bob")
				tr.Track(domain.PlatformDiscord, "user3", "Charlie")
			},
			platform:    domain.PlatformDiscord,
			expectError: false,
			validateUsers: func(t *testing.T, tr *Tracker) {
				seen := make(map[string]bool)
				for i := 0; i < 20; i++ {
					username, userID, err := tr.GetRandomTarget(domain.PlatformDiscord)
					require.NoError(t, err)
					assert.NotEmpty(t, username)
					assert.NotEmpty(t, userID)
					seen[username] = true
				}
				assert.GreaterOrEqual(t, len(seen), 2, "Expected to see at least 2 different users in 20 random selections")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tracker := NewTracker()
			defer tracker.Stop()

			tt.setupTracker(tracker)

			username, userID, err := tracker.GetRandomTarget(tt.platform)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, username)
				assert.NotEmpty(t, userID)
			}

			tt.validateUsers(t, tracker)
		})
	}
}

func TestTracker_Remove(t *testing.T) {
	t.Parallel()
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
	t.Parallel()

	tests := []struct {
		name          string
		setupTracker  func(*Tracker)
		targetPlat    string
		expectError   bool
		expectedErr   string
		expectedCount int
		expectedUser  string
	}{
		{
			name: "Best Case - Retrieve Discord User",
			setupTracker: func(tr *Tracker) {
				tr.Track(domain.PlatformDiscord, "user1", "Alice")
				tr.Track(domain.PlatformTwitch, "user2", "Bob")
				tr.Track(domain.PlatformYoutube, "user3", "Charlie")
			},
			targetPlat:    domain.PlatformDiscord,
			expectError:   false,
			expectedCount: 1,
			expectedUser:  "Alice",
		},
		{
			name: "Best Case - Retrieve Twitch User",
			setupTracker: func(tr *Tracker) {
				tr.Track(domain.PlatformDiscord, "user1", "Alice")
				tr.Track(domain.PlatformTwitch, "user2", "Bob")
				tr.Track(domain.PlatformYoutube, "user3", "Charlie")
			},
			targetPlat:    domain.PlatformTwitch,
			expectError:   false,
			expectedCount: 1,
			expectedUser:  "Bob",
		},
		{
			name: "Best Case - Retrieve Youtube User",
			setupTracker: func(tr *Tracker) {
				tr.Track(domain.PlatformDiscord, "user1", "Alice")
				tr.Track(domain.PlatformTwitch, "user2", "Bob")
				tr.Track(domain.PlatformYoutube, "user3", "Charlie")
			},
			targetPlat:    domain.PlatformYoutube,
			expectError:   false,
			expectedCount: 1,
			expectedUser:  "Charlie",
		},
		{
			name: "Invalid Case - Unknown Platform",
			setupTracker: func(tr *Tracker) {
				tr.Track(domain.PlatformDiscord, "user1", "Alice")
			},
			targetPlat:    "unknown",
			expectError:   true,
			expectedErr:   "no active targets available",
			expectedCount: 0,
		},
		{
			name: "Edge Case - Platform Empty While Others Have Users",
			setupTracker: func(tr *Tracker) {
				tr.Track(domain.PlatformTwitch, "user2", "Bob")
			},
			targetPlat:    domain.PlatformDiscord,
			expectError:   true,
			expectedErr:   "no active targets available",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tracker := NewTracker()
			defer tracker.Stop()

			tt.setupTracker(tracker)

			count := tracker.GetActiveCount(tt.targetPlat)
			assert.Equal(t, tt.expectedCount, count)

			user, _, err := tracker.GetRandomTarget(tt.targetPlat)
			if tt.expectError {
				require.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedUser, user)
			}
		})
	}
}

func TestTracker_Cleanup(t *testing.T) {
	t.Parallel()
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
	t.Parallel()

	tests := []struct {
		name          string
		setupTracker  func(*Tracker)
		platform      string
		requestCount  int
		expectError   bool
		expectedErr   string
		expectedCount int
	}{
		{
			name: "Best Case - Exact Available Count",
			setupTracker: func(tr *Tracker) {
				for i := 1; i <= 5; i++ {
					tr.Track(domain.PlatformDiscord, fmt.Sprintf("user%d", i), fmt.Sprintf("User%d", i))
				}
			},
			platform:      domain.PlatformDiscord,
			requestCount:  5,
			expectError:   false,
			expectedCount: 5,
		},
		{
			name: "Boundary Case - Request More Than Available",
			setupTracker: func(tr *Tracker) {
				for i := 1; i <= 10; i++ {
					tr.Track(domain.PlatformDiscord, fmt.Sprintf("user%d", i), fmt.Sprintf("User%d", i))
				}
			},
			platform:      domain.PlatformDiscord,
			requestCount:  20,
			expectError:   false,
			expectedCount: 10,
		},
		{
			name: "Boundary Case - Request Less Than Available",
			setupTracker: func(tr *Tracker) {
				for i := 1; i <= 10; i++ {
					tr.Track(domain.PlatformDiscord, fmt.Sprintf("user%d", i), fmt.Sprintf("User%d", i))
				}
			},
			platform:      domain.PlatformDiscord,
			requestCount:  5,
			expectError:   false,
			expectedCount: 5,
		},
		{
			name: "Invalid Case - No Users Available",
			setupTracker: func(tr *Tracker) {
			},
			platform:      domain.PlatformDiscord,
			requestCount:  5,
			expectError:   true,
			expectedErr:   "no active targets available",
			expectedCount: 0,
		},
		{
			name: "Edge Case - Request Zero Targets",
			setupTracker: func(tr *Tracker) {
				tr.Track(domain.PlatformDiscord, "user1", "Alice")
			},
			platform:      domain.PlatformDiscord,
			requestCount:  0,
			expectError:   false,
			expectedCount: 0,
		},
		{
			name: "Invalid Case - Request Negative Targets",
			setupTracker: func(tr *Tracker) {
				tr.Track(domain.PlatformDiscord, "user1", "Alice")
			},
			platform:      domain.PlatformDiscord,
			requestCount:  -1,
			expectError:   false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tracker := NewTracker()
			defer tracker.Stop()

			tt.setupTracker(tracker)

			targets, err := tracker.GetRandomTargets(tt.platform, tt.requestCount)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(targets))

				// Verify uniqueness
				seen := make(map[string]bool)
				for _, target := range targets {
					assert.False(t, seen[target.UserID], "Duplicate user found in results")
					seen[target.UserID] = true
				}
			}
		})
	}
}

func TestTracker_GetActiveChatters(t *testing.T) {
	t.Parallel()
	tracker := NewTracker()
	defer tracker.Stop()

	// Initially empty
	chatters := tracker.GetActiveChatters()
	assert.Empty(t, chatters)

	// Add chatters across platforms
	tracker.Track(domain.PlatformDiscord, "user1", "Alice")
	tracker.Track(domain.PlatformTwitch, "user2", "Bob")

	chatters = tracker.GetActiveChatters()
	assert.Equal(t, 2, len(chatters))

	// Verify the contents
	foundDiscord := false
	foundTwitch := false
	for _, c := range chatters {
		if c.Platform == domain.PlatformDiscord && c.Username == "Alice" && c.UserID == "user1" {
			foundDiscord = true
		}
		if c.Platform == domain.PlatformTwitch && c.Username == "Bob" && c.UserID == "user2" {
			foundTwitch = true
		}
	}
	assert.True(t, foundDiscord, "Expected to find Alice on Discord")
	assert.True(t, foundTwitch, "Expected to find Bob on Twitch")
}
