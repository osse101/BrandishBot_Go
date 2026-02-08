package harvest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestCalculateRewards(t *testing.T) {
	tests := []struct {
		name           string
		hoursElapsed   float64
		unlockedItems  map[string]bool
		expectedReward map[string]int
	}{
		{
			name:         "Less than 2 hours - no tier reached",
			hoursElapsed: 1.5,
			unlockedItems: map[string]bool{
				"stick":    true,
				"lootbox1": true,
				"lootbox2": true,
			},
			expectedReward: map[string]int{}, // No tier reached
		},
		{
			name:         "Exactly 2 hours - Tier 1",
			hoursElapsed: 2.0,
			unlockedItems: map[string]bool{
				"stick":    true,
				"lootbox1": true,
				"lootbox2": true,
			},
			expectedReward: map[string]int{
				"money": 2,
			},
		},
		{
			name:         "5 hours - Tier 1 + 2",
			hoursElapsed: 5.0,
			unlockedItems: map[string]bool{
				"stick":    true,
				"lootbox1": true,
				"lootbox2": true,
			},
			expectedReward: map[string]int{
				"money": 12, // 2 + 10
			},
		},
		{
			name:         "24 hours - All stick tiers, stick unlocked",
			hoursElapsed: 24.0,
			unlockedItems: map[string]bool{
				"stick":    true,
				"lootbox1": false,
				"lootbox2": false,
			},
			expectedReward: map[string]int{
				"money": 22, // 2 + 10 + 5 + 5
				"stick": 3,  // 1 + 2
			},
		},
		{
			name:         "24 hours - stick NOT unlocked",
			hoursElapsed: 24.0,
			unlockedItems: map[string]bool{
				"stick":    false,
				"lootbox1": false,
				"lootbox2": false,
			},
			expectedReward: map[string]int{
				"money": 22, // 2 + 10 + 5 + 5 (money from stick tiers still counts)
			},
		},
		{
			name:         "48 hours - includes lootbox0",
			hoursElapsed: 48.0,
			unlockedItems: map[string]bool{
				"stick":    true,
				"lootbox1": false,
				"lootbox2": false,
			},
			expectedReward: map[string]int{
				"money":    32, // 2 + 10 + 5 + 5 + 10
				"stick":    3,  // 1 + 2
				"lootbox0": 1,  // lootbox0 doesn't require unlock
			},
		},
		{
			name:         "168 hours - max tier, all unlocked",
			hoursElapsed: 168.0,
			unlockedItems: map[string]bool{
				"stick":    true,
				"lootbox1": true,
				"lootbox2": true,
			},
			expectedReward: map[string]int{
				"money":    97, // 2 + 10 + 5 + 5 + 10 + 10 + 5 + 15 + 15 + 20
				"stick":    8,  // 1 + 2 + 5
				"lootbox0": 3,  // 1 + 2
				"lootbox1": 2,  // 1 + 1
				"lootbox2": 1,  // 1
			},
		},
		{
			name:         "168 hours - max tier, lootboxes NOT unlocked",
			hoursElapsed: 168.0,
			unlockedItems: map[string]bool{
				"stick":    true,
				"lootbox1": false,
				"lootbox2": false,
			},
			expectedReward: map[string]int{
				"money":    97, // 2 + 10 + 5 + 5 + 10 + 10 + 5 + 15 + 15 + 20 (all money counts)
				"stick":    8,  // 1 + 2 + 5
				"lootbox0": 3,  // 1 + 2 (lootbox0 doesn't require unlock)
			},
		},
		{
			name:         "200 hours - beyond max tier",
			hoursElapsed: 200.0,
			unlockedItems: map[string]bool{
				"stick":    true,
				"lootbox1": true,
				"lootbox2": true,
			},
			expectedReward: map[string]int{
				"money":    97, // Same as 168h tier
				"stick":    8,
				"lootbox0": 3,
				"lootbox1": 2,
				"lootbox2": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockProgressionSvc := new(mocks.MockProgressionService)

			// Setup IsItemUnlocked expectations
			for itemName, unlocked := range tt.unlockedItems {
				mockProgressionSvc.On("IsItemUnlocked", mock.Anything, itemName).Return(unlocked, nil).Maybe()
			}

			// Create service
			svc := &service{
				progressionSvc: mockProgressionSvc,
			}

			// Execute
			rewards := svc.calculateRewards(context.Background(), tt.hoursElapsed)

			// Assert
			assert.Equal(t, tt.expectedReward, rewards)

			mockProgressionSvc.AssertExpectations(t)
		})
	}
}

func TestRewardTiers(t *testing.T) {
	tiers := getRewardTiers()

	// Verify tiers are ordered by MaxHours
	for i := 1; i < len(tiers); i++ {
		assert.Greater(t, tiers[i].MaxHours, tiers[i-1].MaxHours,
			"Tiers must be ordered by MaxHours")
	}

	// Verify tier structure
	assert.Equal(t, 10, len(tiers), "Should have 10 tiers")

	// Verify first tier
	assert.Equal(t, 2.0, tiers[0].MaxHours)
	assert.Equal(t, 2, tiers[0].Items["money"])
	assert.Empty(t, tiers[0].RequiresUnlock)

	// Verify last tier
	assert.Equal(t, 168.0, tiers[9].MaxHours)
	assert.Equal(t, 20, tiers[9].Items["money"])
	assert.Equal(t, 1, tiers[9].Items["lootbox2"])
	assert.True(t, tiers[9].RequiresUnlock["lootbox2"])
}

func TestMinHarvestInterval(t *testing.T) {
	assert.Equal(t, 1.0, minHarvestInterval, "Minimum harvest interval should be 1 hour")
}
