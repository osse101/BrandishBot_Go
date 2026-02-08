package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestCalculateAverageShine verifies shine averaging from consumed materials
func TestCalculateAverageShine(t *testing.T) {
	t.Run("all same shine returns that shine", func(t *testing.T) {
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 10, ShineLevel: domain.ShineCommon},
			{ItemID: 2, Quantity: 5, ShineLevel: domain.ShineCommon},
		}

		result := CalculateAverageShine(materials)
		assert.Equal(t, domain.ShineCommon, result)
	})

	t.Run("mixes common and legendary", func(t *testing.T) {
		// 5x COMMON (value 3) + 3x LEGENDARY (value 7)
		// = (5*3 + 3*7) / 8 = (15 + 21) / 8 = 36 / 8 = 4.5 → rounds to RARE (value 5)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 5, ShineLevel: domain.ShineCommon},
			{ItemID: 2, Quantity: 3, ShineLevel: domain.ShineLegendary},
		}

		result := CalculateAverageShine(materials)
		assert.Equal(t, domain.ShineRare, result, "Should average to RARE")
	})

	t.Run("heavy common bias stays common", func(t *testing.T) {
		// 10x COMMON + 1x LEGENDARY
		// = (10*3 + 1*7) / 11 = 37 / 11 = 3.36 → rounds to COMMON (value 3)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 10, ShineLevel: domain.ShineCommon},
			{ItemID: 2, Quantity: 1, ShineLevel: domain.ShineLegendary},
		}

		result := CalculateAverageShine(materials)
		assert.Equal(t, domain.ShineCommon, result, "Should stay COMMON with heavy bias")
	})

	t.Run("heavy legendary bias approaches legendary", func(t *testing.T) {
		// 1x COMMON + 10x LEGENDARY
		// = (1*3 + 10*7) / 11 = 73 / 11 = 6.64 → rounds to LEGENDARY (value 7)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 1, ShineLevel: domain.ShineCommon},
			{ItemID: 2, Quantity: 10, ShineLevel: domain.ShineLegendary},
		}

		result := CalculateAverageShine(materials)
		assert.Equal(t, domain.ShineLegendary, result, "Should round to LEGENDARY")
	})

	t.Run("mix of uncommon and rare", func(t *testing.T) {
		// 5x UNCOMMON (value 4) + 5x RARE (value 5)
		// = (5*4 + 5*5) / 10 = 45 / 10 = 4.5 → rounds to RARE (value 5)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 5, ShineLevel: domain.ShineUncommon},
			{ItemID: 2, Quantity: 5, ShineLevel: domain.ShineRare},
		}

		result := CalculateAverageShine(materials)
		assert.Equal(t, domain.ShineRare, result)
	})

	t.Run("cursed and junk materials", func(t *testing.T) {
		// 3x CURSED (value 0) + 3x JUNK (value 1)
		// = (3*0 + 3*1) / 6 = 3 / 6 = 0.5 → rounds to JUNK (value 1)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 3, ShineLevel: domain.ShineCursed},
			{ItemID: 2, Quantity: 3, ShineLevel: domain.ShineJunk},
		}

		result := CalculateAverageShine(materials)
		assert.Equal(t, domain.ShineJunk, result)
	})

	t.Run("empty materials returns common", func(t *testing.T) {
		materials := []domain.InventorySlot{}

		result := CalculateAverageShine(materials)
		assert.Equal(t, domain.ShineCommon, result, "Empty should default to COMMON")
	})

	t.Run("all materials with zero quantity returns common", func(t *testing.T) {
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 0, ShineLevel: domain.ShineLegendary},
		}

		result := CalculateAverageShine(materials)
		assert.Equal(t, domain.ShineCommon, result, "Zero quantity should default to COMMON")
	})

	t.Run("three-way mix", func(t *testing.T) {
		// 2x COMMON (value 3) + 3x RARE (value 5) + 1x LEGENDARY (value 7)
		// = (2*3 + 3*5 + 1*7) / 6 = (6 + 15 + 7) / 6 = 28 / 6 = 4.67 → rounds to RARE (value 5)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 2, ShineLevel: domain.ShineCommon},
			{ItemID: 2, Quantity: 3, ShineLevel: domain.ShineRare},
			{ItemID: 3, Quantity: 1, ShineLevel: domain.ShineLegendary},
		}

		result := CalculateAverageShine(materials)
		assert.Equal(t, domain.ShineRare, result)
	})

	t.Run("exact boundary between shines", func(t *testing.T) {
		// 1x COMMON (value 3) + 1x UNCOMMON (value 4)
		// = (1*3 + 1*4) / 2 = 7 / 2 = 3.5 → rounds to UNCOMMON (value 4)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 1, ShineLevel: domain.ShineCommon},
			{ItemID: 2, Quantity: 1, ShineLevel: domain.ShineUncommon},
		}

		result := CalculateAverageShine(materials)
		assert.Equal(t, domain.ShineUncommon, result, "Should round up at 0.5 boundary")
	})
}

