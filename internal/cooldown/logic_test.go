package cooldown

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestConfig_GetCooldownDuration(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		action string
		want   time.Duration
	}{
		{
			name: "custom override",
			config: Config{
				Cooldowns: map[string]time.Duration{
					"custom_action": 10 * time.Minute,
				},
			},
			action: "custom_action",
			want:   10 * time.Minute,
		},
		{
			name: "domain default - search",
			config: Config{
				Cooldowns: nil,
			},
			action: domain.ActionSearch,
			want:   domain.SearchCooldownDuration,
		},
		{
			name: "unknown action default",
			config: Config{
				Cooldowns: nil,
			},
			action: "unknown_action",
			want:   DefaultCooldownDuration,
		},
		{
			name: "override search",
			config: Config{
				Cooldowns: map[string]time.Duration{
					domain.ActionSearch: 1 * time.Minute,
				},
			},
			action: domain.ActionSearch,
			want:   1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetCooldownDuration(tt.action)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Mock for testing
type mockProgressionService struct {
	mockGetModifiedValue func(ctx context.Context, featureKey string, baseValue float64) (float64, error)
}

func (m *mockProgressionService) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	if m.mockGetModifiedValue != nil {
		return m.mockGetModifiedValue(ctx, featureKey, baseValue)
	}
	return baseValue, nil
}

func TestGetEffectiveCooldown(t *testing.T) {
	baseDuration := 5 * time.Minute
	config := Config{
		Cooldowns: map[string]time.Duration{
			domain.ActionSearch: baseDuration,
			"other":             baseDuration,
		},
	}

	tests := []struct {
		name      string
		action    string
		mockSetup func() *mockProgressionService
		want      time.Duration
	}{
		{
			name:      "nil progression service",
			action:    domain.ActionSearch,
			mockSetup: func() *mockProgressionService { return nil },
			want:      baseDuration,
		},
		{
			name:   "success modified value",
			action: domain.ActionSearch,
			mockSetup: func() *mockProgressionService {
				return &mockProgressionService{
					mockGetModifiedValue: func(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
						assert.Equal(t, FeatureKeySearchCooldownReduction, featureKey)
						assert.Equal(t, float64(baseDuration), baseValue)
						return float64(4 * time.Minute), nil
					},
				}
			},
			want: 4 * time.Minute,
		},
		{
			name:   "progression service error",
			action: domain.ActionSearch,
			mockSetup: func() *mockProgressionService {
				return &mockProgressionService{
					mockGetModifiedValue: func(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
						return 0, errors.New("progression error")
					},
				}
			},
			want: baseDuration,
		},
		{
			name:   "not search action",
			action: "other",
			mockSetup: func() *mockProgressionService {
				return &mockProgressionService{
					mockGetModifiedValue: func(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
						t.Fatal("should not be called")
						return 0, nil
					},
				}
			},
			want: baseDuration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var progSvc ProgressionService
			mock := tt.mockSetup()
			if mock != nil {
				progSvc = mock
			}

			b := &postgresBackend{
				config:         config,
				progressionSvc: progSvc,
			}

			got := b.getEffectiveCooldown(context.Background(), tt.action)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
	defaultDuration := 5 * time.Minute

	tests := []struct {
		name           string
		lastUsed       *time.Time
		duration       time.Duration
		wantOnCooldown bool
		wantRemaining  time.Duration
	}{
		{
			name:           "nil lastUsed",
			lastUsed:       nil,
			duration:       defaultDuration,
			wantOnCooldown: false,
			wantRemaining:  0,
		},
		{
			name:           "active cooldown",
			lastUsed:       ptr(now.Add(-2 * time.Minute)), // 2 mins ago
			duration:       defaultDuration,
			wantOnCooldown: true,
			wantRemaining:  3 * time.Minute, // 5 - 2 = 3
		},
		{
			name:           "expired cooldown",
			lastUsed:       ptr(now.Add(-6 * time.Minute)), // 6 mins ago
			duration:       defaultDuration,
			wantOnCooldown: false,
			wantRemaining:  0,
		},
		{
			name:           "exact boundary",
			lastUsed:       ptr(now.Add(-5 * time.Minute)), // 5 mins ago
			duration:       defaultDuration,
			wantOnCooldown: false,
			wantRemaining:  0,
		},
		{
			name:           "just before expiry",
			lastUsed:       ptr(now.Add(-5*time.Minute + 1*time.Second)), // 4m 59s ago
			duration:       defaultDuration,
			wantOnCooldown: true,
			wantRemaining:  1 * time.Second,
		},
		{
			name:           "future lastUsed (clock skew)",
			lastUsed:       ptr(now.Add(1 * time.Minute)), // 1 min in future
			duration:       defaultDuration,
			wantOnCooldown: true,
			wantRemaining:  6 * time.Minute, // 5 - (-1) = 6
		},
		{
			name:           "zero duration",
			lastUsed:       ptr(now), // used just now
			duration:       0,
			wantOnCooldown: false,
			wantRemaining:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOnCooldown, gotRemaining := b.checkCooldownInternal(now, tt.lastUsed, tt.duration)
			assert.Equal(t, tt.wantOnCooldown, gotOnCooldown)
			assert.Equal(t, tt.wantRemaining, gotRemaining)
		})
	}
}

func ptr(t time.Time) *time.Time {
	return &t
}
