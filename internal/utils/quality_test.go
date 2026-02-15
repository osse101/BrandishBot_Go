package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestCalculateAverageQuality verifies quality averaging from consumed materials
func TestCalculateAverageQuality(t *testing.T) {
	t.Run("all same quality returns that quality", func(t *testing.T) {
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
			{ItemID: 2, Quantity: 5, QualityLevel: domain.QualityCommon},
		}

		result := CalculateAverageQuality(materials)
		assert.Equal(t, domain.QualityCommon, result)
	})

	t.Run("mixes common and legendary", func(t *testing.T) {
		// 5x COMMON (value 3) + 3x LEGENDARY (value 7)
		// = (5*3 + 3*7) / 8 = (15 + 21) / 8 = 36 / 8 = 4.5 → rounds to RARE (value 5)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
			{ItemID: 2, Quantity: 3, QualityLevel: domain.QualityLegendary},
		}

		result := CalculateAverageQuality(materials)
		assert.Equal(t, domain.QualityRare, result, "Should average to RARE")
	})

	t.Run("heavy common bias stays common", func(t *testing.T) {
		// 10x COMMON + 1x LEGENDARY
		// = (10*3 + 1*7) / 11 = 37 / 11 = 3.36 → rounds to COMMON (value 3)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
			{ItemID: 2, Quantity: 1, QualityLevel: domain.QualityLegendary},
		}

		result := CalculateAverageQuality(materials)
		assert.Equal(t, domain.QualityCommon, result, "Should stay COMMON with heavy bias")
	})

	t.Run("heavy legendary bias approaches legendary", func(t *testing.T) {
		// 1x COMMON + 10x LEGENDARY
		// = (1*3 + 10*7) / 11 = 73 / 11 = 6.64 → rounds to LEGENDARY (value 7)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 1, QualityLevel: domain.QualityCommon},
			{ItemID: 2, Quantity: 10, QualityLevel: domain.QualityLegendary},
		}

		result := CalculateAverageQuality(materials)
		assert.Equal(t, domain.QualityLegendary, result, "Should round to LEGENDARY")
	})

	t.Run("mix of uncommon and rare", func(t *testing.T) {
		// 5x UNCOMMON (value 4) + 5x RARE (value 5)
		// = (5*4 + 5*5) / 10 = 45 / 10 = 4.5 → rounds to RARE (value 5)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityUncommon},
			{ItemID: 2, Quantity: 5, QualityLevel: domain.QualityRare},
		}

		result := CalculateAverageQuality(materials)
		assert.Equal(t, domain.QualityRare, result)
	})

	t.Run("cursed and junk materials", func(t *testing.T) {
		// 3x CURSED (value 0) + 3x JUNK (value 1)
		// = (3*0 + 3*1) / 6 = 3 / 6 = 0.5 → rounds to JUNK (value 1)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 3, QualityLevel: domain.QualityCursed},
			{ItemID: 2, Quantity: 3, QualityLevel: domain.QualityJunk},
		}

		result := CalculateAverageQuality(materials)
		assert.Equal(t, domain.QualityJunk, result)
	})

	t.Run("empty materials returns common", func(t *testing.T) {
		materials := []domain.InventorySlot{}

		result := CalculateAverageQuality(materials)
		assert.Equal(t, domain.QualityCommon, result, "Empty should default to COMMON")
	})

	t.Run("all materials with zero quantity returns common", func(t *testing.T) {
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 0, QualityLevel: domain.QualityLegendary},
		}

		result := CalculateAverageQuality(materials)
		assert.Equal(t, domain.QualityCommon, result, "Zero quantity should default to COMMON")
	})

	t.Run("three-way mix", func(t *testing.T) {
		// 2x COMMON (value 3) + 3x RARE (value 5) + 1x LEGENDARY (value 7)
		// = (2*3 + 3*5 + 1*7) / 6 = (6 + 15 + 7) / 6 = 28 / 6 = 4.67 → rounds to RARE (value 5)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 2, QualityLevel: domain.QualityCommon},
			{ItemID: 2, Quantity: 3, QualityLevel: domain.QualityRare},
			{ItemID: 3, Quantity: 1, QualityLevel: domain.QualityLegendary},
		}

		result := CalculateAverageQuality(materials)
		assert.Equal(t, domain.QualityRare, result)
	})

	t.Run("exact boundary between qualities", func(t *testing.T) {
		// 1x COMMON (value 3) + 1x UNCOMMON (value 4)
		// = (1*3 + 1*4) / 2 = 7 / 2 = 3.5 → rounds to UNCOMMON (value 4)
		materials := []domain.InventorySlot{
			{ItemID: 1, Quantity: 1, QualityLevel: domain.QualityCommon},
			{ItemID: 2, Quantity: 1, QualityLevel: domain.QualityUncommon},
		}

		result := CalculateAverageQuality(materials)
		assert.Equal(t, domain.QualityUncommon, result, "Should round up at 0.5 boundary")
	})
}

// TestConsumeItemsWithTracking verifies consumption tracking with quality levels
func TestConsumeItemsWithTracking(t *testing.T) {
	t.Run("tracks consumed items with quality", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
			},
		}

		consumed, err := ConsumeItemsWithTracking(inventory, 1, 5, func() float64 { return 0.0 })

		assert.NoError(t, err)
		assert.Equal(t, 1, len(consumed))
		assert.Equal(t, 1, consumed[0].ItemID)
		assert.Equal(t, 5, consumed[0].Quantity)
		assert.Equal(t, domain.QualityCommon, consumed[0].QualityLevel)
		assert.Equal(t, 5, inventory.Slots[0].Quantity, "Inventory should be reduced")
	})

	t.Run("tracks consumption from multiple slots with different qualities", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
				{ItemID: 2, Quantity: 10, QualityLevel: domain.QualityRare},
				{ItemID: 1, Quantity: 3, QualityLevel: domain.QualityLegendary},
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

		// Verify quality levels are tracked
		hasCommon := false
		hasLegendary := false
		for _, slot := range consumed {
			if slot.QualityLevel == domain.QualityCommon {
				hasCommon = true
			}
			if slot.QualityLevel == domain.QualityLegendary {
				hasLegendary = true
			}
		}
		assert.True(t, hasCommon, "Should track common quality consumption")
		assert.True(t, hasLegendary, "Should track legendary quality consumption")

		// Verify inventory reduced correctly
		totalRemaining := GetTotalQuantity(inventory, 1)
		assert.Equal(t, 1, totalRemaining, "Should have 1 item remaining (8 - 7)")
	})

	t.Run("returns error on insufficient items", func(t *testing.T) {
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon},
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
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityLegendary},
				{ItemID: 2, Quantity: 10, QualityLevel: domain.QualityCommon},
			},
		}

		consumed, err := ConsumeItemsWithTracking(inventory, 1, 5, func() float64 { return 0.0 })

		assert.NoError(t, err)
		assert.Equal(t, 1, len(consumed))
		assert.Equal(t, domain.QualityLegendary, consumed[0].QualityLevel)
		assert.Equal(t, 1, len(inventory.Slots), "Fully consumed slot should be removed")
		assert.Equal(t, 2, inventory.Slots[0].ItemID, "Remaining slot should be ItemID 2")
	})
}
