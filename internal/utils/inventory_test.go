package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestBuildSlotMap(t *testing.T) {
	t.Run("builds map correctly for single item", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
			},
		}

		slotMap := BuildSlotMap(inventory)

		assert.Equal(t, 1, len(slotMap))
		idx, exists := slotMap[SlotKey{ItemID: 1, QualityLevel: domain.QualityCommon}]
		assert.True(t, exists)
		assert.Equal(t, 0, idx)
	})

	t.Run("differentiates items by quality level", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
				{ItemID: 1, Quantity: 2, QualityLevel: domain.QualityLegendary},
				{ItemID: 2, Quantity: 10, QualityLevel: domain.QualityCommon},
			},
		}

		slotMap := BuildSlotMap(inventory)

		assert.Equal(t, 3, len(slotMap))
		assert.Equal(t, 0, slotMap[SlotKey{ItemID: 1, QualityLevel: domain.QualityCommon}])
		assert.Equal(t, 1, slotMap[SlotKey{ItemID: 1, QualityLevel: domain.QualityLegendary}])
		assert.Equal(t, 2, slotMap[SlotKey{ItemID: 2, QualityLevel: domain.QualityCommon}])
	})

	t.Run("handles empty inventory", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{},
		}

		slotMap := BuildSlotMap(inventory)
		assert.Equal(t, 0, len(slotMap))
	})
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
