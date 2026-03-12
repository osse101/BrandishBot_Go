package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestBuildSlotMap verifies slot map creation with quality awareness
func TestBuildSlotMap(t *testing.T) {
	t.Run("creates correct map for inventory with quality levels", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, QualityLevel: domain.QualityCommon},
				{ItemID: 10, Quantity: 3, QualityLevel: domain.QualityLegendary},
				{ItemID: 20, Quantity: 10, QualityLevel: domain.QualityRare},
			},
		}

		slotMap := BuildSlotMap(inventory)

		assert.Equal(t, 3, len(slotMap), "Should have 3 entries (different quality levels are separate)")
		assert.Equal(t, 0, slotMap[SlotKey{ItemID: 10, QualityLevel: domain.QualityCommon}])
		assert.Equal(t, 1, slotMap[SlotKey{ItemID: 10, QualityLevel: domain.QualityLegendary}])
		assert.Equal(t, 2, slotMap[SlotKey{ItemID: 20, QualityLevel: domain.QualityRare}])
	})

	t.Run("handles empty inventory", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{},
		}

		slotMap := BuildSlotMap(inventory)

		assert.Equal(t, 0, len(slotMap))
	})
}

// TestAddItemsToInventory verifies the hybrid lookup strategy with quality awareness
func TestAddItemsToInventory(t *testing.T) {
	t.Run("adds new items to empty inventory", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{},
		}

		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 5, QualityLevel: domain.QualityCommon},
			{ItemID: 20, Quantity: 10, QualityLevel: domain.QualityRare},
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 2, len(inventory.Slots))
		assert.Equal(t, 10, inventory.Slots[0].ItemID)
		assert.Equal(t, 5, inventory.Slots[0].Quantity)
		assert.Equal(t, domain.QualityCommon, inventory.Slots[0].QualityLevel)
		assert.Equal(t, 20, inventory.Slots[1].ItemID)
		assert.Equal(t, 10, inventory.Slots[1].Quantity)
		assert.Equal(t, domain.QualityRare, inventory.Slots[1].QualityLevel)
	})

	t.Run("separates items by quality level (prevents corruption)", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, QualityLevel: domain.QualityLegendary},
			},
		}

		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3, QualityLevel: domain.QualityCommon},
		}

		AddItemsToInventory(inventory, items, nil)

		// Should create separate slots, NOT stack into legendary
		assert.Equal(t, 2, len(inventory.Slots), "Should have 2 separate slots")
		assert.Equal(t, 5, inventory.Slots[0].Quantity, "Legendary slot unchanged")
		assert.Equal(t, domain.QualityLegendary, inventory.Slots[0].QualityLevel)
		assert.Equal(t, 3, inventory.Slots[1].Quantity, "Common slot added")
		assert.Equal(t, domain.QualityCommon, inventory.Slots[1].QualityLevel)
	})

	t.Run("stacks items with matching ItemID and QualityLevel", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, QualityLevel: domain.QualityRare},
			},
		}

		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3, QualityLevel: domain.QualityRare},
		}

		AddItemsToInventory(inventory, items, nil)

		// Should stack since both ItemID and QualityLevel match
		assert.Equal(t, 1, len(inventory.Slots), "Should have 1 slot")
		assert.Equal(t, 8, inventory.Slots[0].Quantity, "Should stack: 5 + 3")
		assert.Equal(t, domain.QualityRare, inventory.Slots[0].QualityLevel)
	})

	t.Run("adds to existing items (linear scan path)", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, QualityLevel: domain.QualityCommon},
				{ItemID: 20, Quantity: 10, QualityLevel: domain.QualityCommon},
			},
		}

		// Only adding 3 items, should use linear scan (< threshold of 10)
		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3, QualityLevel: domain.QualityCommon},
			{ItemID: 30, Quantity: 7, QualityLevel: domain.QualityCommon},
			{ItemID: 20, Quantity: 2, QualityLevel: domain.QualityCommon},
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
				{ItemID: 10, Quantity: 5, QualityLevel: domain.QualityCommon},
				{ItemID: 20, Quantity: 10, QualityLevel: domain.QualityCommon},
			},
		}

		// Adding 12 items, should use map-based lookup (>= threshold of 10)
		items := make([]domain.InventorySlot, 12)
		for i := 0; i < 12; i++ {
			items[i] = domain.InventorySlot{ItemID: 10 + i*10, Quantity: i + 1, QualityLevel: domain.QualityCommon}
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
				{ItemID: 1, Quantity: 100, QualityLevel: domain.QualityCommon},
			},
		}

		items := make([]domain.InventorySlot, 9)
		for i := 0; i < 9; i++ {
			items[i] = domain.InventorySlot{ItemID: i + 10, Quantity: 1, QualityLevel: domain.QualityCommon}
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 10, len(inventory.Slots)) // 1 existing + 9 new
	})

	t.Run("boundary test: exactly 10 items uses map", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 100, QualityLevel: domain.QualityCommon},
			},
		}

		items := make([]domain.InventorySlot, 10)
		for i := 0; i < 10; i++ {
			items[i] = domain.InventorySlot{ItemID: i + 10, Quantity: 1, QualityLevel: domain.QualityCommon}
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 11, len(inventory.Slots)) // 1 existing + 10 new
	})

	t.Run("handles duplicate item IDs with same quality", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, QualityLevel: domain.QualityCommon},
			},
		}

		// Adding same item ID and quality multiple times
		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3, QualityLevel: domain.QualityCommon},
			{ItemID: 10, Quantity: 2, QualityLevel: domain.QualityCommon},
			{ItemID: 10, Quantity: 1, QualityLevel: domain.QualityCommon},
		}

		AddItemsToInventory(inventory, items, nil)

		// Should accumulate all quantities
		assert.Equal(t, 1, len(inventory.Slots))
		assert.Equal(t, 11, inventory.Slots[0].Quantity) // 5 + 3 + 2 + 1
	})

	t.Run("handles duplicate item IDs with different qualities", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, QualityLevel: domain.QualityCommon},
			},
		}

		// Adding same item ID but different qualities - should create separate slots
		items := []domain.InventorySlot{
			{ItemID: 10, Quantity: 3, QualityLevel: domain.QualityCommon},
			{ItemID: 10, Quantity: 2, QualityLevel: domain.QualityLegendary},
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
				{ItemID: 10, Quantity: 5, QualityLevel: domain.QualityCommon},
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
				{ItemID: 10, Quantity: 5, QualityLevel: domain.QualityCommon},
				{ItemID: 20, Quantity: 10, QualityLevel: domain.QualityCommon},
			},
		}

		slotMap := BuildSlotMap(inventory)

		// Add 60 items to trigger map-based path (>= 50)
		items := make([]domain.InventorySlot, 60)
		for i := 0; i < 60; i++ {
			items[i] = domain.InventorySlot{ItemID: (i + 1) * 10, Quantity: i + 1, QualityLevel: domain.QualityCommon}
		}

		AddItemsToInventory(inventory, items, slotMap)

		// First two items should be updated via map
		assert.Equal(t, 6, inventory.Slots[0].Quantity)  // 10: 5 + 1
		assert.Equal(t, 12, inventory.Slots[1].Quantity) // 20: 10 + 2
		// Rest should be added
		assert.Equal(t, 60, len(inventory.Slots))
		// Verify map was updated with new items
		assert.Equal(t, 2, slotMap[SlotKey{ItemID: 30, QualityLevel: domain.QualityCommon}])   // Third item added at index 2
		assert.Equal(t, 59, slotMap[SlotKey{ItemID: 600, QualityLevel: domain.QualityCommon}]) // Last item added at index 59
	})
}

