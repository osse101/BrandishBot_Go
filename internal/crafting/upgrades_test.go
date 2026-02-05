package crafting

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// This file contains tests for crafting upgrade node modifier application.
// Tests verify that the crafting_success_rate modifier correctly applies to
// both masterwork crafting and perfect salvage operations.


// TestUpgradeCrafting1_MasterworkModifier_Level1 verifies 10% boost at level 1
func TestUpgradeCrafting1_MasterworkModifier_Level1(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)
	mockProg := &MockProgressionService{}

	svc := NewService(repo, nil, nil, nil, mockProg).(*service)
	ctx := context.Background()

	// Level 1 upgrade: 1.10x multiplier (0.10 * 1.10 = 0.11 = 11% chance)
	mockProg.returnValue = 0.11

	// Unlock recipe
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Test with deterministic RNG - run 100 crafts
	masterworkCount := 0

	for i := 0; i < 100; i++ {
		// Roll values from 0.00 to 0.99
		rollValue := float64(i) / 100.0
		svc.rnd = func() float64 { return rollValue }

		// Setup inventory with materials for each craft
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 10}, // lootbox0
		}})

		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		if err == nil && result.IsMasterwork {
			masterworkCount++
		}
	}

	// ASSERT
	// With 0.11 threshold, rolls 0.00-0.10 (11 values out of 100) should trigger masterwork
	assert.Equal(t, 11, masterworkCount, "Should get 11 masterworks out of 100 crafts with level 1 upgrade (11% rate)")
	assert.NotEmpty(t, mockProg.calls, "Expected GetModifiedValue to be called")
}

// TestUpgradeCrafting1_MasterworkModifier_Level5 verifies 50% boost at level 5
func TestUpgradeCrafting1_MasterworkModifier_Level5(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)
	mockProg := &MockProgressionService{}

	svc := NewService(repo, nil, nil, nil, mockProg).(*service)
	ctx := context.Background()

	// Level 5 upgrade: 1.50x multiplier (0.10 * 1.50 = 0.15 = 15% chance)
	mockProg.returnValue = 0.15

	repo.UnlockRecipe(ctx, "user-alice", 1)

	masterworkCount := 0

	for i := 0; i < 100; i++ {
		rollValue := float64(i) / 100.0
		svc.rnd = func() float64 { return rollValue }

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 10},
		}})

		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		if err == nil && result.IsMasterwork {
			masterworkCount++
		}
	}

	// ASSERT
	// With 0.15 threshold, rolls 0.00-0.14 (15 values out of 100) should trigger masterwork
	assert.Equal(t, 15, masterworkCount, "Should get 15 masterworks out of 100 crafts with level 5 upgrade (15% rate)")
	assert.NotEmpty(t, mockProg.calls, "Expected GetModifiedValue to be called")
}

// TestUpgradeCrafting1_MasterworkModifier_NoUpgrade verifies base rate without upgrade
func TestUpgradeCrafting1_MasterworkModifier_NoUpgrade(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)
	mockProg := &MockProgressionService{}

	svc := NewService(repo, nil, nil, nil, mockProg).(*service)
	ctx := context.Background()

	// No upgrade: return base value (1.0x multiplier)
	mockProg.returnValue = 0.10

	repo.UnlockRecipe(ctx, "user-alice", 1)

	masterworkCount := 0

	for i := 0; i < 100; i++ {
		rollValue := float64(i) / 100.0
		svc.rnd = func() float64 { return rollValue }

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 10},
		}})

		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		if err == nil && result.IsMasterwork {
			masterworkCount++
		}
	}

	// ASSERT
	assert.Equal(t, 10, masterworkCount, "Should get 10 masterworks out of 100 crafts with no upgrade (10% base rate)")
	assert.NotEmpty(t, mockProg.calls, "Expected GetModifiedValue to be called")
}

