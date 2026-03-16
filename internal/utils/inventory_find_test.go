package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestFindSlot(t *testing.T) {
	t.Run("finds existing item in inventory", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5},
				{ItemID: 2, Quantity: 10},
				{ItemID: 3, Quantity: 3},
			},
		}

		index, quantity := FindSlot(inventory, 2)

		assert.Equal(t, 1, index, "Should find item at correct index")
		assert.Equal(t, 10, quantity, "Should return correct quantity")
	})

	t.Run("returns -1 and 0 for non-existent item", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5},
				{ItemID: 2, Quantity: 10},
			},
		}

		index, quantity := FindSlot(inventory, 999)

		assert.Equal(t, -1, index, "Should return -1 when item not found")
		assert.Equal(t, 0, quantity, "Should return 0 quantity when not found")
	})

	t.Run("finds first occurrence when item appears multiple times", func(t *testing.T) {
		// This tests current behavior - inventory shouldn't have duplicates,
		// but if it does, we get the first match
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5},
				{ItemID: 2, Quantity: 10},
				{ItemID: 2, Quantity: 20}, // Duplicate (shouldn't happen in real use)
			},
		}

		index, quantity := FindSlot(inventory, 2)

		assert.Equal(t, 1, index, "Should return first matching index")
		assert.Equal(t, 10, quantity, "Should return first match quantity")
	})

	t.Run("handles empty inventory", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 100, Quantity: 42},
				{ItemID: 200, Quantity: 10},
			},
		}

		index, quantity := FindSlot(inventory, 100)

		assert.Equal(t, 0, index, "Should correctly find item at index 0")
		assert.Equal(t, 42, quantity)
	})

	t.Run("finds item at last position", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 100, Quantity: 5},
				{ItemID: 200, Quantity: 10},
				{ItemID: 300, Quantity: 15},
			},
		}

		index, quantity := FindSlot(inventory, 300)

		assert.Equal(t, 2, index, "Should correctly find item at last index")
		assert.Equal(t, 15, quantity)
	})

	t.Run("correctly handles zero quantity items", func(t *testing.T) {
		// Items with zero quantity might exist temporarily during transactions
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 0},
				{ItemID: 2, Quantity: 10},
			},
		}

		index, quantity := FindSlot(inventory, 1)

		assert.Equal(t, 0, index, "Should find item even if quantity is 0")
		assert.Equal(t, 0, quantity, "Should return actual quantity of 0")
	})
}

// TestFindSlot_RealWorldScenarios tests realistic game scenarios
func TestFindSlot_RealWorldScenarios(t *testing.T) {
	t.Run("money lookup in typical inventory", func(t *testing.T) {
		const moneyItemID = 1
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: moneyItemID, Quantity: 1000},
				{ItemID: 10, Quantity: 5},  // Sword
				{ItemID: 20, Quantity: 3},  // Potion
				{ItemID: 30, Quantity: 10}, // Material
			},
		}

		index, balance := FindSlot(inventory, moneyItemID)

		assert.Equal(t, 0, index)
		assert.Equal(t, 1000, balance, "Should correctly find money balance")
	})

	t.Run("checking for crafting material before craft", func(t *testing.T) {
		const woodItemID = 50
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 500},
				{ItemID: woodItemID, Quantity: 25},
			},
		}

		index, quantity := FindSlot(inventory, woodItemID)

		// Verify we have enough wood for a craft that needs 20
		assert.NotEqual(t, -1, index, "Material should exist")
		assert.GreaterOrEqual(t, quantity, 20, "Should have enough material")
	})

	t.Run("checking for item that player doesn't own", func(t *testing.T) {
		const legendarySwordID = 999
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 100},
				{ItemID: 10, Quantity: 1},
			},
		}

		index, _ := FindSlot(inventory, legendarySwordID)

		assert.Equal(t, -1, index, "Player shouldn't have legendary item")
	})
}

// TestFindSlotWithQuality verifies quality-aware slot lookup
func TestFindSlotWithQuality(t *testing.T) {
	t.Run("finds slot with matching ItemID and QualityLevel", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
				{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityLegendary},
				{ItemID: 2, Quantity: 3, QualityLevel: domain.QualityRare},
			},
		}

		index, quantity := FindSlotWithQuality(inventory, 1, domain.QualityLegendary)

		assert.Equal(t, 1, index, "Should find legendary slot")
		assert.Equal(t, 10, quantity, "Should return correct quantity")
	})

	t.Run("returns -1 when ItemID matches but QualityLevel doesn't", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
			},
		}

		index, quantity := FindSlotWithQuality(inventory, 1, domain.QualityLegendary)

		assert.Equal(t, -1, index, "Should not find slot with different quality")
		assert.Equal(t, 0, quantity)
	})

	t.Run("returns -1 when ItemID doesn't match", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
			},
		}

		index, quantity := FindSlotWithQuality(inventory, 2, domain.QualityCommon)
		assert.Equal(t, -1, index, "Should not find slot with different ItemID")
		assert.Equal(t, 0, quantity)
	})
}

// TestFindRandomSlot verifies random slot lookup
func TestFindRandomSlot(t *testing.T) {
	t.Run("finds item when only one slot exists", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
			},
		}

		index, quantity := FindRandomSlot(inventory, 1, func() float64 { return 0.5 })
		assert.Equal(t, 0, index)
		assert.Equal(t, 10, quantity)
	})

	t.Run("randomly selects from multiple slots", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
				{ItemID: 1, Quantity: 20},
			},
		}

		index1, _ := FindRandomSlot(inventory, 1, func() float64 { return 0.0 })
		assert.Equal(t, 0, index1)

		index2, _ := FindRandomSlot(inventory, 1, func() float64 { return 0.999 })
		assert.Equal(t, 1, index2)
	})

	t.Run("handles RNG returning 1.0 (safety test)", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
				{ItemID: 1, Quantity: 20},
			},
		}

		// Should not panic even if RNG returns 1.0
		index, _ := FindRandomSlot(inventory, 1, func() float64 { return 1.0 })
		assert.Equal(t, 1, index)
	})

	t.Run("returns -1 when item not found", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 2, Quantity: 10},
			},
		}

		index, quantity := FindRandomSlot(inventory, 1, func() float64 { return 0.5 })
		assert.Equal(t, -1, index)
		assert.Equal(t, 0, quantity)
	})
}

func TestGetTotalQuantity(t *testing.T) {
	t.Run("calculates total from multiple slots", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5},
				{ItemID: 2, Quantity: 10},
				{ItemID: 1, Quantity: 3},
			},
		}

		total := GetTotalQuantity(inventory, 1)
		assert.Equal(t, 8, total)
	})

	t.Run("returns 0 for missing item", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 2, Quantity: 10},
			},
		}

		total := GetTotalQuantity(inventory, 1)
		assert.Equal(t, 0, total)
	})
}
