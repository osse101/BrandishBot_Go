package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

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

	t.Run("handles RNG returning 1.0 (safety test)", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
				{ItemID: 1, Quantity: 10},
			},
		}

		// Should not panic even if RNG returns 1.0
		err := ConsumeItems(inventory, 1, 5, func() float64 { return 1.0 })
		assert.NoError(t, err)
	})
}
