package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestFindSlot(t *testing.T) {
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 5},
			{ItemID: 2, Quantity: 10},
			{ItemID: 3, Quantity: 1},
		},
	}

	tests := []struct {
		name        string
		itemID      int
		expectedIdx int
		expectedQty int
	}{
		{"item exists at index 0", 1, 0, 5},
		{"item exists at index 1", 2, 1, 10},
		{"item does not exist", 4, -1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, qty := FindSlot(inventory, tt.itemID)
			assert.Equal(t, tt.expectedIdx, idx)
			assert.Equal(t, tt.expectedQty, qty)
		})
	}
}

// Additional test to cover larger inventories and multiple slots with the same item
func TestFindSlot_RealWorldScenarios(t *testing.T) {
	t.Run("returns first slot when multiple slots have same item", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5},
				{ItemID: 1, Quantity: 10}, // Duplicate item ID (e.g., different qualities in real system)
			},
		}

		idx, qty := FindSlot(inventory, 1)
		assert.Equal(t, 0, idx)
		assert.Equal(t, 5, qty)
	})

	t.Run("handles empty inventory", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{},
		}

		idx, qty := FindSlot(inventory, 1)
		assert.Equal(t, -1, idx)
		assert.Equal(t, 0, qty)
	})

	t.Run("handles large inventory efficiently", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: make([]domain.InventorySlot, 1000),
		}

		// Fill with dummy items
		for i := 0; i < 999; i++ {
			inventory.Slots[i] = domain.InventorySlot{ItemID: i + 10, Quantity: 1}
		}
		// Put target item at the very end
		inventory.Slots[999] = domain.InventorySlot{ItemID: 1, Quantity: 42}

		idx, qty := FindSlot(inventory, 1)
		assert.Equal(t, 999, idx)
		assert.Equal(t, 42, qty)
	})
}

func TestFindSlotWithQuality(t *testing.T) {
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
			{ItemID: 1, Quantity: 2, QualityLevel: domain.QualityLegendary},
			{ItemID: 2, Quantity: 10, QualityLevel: domain.QualityCommon},
		},
	}

	t.Run("finds item with specific quality", func(t *testing.T) {
		idx, qty := FindSlotWithQuality(inventory, 1, domain.QualityLegendary)
		assert.Equal(t, 1, idx)
		assert.Equal(t, 2, qty)
	})

	t.Run("returns -1 if item exists but quality does not match", func(t *testing.T) {
		idx, qty := FindSlotWithQuality(inventory, 2, domain.QualityLegendary)
		assert.Equal(t, -1, idx)
		assert.Equal(t, 0, qty)
	})

	t.Run("returns -1 if item does not exist", func(t *testing.T) {
		idx, qty := FindSlotWithQuality(inventory, 3, domain.QualityCommon)
		assert.Equal(t, -1, idx)
		assert.Equal(t, 0, qty)
	})
}

func TestFindRandomSlot(t *testing.T) {
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
			{ItemID: 1, Quantity: 2, QualityLevel: domain.QualityLegendary},
			{ItemID: 2, Quantity: 10, QualityLevel: domain.QualityCommon},
		},
	}

	t.Run("returns -1 if item not found", func(t *testing.T) {
		idx, qty := FindRandomSlot(inventory, 99, func() float64 { return 0.5 })
		assert.Equal(t, -1, idx)
		assert.Equal(t, 0, qty)
	})

	t.Run("returns exactly the single slot if only one matches", func(t *testing.T) {
		idx, qty := FindRandomSlot(inventory, 2, func() float64 { return 0.5 })
		assert.Equal(t, 2, idx)
		assert.Equal(t, 10, qty)
	})

	t.Run("uses RNG to select among multiple matching slots", func(t *testing.T) {
		// Mock RNG to return 0.0 (should pick first match: index 0)
		idx1, _ := FindRandomSlot(inventory, 1, func() float64 { return 0.0 })
		assert.Equal(t, 0, idx1)

		// Mock RNG to return 0.99 (should pick second match: index 1)
		idx2, _ := FindRandomSlot(inventory, 1, func() float64 { return 0.99 })
		assert.Equal(t, 1, idx2)
	})

	t.Run("handles RNG returning exactly 1.0 (safety boundary test)", func(t *testing.T) {
		// Should not panic, should clamp to last element
		idx, _ := FindRandomSlot(inventory, 1, func() float64 { return 1.0 })
		assert.Equal(t, 1, idx) // The last matching slot
	})
}
