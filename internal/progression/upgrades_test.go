package progression

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// This file contains test stubs for upgrade node modifier application.
// See docs/issues/progression_nodes/upgrades.md for full implementation details.

// TestUpgradeProgressionBasic_ModifierApplication tests single progression rate modifier
func TestUpgradeProgressionBasic_ModifierApplication(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		level         int
		baseValue     float64
		expectedValue float64
	}{
		{"Level 1", 1, 100.0, 110.0}, // 1.0 + 0.1*1 = 1.1x
		{"Level 3", 3, 100.0, 130.0}, // 1.0 + 0.1*3 = 1.3x
		{"Level 5", 5, 100.0, 150.0}, // 1.0 + 0.1*5 = 1.5x
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := NewMockRepository()
			svc := &service{
				repo:          mockRepo,
				modifierCache: NewModifierCache(30 * time.Minute),
			}

			// Create node with progression_rate modifier
			node := &domain.ProgressionNode{
				ID:      1,
				NodeKey: "upgrade_progression_basic",
				ModifierConfig: &domain.ModifierConfig{
					FeatureKey:    "progression_rate",
					ModifierType:  "multiplicative",
					BaseValue:     1.0,
					PerLevelValue: 0.1,
				},
				Tier: 1,
			}
			mockRepo.nodes[1] = node
			mockRepo.nodesByKey["upgrade_progression_basic"] = node

			// Unlock to specified level
			if mockRepo.unlocks[1] == nil {
				mockRepo.unlocks[1] = make(map[int]*domain.ProgressionUnlock)
			}
			mockRepo.unlocks[1][tt.level] = &domain.ProgressionUnlock{
				NodeID:       1,
				CurrentLevel: tt.level,
				UnlockedBy:   "admin",
			}

			// Test GetModifiedValue
			result, err := svc.GetModifiedValue(ctx, "progression_rate", tt.baseValue)

			// Verify (with small tolerance for floating point)
			require.NoError(t, err)
			assert.InDelta(t, tt.expectedValue, result, 0.01)
		})
	}
}

// TestUpgradeProgressionTwo_StackingWithBasic tests double stacking of progression modifiers
func TestUpgradeProgressionTwo_StackingWithBasic(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		basicLevel    int
		tier2Level    int
		baseValue     float64
		expectedValue float64
	}{
		{"Both Level 1", 1, 1, 100.0, 121.0}, // 1.1 * 1.1 = 1.21x
		{"Both Level 3", 3, 3, 100.0, 169.0}, // 1.3 * 1.3 = 1.69x
		{"Both Level 5", 5, 5, 100.0, 225.0}, // 1.5 * 1.5 = 2.25x
		{"Mixed 1+5", 1, 5, 100.0, 165.0},    // 1.1 * 1.5 = 1.65x
		{"Mixed 5+1", 5, 1, 100.0, 165.0},    // 1.5 * 1.1 = 1.65x
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := NewMockRepository()
			svc := &service{
				repo:          mockRepo,
				modifierCache: NewModifierCache(30 * time.Minute),
			}

			// Create first node (tier 1)
			node1 := &domain.ProgressionNode{
				ID:      1,
				NodeKey: "upgrade_progression_basic",
				ModifierConfig: &domain.ModifierConfig{
					FeatureKey:    "progression_rate",
					ModifierType:  "multiplicative",
					BaseValue:     1.0,
					PerLevelValue: 0.1,
				},
				Tier: 1,
			}
			mockRepo.nodes[1] = node1
			mockRepo.nodesByKey["upgrade_progression_basic"] = node1

			// Create second node (tier 3)
			node2 := &domain.ProgressionNode{
				ID:      2,
				NodeKey: "upgrade_progression_two",
				ModifierConfig: &domain.ModifierConfig{
					FeatureKey:    "progression_rate",
					ModifierType:  "multiplicative",
					BaseValue:     1.0,
					PerLevelValue: 0.1,
				},
				Tier: 3,
			}
			mockRepo.nodes[2] = node2
			mockRepo.nodesByKey["upgrade_progression_two"] = node2

			// Unlock both nodes
			mockRepo.unlocks[1] = make(map[int]*domain.ProgressionUnlock)
			mockRepo.unlocks[1][tt.basicLevel] = &domain.ProgressionUnlock{
				NodeID:       1,
				CurrentLevel: tt.basicLevel,
				UnlockedBy:   "admin",
			}

			mockRepo.unlocks[2] = make(map[int]*domain.ProgressionUnlock)
			mockRepo.unlocks[2][tt.tier2Level] = &domain.ProgressionUnlock{
				NodeID:       2,
				CurrentLevel: tt.tier2Level,
				UnlockedBy:   "admin",
			}

			// Test GetModifiedValue
			result, err := svc.GetModifiedValue(ctx, "progression_rate", tt.baseValue)

			// Verify (with small tolerance for floating point)
			require.NoError(t, err)
			assert.InDelta(t, tt.expectedValue, result, 0.01)
		})
	}
}

