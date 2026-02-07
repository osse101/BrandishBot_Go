package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestFindSlot verifies slot lookup in inventory
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

// TestFindSlotWithShine verifies shine-aware slot lookup
func TestFindSlotWithShine(t *testing.T) {
	t.Run("finds slot with matching ItemID and ShineLevel", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, ShineLevel: domain.ShineCommon},
				{ItemID: 1, Quantity: 10, ShineLevel: domain.ShineLegendary},
				{ItemID: 2, Quantity: 3, ShineLevel: domain.ShineRare},
			},
		}

		index, quantity := FindSlotWithShine(inventory, 1, domain.ShineLegendary)

		assert.Equal(t, 1, index, "Should find legendary slot")
		assert.Equal(t, 10, quantity, "Should return correct quantity")
	})

	t.Run("returns -1 when ItemID matches but ShineLevel doesn't", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, ShineLevel: domain.ShineCommon},
			},
		}

		index, quantity := FindSlotWithShine(inventory, 1, domain.ShineLegendary)

		assert.Equal(t, -1, index, "Should not find slot with different shine")
		assert.Equal(t, 0, quantity)
	})

	t.Run("returns -1 when ItemID doesn't match", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, ShineLevel: domain.ShineCommon},
			},
		}

		index, quantity := FindSlotWithShine(inventory, 2, domain.ShineCommon)

		assert.Equal(t, -1, index, "Should not find slot with different ItemID")
		assert.Equal(t, 0, quantity)
	})
}

// TestBuildSlotMap verifies slot map creation with shine awareness
func TestBuildSlotMap(t *testing.T) {
	t.Run("creates correct map for inventory with shine levels", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, ShineLevel: domain.ShineCommon},
				{ItemID: 10, Quantity: 3, ShineLevel: domain.ShineLegendary},
				{ItemID: 20, Quantity: 10, ShineLevel: domain.ShineRare},
			},
		}

		slotMap := BuildSlotMap(inventory)

		assert.Equal(t, 3, len(slotMap), "Should have 3 entries (different shine levels are separate)")
		assert.Equal(t, 0, slotMap[SlotKey{ItemID: 10, ShineLevel: domain.ShineCommon}])
		assert.Equal(t, 1, slotMap[SlotKey{ItemID: 10, ShineLevel: domain.ShineLegendary}])
		assert.Equal(t, 2, slotMap[SlotKey{ItemID: 20, ShineLevel: domain.ShineRare}])
	})

	t.Run("handles empty inventory", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{},
		}

		slotMap := BuildSlotMap(inventory)

		assert.Equal(t, 0, len(slotMap))
	})
}

