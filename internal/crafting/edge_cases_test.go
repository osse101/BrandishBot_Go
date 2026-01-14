package crafting

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrafting_EdgeCases(t *testing.T) {
	// Setup shared repo
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 } // Deterministic RNG
	ctx := context.Background()

	t.Run("UpgradeItem_ZeroQuantity", func(t *testing.T) {
		// Arrange
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 10},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 0)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "quantity must be positive")
	})

	t.Run("UpgradeItem_NegativeQuantity", func(t *testing.T) {
		// Arrange
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 10},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		// This should fail validation, not generate items!
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, -5)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "quantity must be positive")

		// Verify no items were added/removed (exploit check)
		inv, _ := repo.GetInventory(ctx, "user-alice")
		for _, slot := range inv.Slots {
			if slot.ItemID == 1 {
				assert.Equal(t, 10, slot.Quantity, "Inventory should not change on negative input")
			}
		}
	})

	t.Run("DisassembleItem_ZeroQuantity", func(t *testing.T) {
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 10},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "quantity must be positive")
	})

	t.Run("DisassembleItem_NegativeQuantity", func(t *testing.T) {
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 10},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, -5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "quantity must be positive")
	})

	t.Run("UpgradeItem_Overflow_Safe", func(t *testing.T) {
		// Attempt to cause overflow with MaxInt
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 100},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Asking for MaxInt should be clamped to what is affordable (100)
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, math.MaxInt)

		assert.NoError(t, err)
		assert.Equal(t, 100, result.Quantity, "Should clamp to affordable quantity")
	})
}

func TestCrafting_RaceConditions_CrossOperation(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }
	ctx := context.Background()

	// Alice starts with 100 Lootbox1 (Tier 1) and 100 Lootbox0 (Tier 0)
	// She will simultaneously:
	// 1. Upgrade 50 Lootbox0 -> 50 Lootbox1
	// 2. Disassemble 50 Lootbox1 -> 50 Lootbox0
	//
	// Net result should be valid inventory states throughout.
	// We want to ensure no "lost updates" occur on the inventory slice.

	initialLootbox0 := 100
	initialLootbox1 := 100

	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: initialLootbox0},
		{ItemID: 2, Quantity: initialLootbox1},
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	var wg sync.WaitGroup
	start := make(chan struct{})

	// Routine 1: Upgrade 100 times (1 item each)
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 50; i++ {
			_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
			// It's possible to run out if disassemble is slow, but given initial counts, we should be fine.
			// Actually, if we interleave, we might momentarily run out if we didn't have enough buffer.
			// With 100 of each and doing 50 of each, we are safe from running out.
			if err != nil {
				// Log but don't fail, as transient failures might be "ok" in some weird race,
				// but strict locking should prevent it.
				t.Logf("Upgrade error: %v", err)
			}
		}
	}()

	// Routine 2: Disassemble 100 times (1 item each)
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 50; i++ {
			_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
			if err != nil {
				t.Logf("Disassemble error: %v", err)
			}
		}
	}()

	close(start)

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for concurrent operations")
	}

	// Verify final state
	inv, _ := repo.GetInventory(ctx, "user-alice")

	// Upgrade: -50 LB0, +50 LB1
	// Disassemble: -50 LB1, +50 LB0
	// Net change: 0

	lb0 := 0
	lb1 := 0
	for _, slot := range inv.Slots {
		if slot.ItemID == 1 {
			lb0 = slot.Quantity
		}
		if slot.ItemID == 2 {
			lb1 = slot.Quantity
		}
	}

	assert.Equal(t, initialLootbox0, lb0, "Lootbox0 count mismatch")
	assert.Equal(t, initialLootbox1, lb1, "Lootbox1 count mismatch")
}

// TestValidateUser_NotFound ensures explicit error when user doesn't exist
func TestValidateUser_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo, nil, nil, nil).(*service)
	ctx := context.Background()

	_, err := svc.validateUser(ctx, domain.PlatformTwitch, "non-existent")
	require.Error(t, err)
	assert.Equal(t, "user not found", err.Error())
}
