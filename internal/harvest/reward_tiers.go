package harvest

import "github.com/osse101/BrandishBot_Go/internal/domain"

// getRewardTiers returns the harvest reward tier configuration
// Rewards accumulate - users receive ALL items from all tiers up to their current tier
func getRewardTiers() []domain.HarvestReward {
	return []domain.HarvestReward{
		// Tier 1: 2 hours - 2 money
		{
			MaxHours: 2.0,
			Items: map[string]int{
				"money": 2,
			},
			RequiresUnlock: map[string]bool{},
		},
		// Tier 2: 5 hours - +10 money (total: 12 money)
		{
			MaxHours: 5.0,
			Items: map[string]int{
				"money": 10,
			},
			RequiresUnlock: map[string]bool{},
		},
		// Tier 3: 12 hours - +1 stick, +5 money (total: 17 money, 1 stick if unlocked)
		{
			MaxHours: 12.0,
			Items: map[string]int{
				"stick": 1,
				"money": 5,
			},
			RequiresUnlock: map[string]bool{
				"stick": true,
			},
		},
		// Tier 4: 24 hours - +2 stick, +5 money (total: 22 money, 3 stick if unlocked)
		{
			MaxHours: 24.0,
			Items: map[string]int{
				"stick": 2,
				"money": 5,
			},
			RequiresUnlock: map[string]bool{
				"stick": true,
			},
		},
		// Tier 5: 48 hours - +1 lootbox0, +10 money (total: 32 money, 3 stick, 1 lootbox0)
		{
			MaxHours: 48.0,
			Items: map[string]int{
				"lootbox0": 1,
				"money":    10,
			},
			RequiresUnlock: map[string]bool{},
		},
		// Tier 6: 72 hours - +2 lootbox0, +10 money (total: 42 money, 3 stick, 3 lootbox0)
		{
			MaxHours: 72.0,
			Items: map[string]int{
				"lootbox0": 2,
				"money":    10,
			},
			RequiresUnlock: map[string]bool{},
		},
		// Tier 7: 90 hours - +5 stick, +5 money (total: 47 money, 8 stick, 3 lootbox0 if stick unlocked)
		{
			MaxHours: 90.0,
			Items: map[string]int{
				"stick": 5,
				"money": 5,
			},
			RequiresUnlock: map[string]bool{
				"stick": true,
			},
		},
		// Tier 8: 110 hours - +1 lootbox1, +15 money (total: 62 money, 8 stick, 3 lootbox0, 1 lootbox1 if unlocked)
		{
			MaxHours: 110.0,
			Items: map[string]int{
				"lootbox1": 1,
				"money":    15,
			},
			RequiresUnlock: map[string]bool{
				"lootbox1": true,
			},
		},
		// Tier 9: 130 hours - +1 lootbox1, +15 money (total: 77 money, 8 stick, 3 lootbox0, 2 lootbox1 if unlocked)
		{
			MaxHours: 130.0,
			Items: map[string]int{
				"lootbox1": 1,
				"money":    15,
			},
			RequiresUnlock: map[string]bool{
				"lootbox1": true,
			},
		},
		// Tier 10: 168 hours (1 week) - +1 lootbox2, +20 money (total: 97 money, 8 stick, 3 lootbox0, 2 lootbox1, 1 lootbox2 if unlocked)
		{
			MaxHours: 168.0,
			Items: map[string]int{
				"lootbox2": 1,
				"money":    20,
			},
			RequiresUnlock: map[string]bool{
				"lootbox2": true,
			},
		},
	}
}