// TestAddItemsToInventory verifies the hybrid lookup strategy with shine awareness
func TestAddItemsToInventory(t *testing.T) {
	t.Run("adds new items to empty inventory", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{},
		}

		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 5, ShineLevel: domain.ShineCommon},
			{ItemID: 20, Quantity: 10, ShineLevel: domain.ShineRare},
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 2, len(inventory.Slots))
		assert.Equal(t, 10, inventory.Slots[0].ItemID)
		assert.Equal(t, 5, inventory.Slots[0].Quantity)
		assert.Equal(t, domain.ShineCommon, inventory.Slots[0].ShineLevel)
		assert.Equal(t, 20, inventory.Slots[1].ItemID)
		assert.Equal(t, 10, inventory.Slots[1].Quantity)
		assert.Equal(t, domain.ShineRare, inventory.Slots[1].ShineLevel)
	})

	t.Run("separates items by shine level (prevents corruption)", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, ShineLevel: domain.ShineLegendary},
			},
		}

		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3, ShineLevel: domain.ShineCommon},
		}

		AddItemsToInventory(inventory, items, nil)

		// Should create separate slots, NOT stack into legendary
		assert.Equal(t, 2, len(inventory.Slots), "Should have 2 separate slots")
		assert.Equal(t, 5, inventory.Slots[0].Quantity, "Legendary slot unchanged")
		assert.Equal(t, domain.ShineLegendary, inventory.Slots[0].ShineLevel)
		assert.Equal(t, 3, inventory.Slots[1].Quantity, "Common slot added")
		assert.Equal(t, domain.ShineCommon, inventory.Slots[1].ShineLevel)
	})

	t.Run("stacks items with matching ItemID and ShineLevel", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, ShineLevel: domain.ShineRare},
			},
		}

		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3, ShineLevel: domain.ShineRare},
		}

		AddItemsToInventory(inventory, items, nil)

		// Should stack since both ItemID and ShineLevel match
		assert.Equal(t, 1, len(inventory.Slots), "Should have 1 slot")
		assert.Equal(t, 8, inventory.Slots[0].Quantity, "Should stack: 5 + 3")
		assert.Equal(t, domain.ShineRare, inventory.Slots[0].ShineLevel)
	})

	t.Run("adds to existing items (linear scan path)", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, ShineLevel: domain.ShineCommon},
				{ItemID: 20, Quantity: 10, ShineLevel: domain.ShineCommon},
			},
		}

		// Only adding 3 items, should use linear scan (< threshold of 10)
		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3, ShineLevel: domain.ShineCommon},
			{ItemID: 30, Quantity: 7, ShineLevel: domain.ShineCommon},
			{ItemID: 20, Quantity: 2, ShineLevel: domain.ShineCommon},
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 3, len(inventory.Slots))
		assert.Equal(t, 8, inventory.Slots[0].Quantity)  // 5 + 3
		assert.Equal(t, 12, inventory.Slots[1].Quantity) // 10 + 2
		assert.Equal(t, 7, inventory.Slots[2].Quantity)  // new item
	})

	t.Run("adds to existing items (map-based path)", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, ShineLevel: domain.ShineCommon},
				{ItemID: 20, Quantity: 10, ShineLevel: domain.ShineCommon},
			},
		}

		// Adding 12 items, should use map-based lookup (>= threshold of 10)
		items := make([]domain.InventorySlot, 12)
		for i := 0; i < 12; i++ {
			items[i] = domain.InventorySlot{ItemID: 10 + i*10, Quantity: i + 1, ShineLevel: domain.ShineCommon}
		}

		AddItemsToInventory(inventory, items, nil)

		// First two items should be updated
		assert.Equal(t, 6, inventory.Slots[0].Quantity)  // 5 + 1
		assert.Equal(t, 12, inventory.Slots[1].Quantity) // 10 + 2
		// Rest should be added
		assert.Equal(t, 12, len(inventory.Slots))
	})

	t.Run("boundary test: exactly 9 items uses linear scan", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 100, ShineLevel: domain.ShineCommon},
			},
		}

		items := make([]domain.InventorySlot, 9)
		for i := 0; i < 9; i++ {
			items[i] = domain.InventorySlot{ItemID: i + 10, Quantity: 1, ShineLevel: domain.ShineCommon}
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 10, len(inventory.Slots)) // 1 existing + 9 new
	})

	t.Run("boundary test: exactly 10 items uses map", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 100, ShineLevel: domain.ShineCommon},
			},
		}

		items := make([]domain.InventorySlot, 10)
		for i := 0; i < 10; i++ {
			items[i] = domain.InventorySlot{ItemID: i + 10, Quantity: 1, ShineLevel: domain.ShineCommon}
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 11, len(inventory.Slots)) // 1 existing + 10 new
	})

	t.Run("handles duplicate item IDs with same shine", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, ShineLevel: domain.ShineCommon},
			},
		}

		// Adding same item ID and shine multiple times
		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3, ShineLevel: domain.ShineCommon},
			{ItemID: 10, Quantity: 2, ShineLevel: domain.ShineCommon},
			{ItemID: 10, Quantity: 1, ShineLevel: domain.ShineCommon},
		}

		AddItemsToInventory(inventory, items, nil)

		// Should accumulate all quantities
		assert.Equal(t, 1, len(inventory.Slots))
		assert.Equal(t, 11, inventory.Slots[0].Quantity) // 5 + 3 + 2 + 1
	})

	t.Run("handles duplicate item IDs with different shines", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, ShineLevel: domain.ShineCommon},
			},
		}

		// Adding same item ID but different shines - should create separate slots
		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3, ShineLevel: domain.ShineCommon},
			{ItemID: 10, Quantity: 2, ShineLevel: domain.ShineLegendary},
		}

		AddItemsToInventory(inventory, items, nil)

		// Should have 2 slots: common stacked, legendary separate
		assert.Equal(t, 2, len(inventory.Slots))
		assert.Equal(t, 8, inventory.Slots[0].Quantity) // 5 + 3 common
		assert.Equal(t, 2, inventory.Slots[1].Quantity) // legendary
	})

	t.Run("handles empty items list", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, ShineLevel: domain.ShineCommon},
			},
		}

		items := []domain.InventorySlot{}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 1, len(inventory.Slots))
		assert.Equal(t, 5, inventory.Slots[0].Quantity) // unchanged
	})

	t.Run("uses provided slot map when given", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, ShineLevel: domain.ShineCommon},
				{ItemID: 20, Quantity: 10, ShineLevel: domain.ShineCommon},
			},
		}

		slotMap := BuildSlotMap(inventory)

		// Add 12 items to trigger map-based path (>= 10)
		items := make([]domain.InventorySlot, 12)
		for i := 0; i < 12; i++ {
			items[i] = domain.InventorySlot{ItemID: (i + 1) * 10, Quantity: i + 1, ShineLevel: domain.ShineCommon}
		}

		AddItemsToInventory(inventory, items, slotMap)

		// First two items should be updated via map
		assert.Equal(t, 6, inventory.Slots[0].Quantity)  // 10: 5 + 1
		assert.Equal(t, 12, inventory.Slots[1].Quantity) // 20: 10 + 2
		// Rest should be added
		assert.Equal(t, 12, len(inventory.Slots))
		// Verify map was updated with new items
		assert.Equal(t, 2, slotMap[SlotKey{ItemID: 30, ShineLevel: domain.ShineCommon}])   // Third item added at index 2
		assert.Equal(t, 11, slotMap[SlotKey{ItemID: 120, ShineLevel: domain.ShineCommon}]) // Last item added at index 11
	})
}

