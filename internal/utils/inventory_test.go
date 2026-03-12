package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

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