// TestConsumeItemsWithTracking verifies consumption tracking with shine levels
func TestConsumeItemsWithTracking(t *testing.T) {
	t.Run("tracks consumed items with shine", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10, ShineLevel: domain.ShineCommon},
			},
		}

		consumed, err := ConsumeItemsWithTracking(inventory, 1, 5, func() float64 { return 0.0 })

		assert.NoError(t, err)
		assert.Equal(t, 1, len(consumed))
		assert.Equal(t, 1, consumed[0].ItemID)
		assert.Equal(t, 5, consumed[0].Quantity)
		assert.Equal(t, domain.ShineCommon, consumed[0].ShineLevel)
		assert.Equal(t, 5, inventory.Slots[0].Quantity, "Inventory should be reduced")
	})

	t.Run("tracks consumption from multiple slots with different shines", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, ShineLevel: domain.ShineCommon},
				{ItemID: 2, Quantity: 10, ShineLevel: domain.ShineRare},
				{ItemID: 1, Quantity: 3, ShineLevel: domain.ShineLegendary},
			},
		}

		// Consume 7 items of ItemID 1 (should take from both common and legendary slots)
		consumed, err := ConsumeItemsWithTracking(inventory, 1, 7, func() float64 { return 0.0 })

		assert.NoError(t, err)
		assert.Equal(t, 2, len(consumed), "Should consume from 2 different slots")

		// Verify total consumed
		totalConsumed := 0
		for _, slot := range consumed {
			totalConsumed += slot.Quantity
		}
		assert.Equal(t, 7, totalConsumed)

		// Verify shine levels are tracked
		hasCommon := false
		hasLegendary := false
		for _, slot := range consumed {
			if slot.ShineLevel == domain.ShineCommon {
				hasCommon = true
			}
			if slot.ShineLevel == domain.ShineLegendary {
				hasLegendary = true
			}
		}
		assert.True(t, hasCommon, "Should track common shine consumption")
		assert.True(t, hasLegendary, "Should track legendary shine consumption")

		// Verify inventory reduced correctly
		totalRemaining := GetTotalQuantity(inventory, 1)
		assert.Equal(t, 1, totalRemaining, "Should have 1 item remaining (8 - 7)")
	})

	t.Run("returns error on insufficient items", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, ShineLevel: domain.ShineCommon},
			},
		}

		consumed, err := ConsumeItemsWithTracking(inventory, 1, 10, func() float64 { return 0.0 })

		assert.Error(t, err)
		assert.Nil(t, consumed)
		assert.Equal(t, 5, inventory.Slots[0].Quantity, "Inventory should be unchanged on error")
	})

	t.Run("consumes entire slot removes it", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, ShineLevel: domain.ShineLegendary},
				{ItemID: 2, Quantity: 10, ShineLevel: domain.ShineCommon},
			},
		}

		consumed, err := ConsumeItemsWithTracking(inventory, 1, 5, func() float64 { return 0.0 })

		assert.NoError(t, err)
		assert.Equal(t, 1, len(consumed))
		assert.Equal(t, domain.ShineLegendary, consumed[0].ShineLevel)
		assert.Equal(t, 1, len(inventory.Slots), "Fully consumed slot should be removed")
		assert.Equal(t, 2, inventory.Slots[0].ItemID, "Remaining slot should be ItemID 2")
	})
}