// TestUpgradeProgressionThree_TripleStacking tests triple stacking of all progression modifiers
func TestUpgradeProgressionThree_TripleStacking(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		level1        int
		level2        int
		level3        int
		baseValue     float64
		expectedValue float64
	}{
		{"All Level 1", 1, 1, 1, 100.0, 133.1}, // 1.1 * 1.1 * 1.1 = 1.331x
		{"All Level 3", 3, 3, 3, 100.0, 219.7}, // 1.3 * 1.3 * 1.3 = 2.197x
		{"All Level 5", 5, 5, 5, 100.0, 337.5}, // 1.5 * 1.5 * 1.5 = 3.375x
		{"Mixed 1+3+5", 1, 3, 5, 100.0, 214.5}, // 1.1 * 1.3 * 1.5 = 2.145x
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := NewMockRepository()
			svc := &service{
				repo:          mockRepo,
				modifierCache: NewModifierCache(30 * time.Minute),
			}

			// Create all three nodes
			node1 := &domain.ProgressionNode{
				ID:      1,
				NodeKey: "upgrade_progression_basic",
				ModifierConfig: &domain.ModifierConfig{
					FeatureKey:    "progression_rate",
					ModifierType:  "multiplicative",
					BaseValue:     1.0,
					PerLevelValue: 0.1,
				},
				Tier: 1,
			}
			mockRepo.nodes[1] = node1
			mockRepo.nodesByKey["upgrade_progression_basic"] = node1

			node2 := &domain.ProgressionNode{
				ID:      2,
				NodeKey: "upgrade_progression_two",
				ModifierConfig: &domain.ModifierConfig{
					FeatureKey:    "progression_rate",
					ModifierType:  "multiplicative",
					BaseValue:     1.0,
					PerLevelValue: 0.1,
				},
				Tier: 3,
			}
			mockRepo.nodes[2] = node2
			mockRepo.nodesByKey["upgrade_progression_two"] = node2

			node3 := &domain.ProgressionNode{
				ID:      3,
				NodeKey: "upgrade_progression_three",
				ModifierConfig: &domain.ModifierConfig{
					FeatureKey:    "progression_rate",
					ModifierType:  "multiplicative",
					BaseValue:     1.0,
					PerLevelValue: 0.1,
				},
				Tier: 4,
			}
			mockRepo.nodes[3] = node3
			mockRepo.nodesByKey["upgrade_progression_three"] = node3

			// Unlock all three nodes
			mockRepo.unlocks[1] = make(map[int]*domain.ProgressionUnlock)
			mockRepo.unlocks[1][tt.level1] = &domain.ProgressionUnlock{
				NodeID:       1,
				CurrentLevel: tt.level1,
				UnlockedBy:   "admin",
			}

			mockRepo.unlocks[2] = make(map[int]*domain.ProgressionUnlock)
			mockRepo.unlocks[2][tt.level2] = &domain.ProgressionUnlock{
				NodeID:       2,
				CurrentLevel: tt.level2,
				UnlockedBy:   "admin",
			}

			mockRepo.unlocks[3] = make(map[int]*domain.ProgressionUnlock)
			mockRepo.unlocks[3][tt.level3] = &domain.ProgressionUnlock{
				NodeID:       3,
				CurrentLevel: tt.level3,
				UnlockedBy:   "admin",
			}

			// Test GetModifiedValue
			result, err := svc.GetModifiedValue(ctx, "progression_rate", tt.baseValue)

			// Verify (with small tolerance for floating point)
			require.NoError(t, err)
			assert.InDelta(t, tt.expectedValue, result, 0.1)
		})
	}
}

