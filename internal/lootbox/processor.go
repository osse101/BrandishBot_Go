package lootbox

import (
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

type dropInfo struct {
	Qty    int
	Chance float64
}

func (s *service) processLootTable(table []LootItem, quantity int) map[string]dropInfo {
	dropCounts := make(map[string]dropInfo)

	for _, loot := range table {
		if loot.Chance <= ZeroChanceThreshold {
			continue
		}

		if loot.Chance >= GuaranteedDropThreshold {
			s.processGuaranteedDrop(loot, quantity, dropCounts)
		} else {
			s.processChanceDrop(loot, quantity, dropCounts)
		}
	}
	return dropCounts
}

func (s *service) processGuaranteedDrop(loot LootItem, quantity int, dropCounts map[string]dropInfo) {
	totalQty := 0
	if loot.Max > loot.Min {
		for k := 0; k < quantity; k++ {
			totalQty += utils.SecureRandomIntRange(loot.Min, loot.Max)
		}
	} else {
		totalQty = quantity * loot.Min
	}

	s.updateDropCounts(loot, totalQty, dropCounts)
}

func (s *service) processChanceDrop(loot LootItem, quantity int, dropCounts map[string]dropInfo) {
	remaining := quantity
	for remaining > 0 {
		skip := utils.Geometric(loot.Chance)
		if skip >= remaining {
			break
		}

		remaining -= (skip + 1)
		qty := loot.Min
		if loot.Max > loot.Min {
			qty = utils.SecureRandomIntRange(loot.Min, loot.Max)
		}
		s.updateDropCounts(loot, qty, dropCounts)
	}
}

func (s *service) updateDropCounts(loot LootItem, qty int, dropCounts map[string]dropInfo) {
	info, exists := dropCounts[loot.ItemName]
	if !exists {
		info = dropInfo{Qty: 0, Chance: loot.Chance}
	} else if loot.Chance < info.Chance {
		info.Chance = loot.Chance
	}
	info.Qty += qty
	dropCounts[loot.ItemName] = info
}
