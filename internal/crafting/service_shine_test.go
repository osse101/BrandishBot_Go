package crafting

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestUpgradeItem_DoesNotStackWithShiny verifies that standard crafted items
// do not stack with existing shiny items of the same type.
// This ensures that crafting doesn't accidentally "upgrade" items to shiny status
// just because a shiny stack already exists.
func TestUpgradeItem_DoesNotStackWithShiny(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 } // No masterwork
	ctx := context.Background()

	// Arrange: User has 1 Shiny Lootbox1 (Target Item)
	// and enough Lootbox0 to craft another Lootbox1
	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 2, Quantity: 1, ShineLevel: domain.ShineRare}, // Shiny Lootbox1
		{ItemID: 1, Quantity: 1, ShineLevel: domain.ShineCommon}, // Standard Lootbox0 (Material)
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1) // Recipe: Lootbox0 -> Lootbox1

	// Act: Craft 1 Lootbox1
	result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Quantity)

	inv, _ := repo.GetInventory(ctx, "user-alice")

	// We expect 2 slots for ItemID 2 (Lootbox1):
	// 1. The original Shiny one
	// 2. The new Standard one
	lootbox1Slots := 0
	for _, slot := range inv.Slots {
		if slot.ItemID == 2 {
			lootbox1Slots++
			if slot.ShineLevel == domain.ShineRare {
				assert.Equal(t, 1, slot.Quantity, "Shiny stack quantity should not change")
			} else {
				assert.Equal(t, 1, slot.Quantity, "Standard stack should have quantity 1")
				assert.Equal(t, domain.ShineCommon, slot.ShineLevel, "New item should be standard")
			}
		}
	}
	assert.Equal(t, 2, lootbox1Slots, "Should have 2 separate slots for Lootbox1 (Shiny + Standard)")
}

// TestDisassembleItem_DoesNotStackWithShiny verifies that disassembled outputs
// do not stack with existing shiny items of the same type.
func TestDisassembleItem_DoesNotStackWithShiny(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 } // No perfect salvage
	ctx := context.Background()

	// Arrange: User has 1 Shiny Lootbox0 (Output Item)
	// and 1 Lootbox1 (Source Item) to disassemble
	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: 1, ShineLevel: domain.ShineRare}, // Shiny Lootbox0
		{ItemID: 2, Quantity: 1, ShineLevel: domain.ShineCommon}, // Standard Lootbox1
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1) // Unlock upgrade recipe (required for disassemble)

	// Act: Disassemble 1 Lootbox1 -> 1 Lootbox0
	result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, result.QuantityProcessed)

	inv, _ := repo.GetInventory(ctx, "user-alice")

	// We expect 2 slots for ItemID 1 (Lootbox0):
	// 1. The original Shiny one
	// 2. The new Standard one
	lootbox0Slots := 0
	for _, slot := range inv.Slots {
		if slot.ItemID == 1 {
			lootbox0Slots++
			if slot.ShineLevel == domain.ShineRare {
				assert.Equal(t, 1, slot.Quantity, "Shiny stack quantity should not change")
			} else {
				assert.Equal(t, 1, slot.Quantity, "Standard stack should have quantity 1")
				assert.Equal(t, domain.ShineCommon, slot.ShineLevel, "New item should be standard")
			}
		}
	}
	assert.Equal(t, 2, lootbox0Slots, "Should have 2 separate slots for Lootbox0 (Shiny + Standard)")
}

func TestUpgradeItem_ProgressionServiceError(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)

	// Mock progression service to return error
	mockProg := &MockProgressionService{
		returnError: assert.AnError,
	}

	svc := NewService(repo, nil, nil, nil, mockProg, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }
	ctx := context.Background()

	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: 1},
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Act
	result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

	// Assert
	assert.NoError(t, err) // Should succeed despite progression error
	assert.Equal(t, 1, result.Quantity)

	// Verify it called progression service
	assert.NotEmpty(t, mockProg.calls)
	assert.Equal(t, "crafting_success_rate", mockProg.calls[0].featureKey)
}

func TestUpgradeItem_QuestServiceError(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)

	// Mock quest service to return error
	mockQuest := &MockQuestService{
		returnError: assert.AnError,
	}

	svc := NewService(repo, nil, nil, nil, nil, mockQuest).(*service)
	svc.rnd = func() float64 { return 1.0 }
	ctx := context.Background()

	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: 1},
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Act
	result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

	// Assert
	assert.NoError(t, err) // Should succeed despite quest error (async)
	assert.Equal(t, 1, result.Quantity)

	// Wait for async
	svc.Shutdown(ctx)

	// Verify it called quest service
	assert.NotEmpty(t, mockQuest.craftedCalls)
	assert.Equal(t, "user-alice", mockQuest.craftedCalls[0].UserID)
}

func TestDisassembleItem_QuestServiceError(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)

	mockQuest := &MockQuestService{
		returnError: assert.AnError,
	}

	svc := NewService(repo, nil, nil, nil, nil, mockQuest).(*service)
	svc.rnd = func() float64 { return 1.0 }
	ctx := context.Background()

	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 2, Quantity: 1},
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Act
	_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

	// Assert
	assert.NoError(t, err)

	// Wait for async
	done := make(chan struct{})
	go func() {
		svc.Shutdown(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Shutdown timed out")
	}

	assert.NotEmpty(t, mockQuest.craftedCalls)
}
