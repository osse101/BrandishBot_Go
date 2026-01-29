package cooldown

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHashUserAction(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		action string
	}{
		{"normal", "user123", "search"},
		{"empty", "", ""},
		{"long", "user-uuid-long-string", "action-name-very-long"},
		{"symbols", "user!@#", "action$%^"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h1 := hashUserAction(tt.userID, tt.action)
			h2 := hashUserAction(tt.userID, tt.action)

			// Determinism
			assert.Equal(t, h1, h2, "hash should be deterministic")

			// Positive value (MSB masked)
			assert.GreaterOrEqual(t, h1, int64(0), "hash should be positive")
		})
	}

	t.Run("collisions", func(t *testing.T) {
		h1 := hashUserAction("user1", "search")
		h2 := hashUserAction("user1", "explore")
		assert.NotEqual(t, h1, h2, "different actions should have different hashes")

		h3 := hashUserAction("user2", "search")
		assert.NotEqual(t, h1, h3, "different users should have different hashes")
	})
}

func TestCheckCooldownInternal(t *testing.T) {
	// Create a dummy backend just to call the method, though it doesn't use receiver fields
	b := &postgresBackend{}

	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	duration := 5 * time.Minute

	tests := []struct {
		name          string
		lastUsed      *time.Time
		wantOnCooldown bool
		wantRemaining  time.Duration
	}{
		{
			name:          "nil lastUsed",
			lastUsed:      nil,
			wantOnCooldown: false,
			wantRemaining:  0,
		},
		{
			name:          "active cooldown",
			lastUsed:      ptr(now.Add(-2 * time.Minute)), // 2 mins ago
			wantOnCooldown: true,
			wantRemaining:  3 * time.Minute, // 5 - 2 = 3
		},
		{
			name:          "expired cooldown",
			lastUsed:      ptr(now.Add(-6 * time.Minute)), // 6 mins ago
			wantOnCooldown: false,
			wantRemaining:  0,
		},
		{
			name:          "exact boundary",
			lastUsed:      ptr(now.Add(-5 * time.Minute)), // 5 mins ago
			wantOnCooldown: false,
			wantRemaining:  0,
		},
		{
			name:          "just before expiry",
			lastUsed:      ptr(now.Add(-5 * time.Minute + 1*time.Second)), // 4m 59s ago
			wantOnCooldown: true,
			wantRemaining:  1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOnCooldown, gotRemaining := b.checkCooldownInternal(now, tt.lastUsed, duration)
			assert.Equal(t, tt.wantOnCooldown, gotOnCooldown)
			assert.Equal(t, tt.wantRemaining, gotRemaining)
		})
	}
}

func ptr(t time.Time) *time.Time {
	return &t
}