// TestUpgradeCrafting1_PerfectSalvageModifier_Level1 verifies salvage modifier at level 1
func TestUpgradeCrafting1_PerfectSalvageModifier_Level1(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)
	mockProg := &MockProgressionService{}

	svc := NewService(repo, nil, nil, nil, mockProg).(*service)
	ctx := context.Background()

	// Level 1 upgrade: 1.10x multiplier (0.10 * 1.10 = 0.11 = 11% chance)
	mockProg.returnValue = 0.11

	repo.UnlockRecipe(ctx, "user-alice", 1)

	perfectCount := 0

	for i := 0; i < 100; i++ {
		rollValue := float64(i) / 100.0
		svc.rnd = func() float64 { return rollValue }

		// Setup inventory with item to disassemble
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 10}, // lootbox1
		}})

		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		if err == nil && result.IsPerfectSalvage {
			perfectCount++
		}
	}

	// ASSERT
	assert.Equal(t, 11, perfectCount, "Should get 11 perfect salvages out of 100 with level 1 upgrade (11% rate)")
}

// TestUpgradeCrafting1_PerfectSalvageModifier_Level5 verifies salvage modifier at level 5
func TestUpgradeCrafting1_PerfectSalvageModifier_Level5(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)
	mockProg := &MockProgressionService{}

	svc := NewService(repo, nil, nil, nil, mockProg).(*service)
	ctx := context.Background()

	// Level 5 upgrade: 1.50x multiplier (0.10 * 1.50 = 0.15 = 15% chance)
	mockProg.returnValue = 0.15

	repo.UnlockRecipe(ctx, "user-alice", 1)

	perfectCount := 0

	for i := 0; i < 100; i++ {
		rollValue := float64(i) / 100.0
		svc.rnd = func() float64 { return rollValue }

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 10},
		}})

		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		if err == nil && result.IsPerfectSalvage {
			perfectCount++
		}
	}

	// ASSERT
	assert.Equal(t, 15, perfectCount, "Should get 15 perfect salvages out of 100 with level 5 upgrade (15% rate)")
}

// TestUpgradeCrafting1_ModifierFailureFallback_Masterwork verifies graceful fallback on error
func TestUpgradeCrafting1_ModifierFailureFallback_Masterwork(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)
	mockProg := &MockProgressionService{}

	svc := NewService(repo, nil, nil, nil, mockProg).(*service)
	ctx := context.Background()

	// Progression service returns error
	mockProg.returnError = errors.New("progression service error")

	repo.UnlockRecipe(ctx, "user-alice", 1)

	masterworkCount := 0

	for i := 0; i < 100; i++ {
		rollValue := float64(i) / 100.0
		svc.rnd = func() float64 { return rollValue }

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 10},
		}})

		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		if err == nil && result.IsMasterwork {
			masterworkCount++
		}
	}

	// ASSERT
	// Should fall back to base rate (0.10 = 10%)
	assert.Equal(t, 10, masterworkCount, "Should use base masterwork rate (10%) on error")
	assert.NotEmpty(t, mockProg.calls, "Expected GetModifiedValue to be called")
}

// TestUpgradeCrafting1_NilProgressionService_Masterwork verifies graceful degradation
func TestUpgradeCrafting1_NilProgressionService_Masterwork(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)

	svc := NewService(repo, nil, nil, nil, nil).(*service) // nil progression service
	ctx := context.Background()

	repo.UnlockRecipe(ctx, "user-alice", 1)

	masterworkCount := 0

	for i := 0; i < 100; i++ {
		rollValue := float64(i) / 100.0
		svc.rnd = func() float64 { return rollValue }

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 10},
		}})

		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		if err == nil && result.IsMasterwork {
			masterworkCount++
		}
	}

	// ASSERT
	assert.Equal(t, 10, masterworkCount, "Should use base rate (10%) when progression service is nil")
}

