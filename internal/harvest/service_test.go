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
		name            string
		hoursElapsed    float64
		unlockedItems   map[string]bool
		expectedReward  map[string]int
		yieldMultiplier float64
	}{
		{
			name:         "Less than 2 hours - no tier reached",
			hoursElapsed: 1.5,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{}, // No tier reached
		},
		{
			name:         "Exactly 2 hours - Tier 1",
			hoursElapsed: 2.0,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{
				itemMoney: 2,
			},
		},
		{
			name:         "5 hours - Tier 1 + 2",
			hoursElapsed: 5.0,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{
				itemMoney: 12, // 2 + 10
			},
		},
		{
			name:         "24 hours - All stick tiers, stick unlocked",
			hoursElapsed: 24.0,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: false,
				itemLootbox2: false,
			},
			expectedReward: map[string]int{
				itemMoney: 22, // 2 + 10 + 5 + 5
				itemStick: 3,  // 1 + 2
			},
		},
		{
			name:         "24 hours - stick NOT unlocked",
			hoursElapsed: 24.0,
			unlockedItems: map[string]bool{
				itemStick:    false,
				itemLootbox1: false,
				itemLootbox2: false,
			},
			expectedReward: map[string]int{
				itemMoney: 22, // 2 + 10 + 5 + 5 (money from stick tiers still counts)
			},
		},
		{
			name:         "48 hours - includes lootbox0",
			hoursElapsed: 48.0,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: false,
				itemLootbox2: false,
			},
			expectedReward: map[string]int{
				itemMoney:    32, // 2 + 10 + 5 + 5 + 10
				itemStick:    3,  // 1 + 2
				itemLootbox0: 1,  // lootbox0 doesn't require unlock
			},
		},
		{
			name:         "168 hours - max tier, all unlocked",
			hoursElapsed: 168.0,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{
				itemMoney:    97, // 2 + 10 + 5 + 5 + 10 + 10 + 5 + 15 + 15 + 20
				itemStick:    8,  // 1 + 2 + 5
				itemLootbox0: 3,  // 1 + 2
				itemLootbox1: 2,  // 1 + 1
				itemLootbox2: 1,  // 1
			},
		},
		{
			name:         "168 hours - max tier, lootboxes NOT unlocked",
			hoursElapsed: 168.0,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: false,
				itemLootbox2: false,
			},
			expectedReward: map[string]int{
				itemMoney:    97, // 2 + 10 + 5 + 5 + 10 + 10 + 5 + 15 + 15 + 20 (all money counts)
				itemStick:    8,  // 1 + 2 + 5
				itemLootbox0: 3,  // 1 + 2 (lootbox0 doesn't require unlock)
			},
		},
		{
			name:         "200 hours - beyond max tier",
			hoursElapsed: 200.0,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{
				itemMoney:    97, // Same as 168h tier
				itemStick:    8,
				itemLootbox0: 3,
				itemLootbox1: 2,
				itemLootbox2: 1,
			},
		},
		{
			name:         "Yield Bonus - 1.5x multiplier",
			hoursElapsed: 5.0, // Tier 1 + 2 (12 money)
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{
				itemMoney: 18, // 12 * 1.5 = 18
			},
			yieldMultiplier: 1.5,
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
			multiplier := 1.0
			if tt.yieldMultiplier > 0 {
				multiplier = tt.yieldMultiplier
			}
			rewards := svc.calculateRewards(context.Background(), tt.hoursElapsed, multiplier)

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
	assert.Equal(t, 2, tiers[0].Items[itemMoney])
	assert.Empty(t, tiers[0].RequiresUnlock)

	// Verify last tier
	assert.Equal(t, 168.0, tiers[9].MaxHours)
	assert.Equal(t, 20, tiers[9].Items[itemMoney])
	assert.Equal(t, 1, tiers[9].Items[itemLootbox2])
	assert.True(t, tiers[9].RequiresUnlock[itemLootbox2])
}

func TestMinHarvestInterval(t *testing.T) {
	assert.Equal(t, 1.0, minHarvestInterval, "Minimum harvest interval should be 1 hour")
}
