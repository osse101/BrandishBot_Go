package harvest

import "github.com/osse101/BrandishBot_Go/internal/domain"

const (
	minHarvestInterval = 1.0   // Minimum 1 hour between harvests
	farmerXPThreshold  = 5.0   // Minimum 5 hours for Farmer XP
	farmerXPPerHour    = 8     // Base XP per hour of waiting
	spoiledThreshold   = 336.0 // 168h (max tier) + 168h (1 week)

	// Bonus types
	bonusTypeHarvestYield = "harvest_yield"
	bonusTypeGrowthSpeed  = "growth_speed"
)

// Item internal names used in harvest system.
const (
	itemMoney    = domain.ItemMoney
	itemStick    = domain.ItemStick
	itemLootbox0 = domain.ItemLootbox0
	itemLootbox1 = domain.ItemLootbox1
	itemLootbox2 = domain.ItemLootbox2
)
