package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestAddItemsToInventory verifies the AddItemsToInventory function
func TestAddItemsToInventory(t *testing.T) {
	t.Run("adds single new item", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{},
		}

		items := []domain.InventorySlot{
			{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 1, len(inventory.Slots))
		assert.Equal(t, 1, inventory.Slots[0].ItemID)
		assert.Equal(t, 5, inventory.Slots[0].Quantity)
		assert.Equal(t, domain.QualityCommon, inventory.Slots[0].QualityLevel)
	})

	t.Run("updates existing item quantity", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
			},
		}

		items := []domain.InventorySlot{
			{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 1, len(inventory.Slots))
		assert.Equal(t, 15, inventory.Slots[0].Quantity)
	})

	t.Run("adds multiple new items", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{},
		}

		items := []domain.InventorySlot{
			{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
			{ItemID: 2, Quantity: 10, QualityLevel: domain.QualityUncommon},
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 2, len(inventory.Slots))
		assert.Equal(t, 1, inventory.Slots[0].ItemID)
		assert.Equal(t, 5, inventory.Slots[0].Quantity)
		assert.Equal(t, 2, inventory.Slots[1].ItemID)
		assert.Equal(t, 10, inventory.Slots[1].Quantity)
	})

	t.Run("updates existing and adds new items", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
			},
		}

		items := []domain.InventorySlot{
			{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
			{ItemID: 2, Quantity: 10, QualityLevel: domain.QualityUncommon},
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 2, len(inventory.Slots))
		assert.Equal(t, 15, inventory.Slots[0].Quantity)
		assert.Equal(t, 2, inventory.Slots[1].ItemID)
	})

	t.Run("differentiates by quality level", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
			},
		}

		items := []domain.InventorySlot{
			{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityRare},
		}

		AddItemsToInventory(inventory, items, nil)

		assert.Equal(t, 2, len(inventory.Slots))
		assert.Equal(t, 5, inventory.Slots[0].Quantity)
		assert.Equal(t, domain.QualityCommon, inventory.Slots[0].QualityLevel)
		assert.Equal(t, 10, inventory.Slots[1].Quantity)
		assert.Equal(t, domain.QualityRare, inventory.Slots[1].QualityLevel)
	})

	t.Run("handles large number of items (map threshold logic)", func(t *testing.T) {
		// Use a large number to trigger the map-based optimization threshold
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 10, Quantity: 5, QualityLevel: domain.QualityCommon},
				{ItemID: 20, Quantity: 10, QualityLevel: domain.QualityCommon},
			},
		}

		items := make([]domain.InventorySlot, 60)
		for i := 0; i < 60; i++ {
			items[i] = domain.InventorySlot{ItemID: 10 + i*10, Quantity: i + 1, QualityLevel: domain.QualityCommon}
		}

		AddItemsToInventory(inventory, items, nil)

		// First two items should be updated
		assert.Equal(t, 6, inventory.Slots[0].Quantity)  // 5 + 1
		assert.Equal(t, 12, inventory.Slots[1].Quantity) // 10 + 2
		// Rest should be added
		assert.Equal(t, 60, len(inventory.Slots))
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