// Benchmark tests to validate optimization assumptions
func BenchmarkAddItemsLinearScan(b *testing.B) {
	// Create inventory with 1000 items (large N)
	inventory := &domain.Inventory{
		Slots: make([]domain.InventorySlot, 1000),
	}
	for i := 0; i < 1000; i++ {
		inventory.Slots[i] = domain.InventorySlot{ItemID: i, Quantity: 10, QualityLevel: domain.QualityCommon}
	}

	// Add 5 items (small M, below threshold)
	items := []domain.InventorySlot{
		{ItemID: 0, Quantity: 1, QualityLevel: domain.QualityCommon},
		{ItemID: 1, Quantity: 1, QualityLevel: domain.QualityCommon},
		{ItemID: 2, Quantity: 1, QualityLevel: domain.QualityCommon},
		{ItemID: 3, Quantity: 1, QualityLevel: domain.QualityCommon},
		{ItemID: 4, Quantity: 1, QualityLevel: domain.QualityCommon},
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
		inventory.Slots[i] = domain.InventorySlot{ItemID: i, Quantity: 10, QualityLevel: domain.QualityCommon}
	}

	// Add 50 items (large M, above threshold)
	items := make([]domain.InventorySlot, 50)
	for i := 0; i < 50; i++ {
		items[i] = domain.InventorySlot{ItemID: i, Quantity: 1, QualityLevel: domain.QualityCommon}
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
		inventory.Slots[i] = domain.InventorySlot{ItemID: i, Quantity: 10, QualityLevel: domain.QualityCommon}
	}

	// Add 50 items (large M)
	items := make([]domain.InventorySlot, 50)
	for i := 0; i < 50; i++ {
		items[i] = domain.InventorySlot{ItemID: i, Quantity: 1, QualityLevel: domain.QualityCommon}
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