// TestUpgradeCrafting1_NilProgressionService_PerfectSalvage verifies salvage degradation
func TestUpgradeCrafting1_NilProgressionService_PerfectSalvage(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)

	svc := NewService(repo, nil, nil, nil, nil).(*service)
	ctx := context.Background()

	repo.UnlockRecipe(ctx, "user-alice", 1)

	perfectCount := 0

	for i := 0; i < 100; i++ {
		rollValue := float64(i) / 100.0
		svc.rnd = func() float64 { return rollValue }

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 10},
		}})

		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		if err == nil && result.IsPerfectSalvage {
			perfectCount++
		}
	}

	// ASSERT
	assert.Equal(t, 10, perfectCount, "Should use base salvage rate (10%) when progression service is nil")
}

// TestUpgradeCrafting1_MasterworkMultiplier verifies 2x output on masterwork
func TestUpgradeCrafting1_MasterworkMultiplier(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)
	mockProg := &MockProgressionService{}

	svc := NewService(repo, nil, nil, nil, mockProg).(*service)
	ctx := context.Background()

	// Return base value
	mockProg.returnValue = 0.10

	// Force masterwork with low RNG roll
	svc.rnd = func() float64 { return 0.05 }

	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Setup inventory
	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: 10}, // lootbox0
	}})

	// ACT
	result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

	// ASSERT
	require.NoError(t, err)
	assert.True(t, result.IsMasterwork, "Should trigger masterwork")
	assert.Equal(t, 2, result.Quantity, "Masterwork should give 2x output (base 1 * 2)")
	assert.Equal(t, 1, result.BonusQuantity, "Bonus should be 1 (2 - 1)")
}

// TestUpgradeCrafting1_PerfectSalvageMultiplier verifies 1.5x output on perfect salvage
func TestUpgradeCrafting1_PerfectSalvageMultiplier(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)
	mockProg := &MockProgressionService{}

	svc := NewService(repo, nil, nil, nil, mockProg).(*service)
	ctx := context.Background()

	// Return base value
	mockProg.returnValue = 0.10

	// Force perfect salvage with low RNG roll
	svc.rnd = func() float64 { return 0.05 }

	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Setup inventory with item to disassemble
	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 2, Quantity: 10}, // lootbox1
	}})

	// ACT
	result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

	// ASSERT
	require.NoError(t, err)
	assert.True(t, result.IsPerfectSalvage, "Should trigger perfect salvage")
	assert.Equal(t, 1.5, result.Multiplier, "Perfect salvage multiplier should be 1.5x")
}

// TestUpgradeCrafting1_BulkCrafting_WithModifier verifies modifier applies to bulk operations
func TestUpgradeCrafting1_BulkCrafting_WithModifier(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)
	mockProg := &MockProgressionService{}

	svc := NewService(repo, nil, nil, nil, mockProg).(*service)
	ctx := context.Background()

	// Level 3 upgrade: 1.30x multiplier (0.10 * 1.30 = 0.13 = 13% chance)
	mockProg.returnValue = 0.13

	// Use RNG sequence that triggers masterwork for ~13% of 10 crafts
	callCount := 0
	svc.rnd = func() float64 {
		// Return values: 0.05, 0.11 trigger masterwork (< 0.13)
		// Others don't
		values := []float64{0.05, 0.15, 0.25, 0.35, 0.45, 0.55, 0.65, 0.75, 0.85, 0.11}
		val := values[callCount%len(values)]
		callCount++
		return val
	}

	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Setup inventory with materials for 10 crafts
	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: 100}, // Plenty of materials
	}})

	// ACT
	result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 10)

	// ASSERT
	require.NoError(t, err)
	// 2 masterworks out of 10 = 2 bonus items
	// Total: 10 + 2 = 12
	assert.Equal(t, 12, result.Quantity, "Should get 12 items total (10 normal + 2 masterwork bonuses)")
	assert.Equal(t, 2, result.BonusQuantity, "Should get 2 bonus items from masterworks")
}
