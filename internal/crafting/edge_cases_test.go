package crafting

import (
	"context"
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestCrossOperationConcurrency verifies that UpgradeItem and DisassembleItem
// can run concurrently for the same user without corrupting inventory.
// This simulates a user trying to "dupe" or race the system by performing opposing actions rapidly.
func TestCrossOperationConcurrency(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 } // Deterministic RNG
	ctx := context.Background()

	// Arrange:
	// Start with:
	// - 100 Lootbox0 (Base material for Upgrade)
	// - 100 Lootbox1 (Source material for Disassemble)
	//
	// Recipes:
	// - Upgrade: 1 Lootbox0 -> 1 Lootbox1
	// - Disassemble: 1 Lootbox1 -> 1 Lootbox0
	//
	// Operations:
	// - 50 concurrent Upgrades of 1 Lootbox0
	// - 50 concurrent Disassembles of 1 Lootbox1
	//
	// Expected Result:
	// - Net change should be zero if all succeed (50 LB0->LB1, 50 LB1->LB0).
	// - Final state: 100 LB0, 100 LB1.
	// - However, due to concurrency, intermediate states must be safe.

	initialQty := 100
	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: initialQty}, // Lootbox0
		{ItemID: 2, Quantity: initialQty}, // Lootbox1
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1) // Unlock Upgrade Recipe (ID 1)

	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	// Launch 50 upgrades
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
			if err != nil {
				errChan <- err
			}
		}()
	}

	// Launch 50 disassembles
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		// It is acceptable for operations to fail due to transient lock contention if the system uses timeouts,
		// but our mock uses mutexes which just block, so they should all succeed eventually.
		// However, if logic was wrong (e.g., overdraft), we'd see errors here.
		assert.NoError(t, err)
	}

	// Verify Inventory Integrity
	inv, err := repo.GetInventory(ctx, "user-alice")
	require.NoError(t, err)

	lb0Count := 0
	lb1Count := 0
	for _, slot := range inv.Slots {
		if slot.ItemID == 1 {
			lb0Count = slot.Quantity
		}
		if slot.ItemID == 2 {
			lb1Count = slot.Quantity
		}
	}

	// Since we did 50 upgrades ( -50 LB0, +50 LB1)
	// And 50 disassembles ( -50 LB1, +50 LB0)
	// The net result should be back to initial state.
	assert.Equal(t, initialQty, lb0Count, "Lootbox0 count mismatch")
	assert.Equal(t, initialQty, lb1Count, "Lootbox1 count mismatch")
}

// TestIntegerOverflow attempts to trigger overflow issues using MaxInt.
func TestIntegerOverflow(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil).(*service)
	ctx := context.Background()

	// Arrange: Give alice minimal materials
	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: 10},
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	t.Run("Upgrade with MaxInt", func(t *testing.T) {
		// Act: Request MaxInt items
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, math.MaxInt)

		// Assert: Should not panic, should cap at available materials (10)
		assert.NoError(t, err)
		assert.Equal(t, 10, result.Quantity)
	})

	t.Run("Disassemble with MaxInt", func(t *testing.T) {
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 5},
		}})

		// Act
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, math.MaxInt)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 5, result.QuantityProcessed)
	})
}

// TestZeroNegativeQuantity verifies input validation logic.
func TestZeroNegativeQuantity(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: 10},
		{ItemID: 2, Quantity: 10},
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	tests := []struct {
		name     string
		quantity int
	}{
		{"Zero", 0},
		{"Negative", -1},
		{"Large Negative", -1000},
	}

	for _, tt := range tests {
		t.Run("Upgrade "+tt.name, func(t *testing.T) {
			_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, tt.quantity)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid quantity")
		})

		t.Run("Disassemble "+tt.name, func(t *testing.T) {
			_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, tt.quantity)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid quantity")
		})
	}
}

// TestTransactionRollback ensures that inventory changes are rolled back if the commit/update fails.
func TestTransactionRollback(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	// Arrange: Alice has 1 Lootbox0
	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: 1},
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Inject Failure on UpdateInventory
	repo.shouldFailUpdateInventory = true

	// Act
	_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update inventory")

	// Verify Inventory State: Should still have 1 Lootbox0, NOT 0.
	inv, _ := repo.GetInventory(ctx, "user-alice")
	require.Len(t, inv.Slots, 1)
	assert.Equal(t, 1, inv.Slots[0].Quantity, "Materials should not be consumed if transaction fails")
	assert.Equal(t, 1, inv.Slots[0].ItemID)
}
