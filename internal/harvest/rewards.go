package harvest

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

func (s *service) calculateBonuses(ctx context.Context, userID string) (float64, float64) {
	log := logger.FromContext(ctx)
	yieldMultiplier := 1.0
	growthMultiplier := 1.0

	if yieldBonus, err := s.jobSvc.GetJobBonus(ctx, userID, "farmer", bonusTypeHarvestYield); err == nil {
		yieldMultiplier += yieldBonus
	} else {
		log.Warn("Failed to get yield bonus", "error", err)
	}

	if growthBonus, err := s.jobSvc.GetJobBonus(ctx, userID, "farmer", bonusTypeGrowthSpeed); err == nil {
		growthMultiplier += growthBonus
	} else {
		log.Warn("Failed to get growth bonus", "error", err)
	}
	return yieldMultiplier, growthMultiplier
}

func (s *service) calculateHarvestRewards(ctx context.Context, hoursElapsed float64, yieldMultiplier float64) (map[string]int, string) {
	if hoursElapsed > spoiledThreshold {
		logger.FromContext(ctx).Info("Harvest spoiled", "hours", hoursElapsed)
		return map[string]int{
			itemLootbox1: 1,
			itemStick:    3,
		}, "Your crops spoiled! You salvaged 1 Decent Lootbox and 3 Sticks."
	}
	return s.calculateRewards(ctx, hoursElapsed, yieldMultiplier), "Harvest successful!"
}

// calculateRewards calculates the total rewards for a given elapsed time
// Accumulates ALL items from all tiers up to and including the current tier
func (s *service) calculateRewards(ctx context.Context, hoursElapsed float64, yieldMultiplier float64) map[string]int {
	log := logger.FromContext(ctx)
	rewards := make(map[string]int)
	tiers := getRewardTiers()

	// Find the applicable tier (highest tier where hoursElapsed >= MaxHours)
	maxTierIndex := -1
	for i := range tiers {
		if hoursElapsed >= tiers[i].MaxHours {
			maxTierIndex = i
		} else {
			break // Tiers are ordered, so we can stop here
		}
	}

	// No tier reached
	if maxTierIndex < 0 {
		log.Info("No tier reached", "hoursElapsed", hoursElapsed)
		return rewards
	}

	log.Info("Calculating rewards", "hoursElapsed", hoursElapsed, "maxTier", maxTierIndex)

	// ACCUMULATE ALL ITEMS from all tiers up to and including current tier
	for i := 0; i <= maxTierIndex; i++ {
		tier := &tiers[i]

		for itemName, quantity := range tier.Items {
			// Check progression unlock for gated items
			if tier.RequiresUnlock[itemName] {
				unlocked, err := s.progressionSvc.IsItemUnlocked(ctx, itemName)
				if err != nil {
					log.Warn("Failed to check item unlock status", "item", itemName, "error", err)
					continue // Skip on error
				}
				if !unlocked {
					log.Info("Item locked by progression, skipping", "item", itemName, "tier", i)
					continue // Skip locked items
				}
			}

			// SUM all items (accumulate)
			baseQty := quantity
			bonusQty := int(float64(baseQty) * yieldMultiplier)
			if bonusQty < baseQty {
				bonusQty = baseQty
			}
			rewards[itemName] += bonusQty
		}
	}

	log.Info("Rewards calculated", "rewards", rewards, "tierCount", maxTierIndex+1)

	return rewards
}

// getRewardTiers returns the harvest reward tier configuration
// Rewards accumulate - users receive ALL items from all tiers up to their current tier
func getRewardTiers() []domain.HarvestReward {
	return []domain.HarvestReward{
		// Tier 1: 2 hours - 2 money
		{
			MaxHours: 2.0,
			Items: map[string]int{
				itemMoney: 2,
			},
			RequiresUnlock: map[string]bool{},
		},
		// Tier 2: 5 hours - +10 money (total: 12 money)
		{
			MaxHours: 5.0,
			Items: map[string]int{
				itemMoney: 10,
			},
			RequiresUnlock: map[string]bool{},
		},
		// Tier 3: 12 hours - +1 stick, +5 money (total: 17 money, 1 stick if unlocked)
		{
			MaxHours: 12.0,
			Items: map[string]int{
				itemStick: 1,
				itemMoney: 5,
			},
			RequiresUnlock: map[string]bool{
				itemStick: true,
			},
		},
		// Tier 4: 24 hours - +2 stick, +5 money (total: 22 money, 3 stick if unlocked)
		{
			MaxHours: 24.0,
			Items: map[string]int{
				itemStick: 2,
				itemMoney: 5,
			},
			RequiresUnlock: map[string]bool{
				itemStick: true,
			},
		},
		// Tier 5: 48 hours - +1 lootbox0, +10 money (total: 32 money, 3 stick, 1 lootbox0)
		{
			MaxHours: 48.0,
			Items: map[string]int{
				itemLootbox0: 1,
				itemMoney:    10,
			},
			RequiresUnlock: map[string]bool{},
		},
		// Tier 6: 72 hours - +2 lootbox0, +10 money (total: 42 money, 3 stick, 3 lootbox0)
		{
			MaxHours: 72.0,
			Items: map[string]int{
				itemLootbox0: 2,
				itemMoney:    10,
			},
			RequiresUnlock: map[string]bool{},
		},
		// Tier 7: 90 hours - +5 stick, +5 money (total: 47 money, 8 stick, 3 lootbox0 if stick unlocked)
		{
			MaxHours: 90.0,
			Items: map[string]int{
				itemStick: 5,
				itemMoney: 5,
			},
			RequiresUnlock: map[string]bool{
				itemStick: true,
			},
		},
		// Tier 8: 110 hours - +1 lootbox1, +15 money (total: 62 money, 8 stick, 3 lootbox0, 1 lootbox1 if unlocked)
		{
			MaxHours: 110.0,
			Items: map[string]int{
				itemLootbox1: 1,
				itemMoney:    15,
			},
			RequiresUnlock: map[string]bool{
				itemLootbox1: true,
			},
		},
		// Tier 9: 130 hours - +1 lootbox1, +15 money (total: 77 money, 8 stick, 3 lootbox0, 2 lootbox1 if unlocked)
		{
			MaxHours: 130.0,
			Items: map[string]int{
				itemLootbox1: 1,
				itemMoney:    15,
			},
			RequiresUnlock: map[string]bool{
				itemLootbox1: true,
			},
		},
		// Tier 10: 168 hours (1 week) - +1 lootbox2, +20 money (total: 97 money, 8 stick, 3 lootbox0, 2 lootbox1, 1 lootbox2 if unlocked)
		{
			MaxHours: 168.0,
			Items: map[string]int{
				itemLootbox2: 1,
				itemMoney:    20,
			},
			RequiresUnlock: map[string]bool{
				itemLootbox2: true,
			},
		},
	}
}
