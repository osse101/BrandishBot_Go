package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

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
