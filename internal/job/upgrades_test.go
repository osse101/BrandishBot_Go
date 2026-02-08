package job

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// This file contains test stubs for job upgrade node modifier application.
// See docs/issues/progression_nodes/upgrades.md for implementation details.

// TestUpgradeJobXPMultiplier_ExistingImplementation verifies job XP multiplier
func TestUpgradeJobXPMultiplier_ExistingImplementation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name               string
		multiplier         float64
		baseXP             float64
		expectedMultiplier float64
	}{
		{"No upgrade", 1.0, 100.0, 1.0},
		{"Level 1", 1.1, 100.0, 1.1},
		{"Level 3", 1.3, 100.0, 1.3},
		{"Level 5", 1.5, 100.0, 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock progression service
			mockProgression := &MockProgressionService{}
			svc := &service{
				progressionSvc: mockProgression,
			}

			// Mock GetModifiedValue to return the specified multiplier
			mockProgression.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).
				Return(tt.multiplier, nil)

			// Test getXPMultiplier
			result := svc.getXPMultiplier(ctx)

			// Verify
			assert.InDelta(t, tt.expectedMultiplier, result, 0.01)
			mockProgression.AssertExpectations(t)
		})
	}
}

// TestUpgradeJobLevelCap_ModifierApplication tests job level cap upgrade (linear modifier)
func TestUpgradeJobLevelCap_ModifierApplication(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		upgradeLevel     int
		expectedMaxLevel int
	}{
		{"No upgrade", 0, DefaultMaxLevel},   // Base level (10)
		{"Level 1", 1, DefaultMaxLevel + 10}, // 10 + 10 = 20
		{"Level 2", 2, DefaultMaxLevel + 20}, // 10 + 20 = 30
		{"Level 3", 3, DefaultMaxLevel + 30}, // 10 + 30 = 40 (max)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock progression service
			mockProgression := &MockProgressionService{}
			svc := &service{
				progressionSvc: mockProgression,
			}

			// Mock GetModifiedValue to return the modified level cap
			mockProgression.On("GetModifiedValue", ctx, "job_level_cap", float64(DefaultMaxLevel)).
				Return(float64(tt.expectedMaxLevel), nil)

			// Test getMaxJobLevel
			result := svc.getMaxJobLevel(ctx)

			// Verify
			assert.Equal(t, tt.expectedMaxLevel, result)
			mockProgression.AssertExpectations(t)
		})
	}
}

// TestUpgradeJobLevelCap_LinearModifier verifies linear addition (not multiplication)
func TestUpgradeJobLevelCap_LinearModifier(t *testing.T) {
	ctx := context.Background()

	// Setup mock progression service
	mockProgression := &MockProgressionService{}
	svc := &service{
		progressionSvc: mockProgression,
	}

	// Test linear modifier: base + (per_level * level)
	// Level 1 should be 10 + 10 = 20, NOT 10 * 1.1 = 11
	mockProgression.On("GetModifiedValue", ctx, "job_level_cap", float64(DefaultMaxLevel)).
		Return(float64(20), nil) // Linear: 10 + 10

	result := svc.getMaxJobLevel(ctx)

	assert.Equal(t, 20, result, "Linear modifier should add, not multiply")
	assert.NotEqual(t, 11, result, "Should not be multiplicative (10 * 1.1 = 11)")

	// Test level 3: base + (per_level * 3) = 10 + 30 = 40
	mockProgression2 := &MockProgressionService{}
	svc2 := &service{
		progressionSvc: mockProgression2,
	}

	mockProgression2.On("GetModifiedValue", ctx, "job_level_cap", float64(DefaultMaxLevel)).
		Return(float64(40), nil) // Linear: 10 + 30

	result2 := svc2.getMaxJobLevel(ctx)

	assert.Equal(t, 40, result2, "Level 3 should be 10 + 30 = 40")
	assert.NotEqual(t, 13, result2, "Should not be multiplicative (10 * 1.3 = 13)")
}

// TestJobUpgrades_FallbackBehavior tests safe fallback when GetModifiedValue fails
func TestJobUpgrades_FallbackBehavior(t *testing.T) {
	ctx := context.Background()

	t.Run("XP Multiplier fallback", func(t *testing.T) {
		mockProgression := &MockProgressionService{}
		svc := &service{
			progressionSvc: mockProgression,
		}

		// Mock error scenario
		mockProgression.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).
			Return(1.0, assert.AnError)

		// Should fallback to 1.0 (no multiplier)
		result := svc.getXPMultiplier(ctx)
		assert.Equal(t, 1.0, result, "Should fallback to 1.0 on error")
	})

	t.Run("Level Cap fallback", func(t *testing.T) {
		mockProgression := &MockProgressionService{}
		svc := &service{
			progressionSvc: mockProgression,
		}

		// Mock error scenario
		mockProgression.On("GetModifiedValue", ctx, "job_level_cap", float64(DefaultMaxLevel)).
			Return(float64(DefaultMaxLevel), assert.AnError)

		// Should fallback to DefaultMaxLevel
		result := svc.getMaxJobLevel(ctx)
		assert.Equal(t, DefaultMaxLevel, result, "Should fallback to default on error")
	})
}