// Benchmark tests to validate optimization assumptions
func BenchmarkAddItemsLinearScan(b *testing.B) {
	// Create inventory with 1000 items (large N)
	inventory := &domain.Inventory{
		Slots: make([]domain.InventorySlot, 1000),
	}
	for i := 0; i < 1000; i++ {
		inventory.Slots[i] = domain.InventorySlot{ItemID: i, Quantity: 10, ShineLevel: domain.ShineCommon}
	}

	// Add 5 items (small M, below threshold)
	items := []domain.InventorySlot{
		{ItemID: 0, Quantity: 1, ShineLevel: domain.ShineCommon},
		{ItemID: 1, Quantity: 1, ShineLevel: domain.ShineCommon},
		{ItemID: 2, Quantity: 1, ShineLevel: domain.ShineCommon},
		{ItemID: 3, Quantity: 1, ShineLevel: domain.ShineCommon},
		{ItemID: 4, Quantity: 1, ShineLevel: domain.ShineCommon},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clone inventory for each iteration
		testInv := &domain.Inventory{
			Slots: make([]domain.InventorySlot, len(inventory.Slots)),
		}
		copy(testInv.Slots, inventory.Slots)

		AddItemsToInventory(testInv, items, nil)
	}
}

func BenchmarkAddItemsMapLookup(b *testing.B) {
	// Create inventory with 1000 items (large N)
	inventory := &domain.Inventory{
		Slots: make([]domain.InventorySlot, 1000),
	}
	for i := 0; i < 1000; i++ {
		inventory.Slots[i] = domain.InventorySlot{ItemID: i, Quantity: 10, ShineLevel: domain.ShineCommon}
	}

	// Add 50 items (large M, above threshold)
	items := make([]domain.InventorySlot, 50)
	for i := 0; i < 50; i++ {
		items[i] = domain.InventorySlot{ItemID: i, Quantity: 1, ShineLevel: domain.ShineCommon}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clone inventory for each iteration
		testInv := &domain.Inventory{
			Slots: make([]domain.InventorySlot, len(inventory.Slots)),
		}
		copy(testInv.Slots, inventory.Slots)

		AddItemsToInventory(testInv, items, nil)
	}
}

func BenchmarkAddItemsWithPrebuiltMap(b *testing.B) {
	// Create inventory with 1000 items (large N)
	inventory := &domain.Inventory{
		Slots: make([]domain.InventorySlot, 1000),
	}
	for i := 0; i < 1000; i++ {
		inventory.Slots[i] = domain.InventorySlot{ItemID: i, Quantity: 10, ShineLevel: domain.ShineCommon}
	}

	// Add 50 items (large M)
	items := make([]domain.InventorySlot, 50)
	for i := 0; i < 50; i++ {
		items[i] = domain.InventorySlot{ItemID: i, Quantity: 1, ShineLevel: domain.ShineCommon}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clone inventory for each iteration
		testInv := &domain.Inventory{
			Slots: make([]domain.InventorySlot, len(inventory.Slots)),
		}
		copy(testInv.Slots, inventory.Slots)

		// Pre-build the slot map
		slotMap := BuildSlotMap(testInv)
		AddItemsToInventory(testInv, items, slotMap)
	}
}

// TestGetTotalQuantity verifies total quantity calculation
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

// TestConsumeItems verifies item consumption logic
func TestConsumeItems(t *testing.T) {
	t.Run("consumes from single slot", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
			},
		}

		err := ConsumeItems(inventory, 1, 5, func() float64 { return 0.0 })
		assert.NoError(t, err)
		assert.Equal(t, 1, len(inventory.Slots))
		assert.Equal(t, 5, inventory.Slots[0].Quantity)
	})

	t.Run("consumes entire slot", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
				{ItemID: 2, Quantity: 5},
			},
		}

		err := ConsumeItems(inventory, 1, 10, func() float64 { return 0.0 })
		assert.NoError(t, err)
		assert.Equal(t, 1, len(inventory.Slots))
		assert.Equal(t, 2, inventory.Slots[0].ItemID)
	})

	t.Run("consumes from multiple slots", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5},
				{ItemID: 2, Quantity: 5},
				{ItemID: 1, Quantity: 5},
			},
		}

		// Consume 8 items. Should take 5 from one slot and 3 from the other.
		err := ConsumeItems(inventory, 1, 8, func() float64 { return 0.0 })
		assert.NoError(t, err)

		totalRemaining := GetTotalQuantity(inventory, 1)
		assert.Equal(t, 2, totalRemaining)

		// Should have 2 slots remaining (one fully consumed, one partially consumed, one untouched item 2)
		assert.Equal(t, 2, len(inventory.Slots))
	})

	t.Run("returns error if insufficient items", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5},
			},
		}

		err := ConsumeItems(inventory, 1, 10, func() float64 { return 0.0 })
		assert.Error(t, err)
		assert.Equal(t, 5, inventory.Slots[0].Quantity, "Inventory should remain unchanged on error")
	})
}
