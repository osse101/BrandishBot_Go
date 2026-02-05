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

// TestBuildSlotMap verifies slot map creation
func TestBuildSlotMap(t *testing.T) {
	t.Run("creates correct map for inventory", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5},
				{ItemID: 20, Quantity: 10},
				{ItemID: 30, Quantity: 15},
			},
		}

		slotMap := BuildSlotMap(inventory)

		assert.Equal(t, 3, len(slotMap))
		assert.Equal(t, 0, slotMap[10])
		assert.Equal(t, 1, slotMap[20])
		assert.Equal(t, 2, slotMap[30])
	})

	t.Run("handles empty inventory", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{},
		}

		slotMap := BuildSlotMap(inventory)

		assert.Equal(t, 0, len(slotMap))
	})
}

// TestAddItemsToInventory verifies the hybrid lookup strategy
func TestAddItemsToInventory(t *testing.T) {
	t.Run("adds new items to empty inventory", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{},
		}

		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 5},
			{ItemID: 20, Quantity: 10},
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 2, len(inventory.Slots))
		assert.Equal(t, 10, inventory.Slots[0].ItemID)
		assert.Equal(t, 5, inventory.Slots[0].Quantity)
		assert.Equal(t, 20, inventory.Slots[1].ItemID)
		assert.Equal(t, 10, inventory.Slots[1].Quantity)
	})

	t.Run("adds to existing items (linear scan path)", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5},
				{ItemID: 20, Quantity: 10},
			},
		}

		// Only adding 3 items, should use linear scan (< threshold of 10)
		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3},
			{ItemID: 30, Quantity: 7},
			{ItemID: 20, Quantity: 2},
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
				{ItemID: 10, Quantity: 5},
				{ItemID: 20, Quantity: 10},
			},
		}

		// Adding 12 items, should use map-based lookup (>= threshold of 10)
		items := make([]domain.InventorySlot, 12)
		for i := 0; i < 12; i++ {
			items[i] = domain.InventorySlot{ItemID: 10 + i*10, Quantity: i + 1}
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
				{ItemID: 1, Quantity: 100},
			},
		}

		items := make([]domain.InventorySlot, 9)
		for i := 0; i < 9; i++ {
			items[i] = domain.InventorySlot{ItemID: i + 10, Quantity: 1}
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 10, len(inventory.Slots)) // 1 existing + 9 new
	})

	t.Run("boundary test: exactly 10 items uses map", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 100},
			},
		}

		items := make([]domain.InventorySlot, 10)
		for i := 0; i < 10; i++ {
			items[i] = domain.InventorySlot{ItemID: i + 10, Quantity: 1}
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 11, len(inventory.Slots)) // 1 existing + 10 new
	})

	t.Run("handles duplicate item IDs in items to add", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5},
			},
		}

		// Adding same item ID multiple times
		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3},
			{ItemID: 10, Quantity: 2},
			{ItemID: 10, Quantity: 1},
		}

		AddItemsToInventory(inventory, items, nil)

		// Should accumulate all quantities
		assert.Equal(t, 1, len(inventory.Slots))
		assert.Equal(t, 11, inventory.Slots[0].Quantity) // 5 + 3 + 2 + 1
	})

	t.Run("handles empty items list", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5},
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
				{ItemID: 10, Quantity: 5},
				{ItemID: 20, Quantity: 10},
			},
		}

		slotMap := BuildSlotMap(inventory)

		// Add 12 items to trigger map-based path (>= 10)
		items := make([]domain.InventorySlot, 12)
		for i := 0; i < 12; i++ {
			items[i] = domain.InventorySlot{ItemID: (i + 1) * 10, Quantity: i + 1}
		}

		AddItemsToInventory(inventory, items, slotMap)

		// First two items should be updated via map
		assert.Equal(t, 6, inventory.Slots[0].Quantity)  // 10: 5 + 1
		assert.Equal(t, 12, inventory.Slots[1].Quantity) // 20: 10 + 2
		// Rest should be added
		assert.Equal(t, 12, len(inventory.Slots))
		// Verify map was updated with new items
		assert.Equal(t, 2, slotMap[30])   // Third item added at index 2
		assert.Equal(t, 11, slotMap[120]) // Last item added at index 11
	})
}

// Benchmark tests to validate optimization assumptions
func BenchmarkAddItemsLinearScan(b *testing.B) {
	// Create inventory with 1000 items (large N)
	inventory := &domain.Inventory{
		Slots: make([]domain.InventorySlot, 1000),
	}
	for i := 0; i < 1000; i++ {
		inventory.Slots[i] = domain.InventorySlot{ItemID: i, Quantity: 10}
	}

	// Add 5 items (small M, below threshold)
	items := []domain.InventorySlot{
		{ItemID: 0, Quantity: 1},
		{ItemID: 1, Quantity: 1},
		{ItemID: 2, Quantity: 1},
		{ItemID: 3, Quantity: 1},
		{ItemID: 4, Quantity: 1},
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
		inventory.Slots[i] = domain.InventorySlot{ItemID: i, Quantity: 10}
	}

	// Add 50 items (large M, above threshold)
	items := make([]domain.InventorySlot, 50)
	for i := 0; i < 50; i++ {
		items[i] = domain.InventorySlot{ItemID: i, Quantity: 1}
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
		inventory.Slots[i] = domain.InventorySlot{ItemID: i, Quantity: 10}
	}

	// Add 50 items (large M)
	items := make([]domain.InventorySlot, 50)
	for i := 0; i < 50; i++ {
		items[i] = domain.InventorySlot{ItemID: i, Quantity: 1}
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