// TestProgressionUpgrades_CacheInvalidation tests that cache is properly invalidated on unlock
func TestProgressionUpgrades_CacheInvalidation(t *testing.T) {
	ctx := context.Background()

	// Setup mock repository
	mockRepo := NewMockRepository()
	svc := &service{
		repo:          mockRepo,
		modifierCache: NewModifierCache(30 * time.Minute),
	}

	// Create node
	node := &domain.ProgressionNode{
		ID:      1,
		NodeKey: "upgrade_progression_basic",
		ModifierConfig: &domain.ModifierConfig{
			FeatureKey:    "progression_rate",
			ModifierType:  "multiplicative",
			BaseValue:     1.0,
			PerLevelValue: 0.1,
		},
		Tier: 1,
	}
	mockRepo.nodes[1] = node
	mockRepo.nodesByKey["upgrade_progression_basic"] = node

	// Initial state: level 1
	mockRepo.unlocks[1] = make(map[int]*domain.ProgressionUnlock)
	mockRepo.unlocks[1][1] = &domain.ProgressionUnlock{
		NodeID:       1,
		CurrentLevel: 1,
		UnlockedBy:   "admin",
	}

	// First call - should return 1.1x and cache it
	result1, err := svc.GetModifiedValue(ctx, "progression_rate", 100.0)
	require.NoError(t, err)
	assert.InDelta(t, 110.0, result1, 0.01)

	// Verify cache hit on second call
	result2, err := svc.GetModifiedValue(ctx, "progression_rate", 100.0)
	require.NoError(t, err)
	assert.InDelta(t, 110.0, result2, 0.01)

	// Upgrade to level 5
	mockRepo.unlocks[1][5] = &domain.ProgressionUnlock{
		NodeID:       1,
		CurrentLevel: 5,
		UnlockedBy:   "admin",
	}

	// Cache should still return old value
	result3, err := svc.GetModifiedValue(ctx, "progression_rate", 100.0)
	require.NoError(t, err)
	assert.InDelta(t, 110.0, result3, 0.01, "Cache should return stale value before invalidation")

	// Invalidate cache
	svc.modifierCache.InvalidateAll()

	// Now should return new value (1.5x)
	result4, err := svc.GetModifiedValue(ctx, "progression_rate", 100.0)
	require.NoError(t, err)
	assert.InDelta(t, 150.0, result4, 0.01, "Should return updated value after cache invalidation")
}

// TestProgressionUpgrades_FallbackBehavior tests safe fallback when GetModifiedValue fails
func TestProgressionUpgrades_FallbackBehavior(t *testing.T) {
	ctx := context.Background()

	// Setup mock repository that returns empty results
	mockRepo := NewMockRepository()
	svc := &service{
		repo:          mockRepo,
		modifierCache: NewModifierCache(30 * time.Minute),
	}

	// Test with no modifiers configured - should return base value
	result, err := svc.GetModifiedValue(ctx, "nonexistent_feature", 100.0)
	require.NoError(t, err)
	assert.Equal(t, 100.0, result, "Should return base value when no modifiers exist")
}
