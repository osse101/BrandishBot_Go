package crafting

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// ==================== Tests ====================

func TestDisassembleItem(t *testing.T) {
	t.Parallel()

	t.Run("Best Case: Success", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // No perfect salvage
		ctx := context.Background()

		// Arrange: Give alice 3 lootbox1 and unlock recipe
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID2, Quantity: 3},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 2, result.QuantityProcessed)
		assert.Equal(t, 2, result.Outputs[domain.ItemLootbox0])

		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, 1, inv.Slots[0].Quantity, "Should have 1 lootbox1 remaining")
	})

	t.Run("Best Case: Perfect Salvage", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		mockStats := &MockStatsService{}
		svc := NewService(repo, nil, mockStats, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 0.0 } // Trigger perfect salvage
		ctx := context.Background()

		// Arrange: Give alice 1 lootbox1
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID2, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		// Assert
		assert.NoError(t, err)
		assert.True(t, result.IsPerfectSalvage)

		// Logic: 1 item disassembled -> 1 source consumed.
		// Recipe Output: 1 Lootbox0.
		// Perfect Salvage: ceil(1 * 1.5) = 2.
		// Total Output: 1 * 2 = 2.
		assert.Equal(t, 2, result.Outputs[domain.ItemLootbox0])

		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, 2, inv.Slots[0].Quantity, "Should have 2 lootbox0")

		// Verify event
		foundEvent := false
		mockStats.mu.Lock()
		events := mockStats.events
		mockStats.mu.Unlock()
		for _, e := range events {
			if e == domain.EventCraftingPerfectSalvage {
				foundEvent = true
				break
			}
		}
		assert.True(t, foundEvent, "Should log perfect salvage event")
	})

	t.Run("Boundary Case: Exact Items", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Arrange: Give alice exactly 2 lootbox1
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID2, Quantity: 2},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 2, result.QuantityProcessed)

		inv, _ := repo.GetInventory(ctx, "user-alice")
		foundLootbox1 := false
		for _, slot := range inv.Slots {
			if slot.ItemID == TestItemID2 {
				foundLootbox1 = true
			}
		}
		assert.False(t, foundLootbox1, "Lootbox1 slot should be removed")
	})

	t.Run("Error Case: Insufficient Items", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		// Arrange: Alice has 1 lootbox1, wants to disassemble 2
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID2, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 1, result.QuantityProcessed, "Should only process 1 available item")
	})

	t.Run("Error Case: Recipe Not Unlocked", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID2, Quantity: 1},
		}})

		// Act
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRecipeLocked)
	})

	t.Run("Error Case: No Recipe Exists", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		// Act
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox0, 1) // No disassemble recipe for lootbox0

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRecipeNotFound)
	})

	t.Run("Nil/Empty Case: Empty User", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo) // Need to setup items so item validation passes
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "nonexistent", "", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("Concurrent Case: Parallel Disassemble", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Arrange: Give alice 100 items
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID2, Quantity: 100},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act: Run 10 goroutines disassembling 1 item each
		var wg sync.WaitGroup
		startChan := make(chan struct{})
		errChan := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startChan
				_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
				if err != nil {
					errChan <- err
				}
			}()
		}
		close(startChan)
		wg.Wait()
		close(errChan)

		// Assert
		for err := range errChan {
			assert.NoError(t, err)
		}

		inv, _ := repo.GetInventory(ctx, "user-alice")
		for _, slot := range inv.Slots {
			if slot.ItemID == TestItemID2 {
				assert.Equal(t, 90, slot.Quantity, "Should have 90 items left")
			}
		}
	})

	t.Run("Split Stack Case: Disassemble", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 0.5 }
		ctx := context.Background()

		// Arrange: Split stack of 10 items (5 + 5)
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID2, Quantity: 5},
			{ItemID: TestItemID2, Quantity: 5},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1) // recipe for disassemble item 2 (lootbox1)

		// Act: Disassemble 10 items (needs to consume both stacks)
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 10)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 10, result.QuantityProcessed)

		inv, _ := repo.GetInventory(ctx, "user-alice")
		for _, slot := range inv.Slots {
			if slot.ItemID == TestItemID2 {
				assert.Equal(t, 0, slot.Quantity)
			}
		}
	})
}

func TestUpgradeItem(t *testing.T) {
	t.Parallel()

	t.Run("Best Case: Success", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // Fail masterwork
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 2}, // 2 lootbox0
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 2, result.Quantity)
		assert.Equal(t, domain.ItemLootbox1, result.ItemName)

		inv, _ := repo.GetInventory(ctx, "user-alice")
		// Should have 0 lootbox0 and 2 lootbox1
		for _, slot := range inv.Slots {
			if slot.ItemID == TestItemID1 {
				assert.Equal(t, 0, slot.Quantity)
			}
			if slot.ItemID == TestItemID2 {
				assert.Equal(t, 2, slot.Quantity)
			}
		}
	})

	t.Run("Best Case: Masterwork", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		mockStats := &MockStatsService{}
		svc := NewService(repo, nil, mockStats, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 0.0 } // Trigger masterwork
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 2},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 4, result.Quantity, "Should double quantity")
		assert.True(t, result.IsMasterwork)

		foundEvent := false
		mockStats.mu.Lock()
		events := mockStats.events
		mockStats.mu.Unlock()
		for _, e := range events {
			if e == domain.EventCraftingCriticalSuccess {
				foundEvent = true
				break
			}
		}
		assert.True(t, foundEvent, "Should log critical success event")
	})

	t.Run("Error Case: Insufficient Materials", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Have 1, want 2
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 1, result.Quantity, "Should process max available")
	})

	t.Run("Error Case: Recipe Not Unlocked", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID2, Quantity: 2},
		}})

		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRecipeLocked)
	})

	t.Run("Concurrent Case: Parallel Upgrades", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// 100 items
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 100},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		var wg sync.WaitGroup
		startChan := make(chan struct{})
		errChan := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startChan
				_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
				if err != nil {
					errChan <- err
				}
			}()
		}
		close(startChan)
		wg.Wait()
		close(errChan)

		for err := range errChan {
			assert.NoError(t, err)
		}

		inv, _ := repo.GetInventory(ctx, "user-alice")
		for _, slot := range inv.Slots {
			if slot.ItemID == TestItemID1 {
				assert.Equal(t, 90, slot.Quantity)
			}
			if slot.ItemID == TestItemID2 {
				assert.Equal(t, 10, slot.Quantity)
			}
		}
	})

	t.Run("Split Stack Case: Upgrade", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 0.5 }
		ctx := context.Background()

		// Arrange: Split stack of 10 items (5 + 5)
		// Recipe needs 1 item per craft. Requesting 10 crafts.
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 5},
			{ItemID: TestItemID1, Quantity: 5},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 10)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 10, result.Quantity, "Should craft 10 items using materials from both stacks")

		inv, _ := repo.GetInventory(ctx, "user-alice")
		totalRemaining := 0
		for _, slot := range inv.Slots {
			if slot.ItemID == TestItemID1 {
				totalRemaining += slot.Quantity
			}
		}
		assert.Equal(t, 0, totalRemaining, "All materials should be consumed")
	})

	t.Run("Quality Inheritance: Mixed Quality", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // Prevent masterwork
		ctx := context.Background()

		// Setup: Create a recipe requiring 2 materials: 2x lootbox0 + 1x lootbox1 -> lootbox2
		// This consumes 3 items per craft.
		// We will test mixing qualities to verify average calculation.
		repo.Lock()
		repo.recipes[99] = &domain.Recipe{
			ID:           99,
			TargetItemID: TestItemID3, // lootbox_tier2
			BaseCost: []domain.RecipeCost{
				{ItemID: TestItemID1, Quantity: 2}, // 2x lootbox_tier0
				{ItemID: TestItemID2, Quantity: 1}, // 1x lootbox_tier1
			},
		}
		repo.Unlock()
		repo.UnlockRecipe(ctx, "user-alice", 99)

		// Scenario:
		// We want to craft 1 item.
		// Needs 2x lootbox0 and 1x lootbox1.
		// We provide:
		// - 2x lootbox0 of Quality COMMON (value 3)
		// - 1x lootbox1 of Quality RARE (value 5)
		// Total Value: (2*3) + (1*5) = 6 + 5 = 11
		// Total Count: 3
		// Average: (11 + 3/2) / 3 = 12 / 3 = 4 -> UNCOMMON
		// Output should be UNCOMMON.

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 2, QualityLevel: domain.QualityCommon},
			{ItemID: TestItemID2, Quantity: 1, QualityLevel: domain.QualityRare},
		}})

		// Act
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox2, 1)
		assert.NoError(t, err)

		// Assert
		inv, _ := repo.GetInventory(ctx, "user-alice")
		found := false
		for _, slot := range inv.Slots {
			if slot.ItemID == TestItemID3 {
				found = true
				assert.Equal(t, domain.QualityUncommon, slot.QualityLevel, "Output should average to UNCOMMON")
			}
		}
		assert.True(t, found, "Should have crafted lootbox2")
	})

	t.Run("XP Failure Ignored", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)

		// Setup mock job service to return error
		mockJob := &MockJobService{
			calls: []struct {
				UserID   string
				JobKey   string
				Amount   int
				Source   string
				Metadata map[string]interface{}
			}{},
			returnError: fmt.Errorf("XP service unavailable"),
		}

		svc := NewService(repo, mockJob, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		// Assert
		assert.NoError(t, err, "Should succeed even if XP award fails")
		assert.Equal(t, 1, result.Quantity)
	})
}

func TestGetRecipe(t *testing.T) {
	t.Parallel()

	t.Run("Best Case: Unlocked", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		repo.UnlockRecipe(ctx, "user-alice", 1)
		recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, domain.PlatformTwitch, "twitch-alice", "alice")
		assert.NoError(t, err)
		assert.False(t, recipe.Locked)
	})

	t.Run("Best Case: No User Context", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, "", "", "")
		assert.NoError(t, err)
		assert.False(t, recipe.Locked, "Should default to false if no user")
	})

	t.Run("Boundary Case: Locked", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, domain.PlatformTwitch, "twitch-alice", "alice")
		assert.NoError(t, err)
		assert.True(t, recipe.Locked)
	})

	t.Run("Error Case: Item Not Found", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		_, err := svc.GetRecipe(ctx, "invalid-item", "", "", "")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrItemNotFound)
	})
}

func TestGetAllRecipes(t *testing.T) {
	t.Parallel()

	t.Run("Best Case: Returns Recipes", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		recipes, err := svc.GetAllRecipes(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, recipes)
	})
}

func TestShutdown(t *testing.T) {
	t.Parallel()
	repo := NewMockRepository()
	svc := NewService(repo, nil, nil, nil, nil, nil)
	assert.NoError(t, svc.Shutdown(context.Background()))
}

// Additional test for GetUnlockedRecipes
func TestGetUnlockedRecipes(t *testing.T) {
	t.Parallel()

	t.Run("Best Case: Returns Unlocked", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		repo.UnlockRecipe(ctx, "user-alice", 1)
		recipes, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-alice", "alice")
		assert.NoError(t, err)
		assert.Len(t, recipes, 1)
	})

	t.Run("Nil/Empty Case: No Unlocked", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil)
		ctx := context.Background()

		recipes, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-alice", "alice")
		assert.NoError(t, err)
		assert.Empty(t, recipes)
	})
}

// Example: Concurrent access to GetRecipe is read-only, but let's verify it doesn't race
func TestGetRecipe_Concurrent(t *testing.T) {
	t.Parallel()
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil, nil)
	ctx := context.Background()

	var wg sync.WaitGroup
	startChan := make(chan struct{})
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startChan
			svc.GetRecipe(ctx, domain.ItemLootbox1, domain.PlatformTwitch, "twitch-alice", "alice")
		}()
	}
	close(startChan)
	wg.Wait()
}

// Phase 3: Input Validation Tests

func TestUpgradeItem_InputValidation(t *testing.T) {
	t.Parallel()
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name       string
		platform   string
		platformID string
		itemName   string
		quantity   int
		expected   error
		errorMsg   string
	}{
		{
			name:       "Negative Quantity",
			platform:   domain.PlatformTwitch,
			platformID: "twitch-alice",
			itemName:   domain.ItemLootbox1,
			quantity:   -1,
			expected:   domain.ErrInvalidQuantity,
			errorMsg:   "quantity must be positive",
		},
		{
			name:       "Zero Quantity",
			platform:   domain.PlatformTwitch,
			platformID: "twitch-alice",
			itemName:   domain.ItemLootbox1,
			quantity:   0,
			expected:   domain.ErrInvalidQuantity,
			errorMsg:   "quantity must be positive",
		},
		{
			name:       "Empty Platform",
			platform:   "",
			platformID: "twitch-alice",
			itemName:   domain.ItemLootbox1,
			quantity:   1,
			expected:   domain.ErrInvalidInput,
			errorMsg:   "platform and platformID cannot be empty",
		},
		{
			name:       "Empty PlatformID",
			platform:   domain.PlatformTwitch,
			platformID: "",
			itemName:   domain.ItemLootbox1,
			quantity:   1,
			expected:   domain.ErrInvalidInput,
			errorMsg:   "platform and platformID cannot be empty",
		},
		{
			name:       "Invalid Platform",
			platform:   "invalid-platform",
			platformID: "some-id",
			itemName:   domain.ItemLootbox1,
			quantity:   1,
			expected:   domain.ErrInvalidPlatform,
			errorMsg:   "invalid platform",
		},
		{
			name:       "Empty Item Name",
			platform:   domain.PlatformTwitch,
			platformID: "twitch-alice",
			itemName:   "",
			quantity:   1,
			expected:   domain.ErrInvalidInput,
			errorMsg:   "item name cannot be empty",
		},
	}

	for _, tt := range tests {
		tt := tt // capture loop var
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.UpgradeItem(ctx, tt.platform, tt.platformID, "alice", tt.itemName, tt.quantity)
			assert.Error(t, err)
			assert.ErrorIs(t, err, tt.expected)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestDisassembleItem_InputValidation(t *testing.T) {
	t.Parallel()
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name       string
		platform   string
		platformID string
		itemName   string
		quantity   int
		expected   error
		errorMsg   string
	}{
		{
			name:       "Negative Quantity",
			platform:   domain.PlatformTwitch,
			platformID: "twitch-alice",
			itemName:   domain.ItemLootbox1,
			quantity:   -1,
			expected:   domain.ErrInvalidQuantity,
			errorMsg:   "quantity must be positive",
		},
		{
			name:       "Zero Quantity",
			platform:   domain.PlatformTwitch,
			platformID: "twitch-alice",
			itemName:   domain.ItemLootbox1,
			quantity:   0,
			expected:   domain.ErrInvalidQuantity,
			errorMsg:   "quantity must be positive",
		},
		{
			name:       "Empty Platform",
			platform:   "",
			platformID: "twitch-alice",
			itemName:   domain.ItemLootbox1,
			quantity:   1,
			expected:   domain.ErrInvalidInput,
			errorMsg:   "platform and platformID cannot be empty",
		},
		{
			name:       "Empty PlatformID",
			platform:   domain.PlatformTwitch,
			platformID: "",
			itemName:   domain.ItemLootbox1,
			quantity:   1,
			expected:   domain.ErrInvalidInput,
			errorMsg:   "platform and platformID cannot be empty",
		},
		{
			name:       "Invalid Platform",
			platform:   "invalid-platform",
			platformID: "some-id",
			itemName:   domain.ItemLootbox1,
			quantity:   1,
			expected:   domain.ErrInvalidPlatform,
			errorMsg:   "invalid platform",
		},
		{
			name:       "Empty Item Name",
			platform:   domain.PlatformTwitch,
			platformID: "twitch-alice",
			itemName:   "",
			quantity:   1,
			expected:   domain.ErrInvalidInput,
			errorMsg:   "item name cannot be empty",
		},
	}

	for _, tt := range tests {
		tt := tt // capture loop var
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.DisassembleItem(ctx, tt.platform, tt.platformID, "alice", tt.itemName, tt.quantity)
			assert.Error(t, err)
			assert.ErrorIs(t, err, tt.expected)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestGetRecipe_InputValidation(t *testing.T) {
	t.Parallel()
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name     string
		itemName string
		expected error
		errorMsg string
	}{
		{
			name:     "Empty Item Name",
			itemName: "",
			expected: domain.ErrInvalidInput,
			errorMsg: "item name cannot be empty",
		},
	}

	for _, tt := range tests {
		tt := tt // capture loop var
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.GetRecipe(ctx, tt.itemName, domain.PlatformTwitch, "twitch-alice", "alice")
			assert.Error(t, err)
			assert.ErrorIs(t, err, tt.expected)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestGetUnlockedRecipes_InputValidation(t *testing.T) {
	t.Parallel()
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name       string
		platform   string
		platformID string
		expected   error
		errorMsg   string
	}{
		{
			name:       "Empty Platform",
			platform:   "",
			platformID: "twitch-alice",
			expected:   domain.ErrInvalidInput,
			errorMsg:   "platform and platformID cannot be empty",
		},
		{
			name:       "Empty PlatformID",
			platform:   domain.PlatformTwitch,
			platformID: "",
			expected:   domain.ErrInvalidInput,
			errorMsg:   "platform and platformID cannot be empty",
		},
		{
			name:       "Invalid Platform",
			platform:   "invalid-platform",
			platformID: "some-id",
			expected:   domain.ErrInvalidPlatform,
			errorMsg:   "invalid platform",
		},
	}

	for _, tt := range tests {
		tt := tt // capture loop var
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.GetUnlockedRecipes(ctx, tt.platform, tt.platformID, "alice")
			assert.Error(t, err)
			assert.ErrorIs(t, err, tt.expected)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

// Phase 4: Transaction Failure Tests

func TestUpgradeItem_TransactionFailures(t *testing.T) {
	t.Parallel()

	t.Run("BeginTx Failure", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add materials
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID1, Quantity: 10}, // lootbox_tier0
			},
		}

		// Get original inventory state
		originalInv, _ := repo.GetInventory(ctx, "user-alice")
		originalQuantity := originalInv.Slots[0].Quantity

		// Inject BeginTx error
		repo.Lock()
		repo.shouldFailBeginTx = true
		repo.Unlock()

		// Attempt upgrade
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")

		// Verify inventory unchanged
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after BeginTx failure")
	})

	t.Run("GetInventory Failure", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add materials
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID1, Quantity: 10},
			},
		}

		originalQuantity := repo.inventories["user-alice"].Slots[0].Quantity

		// Inject GetInventory error
		repo.Lock()
		repo.shouldFailGetInventory = true
		repo.Unlock()

		// Attempt upgrade
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get inventory")

		// Verify inventory unchanged (rollback should have occurred)
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after GetInventory failure")
	})

	t.Run("UpdateInventory Failure", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add materials
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID1, Quantity: 10},
			},
		}

		originalQuantity := repo.inventories["user-alice"].Slots[0].Quantity

		// Inject UpdateInventory error
		repo.Lock()
		repo.shouldFailUpdateInventory = true
		repo.Unlock()

		// Attempt upgrade
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update inventory")

		// Verify inventory unchanged (rollback should have occurred)
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after UpdateInventory failure")
	})

	t.Run("Commit Failure", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add materials
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID1, Quantity: 10},
			},
		}

		// Inject Commit error
		repo.Lock()
		repo.shouldFailCommit = true
		repo.Unlock()

		// Attempt upgrade
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")

		// Note: Our mock doesn't support true transaction isolation, so inventory changes
		// persist even on commit failure. In a real database, the transaction would be rolled back.
		// The important thing is that the service correctly returns the error.
		repo.ResetErrorFlags()
	})
}

func TestDisassembleItem_TransactionFailures(t *testing.T) {
	t.Parallel()

	t.Run("BeginTx Failure", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add item to disassemble
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID2, Quantity: 5}, // lootbox_tier1
			},
		}

		originalQuantity := repo.inventories["user-alice"].Slots[0].Quantity

		// Inject BeginTx error
		repo.Lock()
		repo.shouldFailBeginTx = true
		repo.Unlock()

		// Attempt disassemble
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")

		// Verify inventory unchanged
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after BeginTx failure")
	})

	t.Run("GetInventory Failure", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add item
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID2, Quantity: 5},
			},
		}

		originalQuantity := repo.inventories["user-alice"].Slots[0].Quantity

		// Inject GetInventory error
		repo.Lock()
		repo.shouldFailGetInventory = true
		repo.Unlock()

		// Attempt disassemble
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get inventory")

		// Verify inventory unchanged
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after GetInventory failure")
	})

	t.Run("UpdateInventory Failure", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add item
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID2, Quantity: 5},
			},
		}

		originalQuantity := repo.inventories["user-alice"].Slots[0].Quantity

		// Inject UpdateInventory error
		repo.Lock()
		repo.shouldFailUpdateInventory = true
		repo.Unlock()

		// Attempt disassemble
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update inventory")

		// Verify inventory unchanged
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after UpdateInventory failure")
	})

	t.Run("Commit Failure", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add item
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID2, Quantity: 5},
			},
		}

		// Inject Commit error
		repo.Lock()
		repo.shouldFailCommit = true
		repo.Unlock()

		// Attempt disassemble
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")

		// Note: Our mock doesn't support true transaction isolation, so inventory changes
		// persist even on commit failure. In a real database, the transaction would be rolled back.
		// The important thing is that the service correctly returns the error.
		repo.ResetErrorFlags()
	})
}

// Phase 6: Multi-Material Recipe Tests

func TestUpgradeItem_MultiMaterialRecipe(t *testing.T) {
	t.Parallel()

	t.Run("Both Materials Available", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // Prevent masterwork
		ctx := context.Background()

		// Create a recipe requiring 2 materials: 2x lootbox0 + 1x lootbox1 -> lootbox2
		repo.Lock()
		repo.recipes[99] = &domain.Recipe{
			ID:           99,
			TargetItemID: TestItemID3, // lootbox_tier2
			BaseCost: []domain.RecipeCost{
				{ItemID: TestItemID1, Quantity: 2}, // 2x lootbox_tier0
				{ItemID: TestItemID2, Quantity: 1}, // 1x lootbox_tier1
			},
		}
		repo.Unlock()

		// Unlock the recipe
		repo.UnlockRecipe(ctx, "user-alice", 99)

		// Give user both materials
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID1, Quantity: 10}, // lootbox_tier0
				{ItemID: TestItemID2, Quantity: 5},  // lootbox_tier1
			},
		}

		// Craft 2 items
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox2, 2)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify both materials consumed correctly
		inv, _ := repo.GetInventory(ctx, "user-alice")
		lootbox0Slot := -1
		lootbox1Slot := -1
		lootbox2Slot := -1
		for i, slot := range inv.Slots {
			if slot.ItemID == TestItemID1 {
				lootbox0Slot = i
			} else if slot.ItemID == TestItemID2 {
				lootbox1Slot = i
			} else if slot.ItemID == TestItemID3 {
				lootbox2Slot = i
			}
		}
		assert.Equal(t, 6, inv.Slots[lootbox0Slot].Quantity, "should consume 4x lootbox0 (2 per craft)")
		assert.Equal(t, 3, inv.Slots[lootbox1Slot].Quantity, "should consume 2x lootbox1 (1 per craft)")
		assert.NotEqual(t, -1, lootbox2Slot, "should have created lootbox2")
		assert.Equal(t, 2, inv.Slots[lootbox2Slot].Quantity, "should create 2x lootbox2")
	})

	t.Run("Limited By Scarcest Material", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // Prevent masterwork
		ctx := context.Background()

		// Create a recipe requiring 2 materials
		repo.Lock()
		repo.recipes[99] = &domain.Recipe{
			ID:           99,
			TargetItemID: TestItemID3,
			BaseCost: []domain.RecipeCost{
				{ItemID: TestItemID1, Quantity: 2}, // 2x lootbox_tier0
				{ItemID: TestItemID2, Quantity: 1}, // 1x lootbox_tier1
			},
		}
		repo.Unlock()
		repo.UnlockRecipe(ctx, "user-alice", 99)

		// Give user materials where lootbox1 is the bottleneck
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID1, Quantity: 100}, // Plenty of lootbox0
				{ItemID: TestItemID2, Quantity: 2},   // Only 2 lootbox1 (bottleneck)
			},
		}

		// Request 10 crafts, but should only do 2 (limited by lootbox1)
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox2, 10)
		assert.NoError(t, err)
		assert.Equal(t, 2, result.Quantity, "should be limited by scarcest material (lootbox1)")
	})

	t.Run("One Material Missing", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // Prevent masterwork
		ctx := context.Background()

		// Create a recipe requiring 2 materials
		repo.Lock()
		repo.recipes[99] = &domain.Recipe{
			ID:           99,
			TargetItemID: TestItemID3,
			BaseCost: []domain.RecipeCost{
				{ItemID: TestItemID1, Quantity: 2}, // 2x lootbox_tier0
				{ItemID: TestItemID2, Quantity: 1}, // 1x lootbox_tier1
			},
		}
		repo.Unlock()
		repo.UnlockRecipe(ctx, "user-alice", 99)

		// Give user only one material
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID1, Quantity: 10}, // Has lootbox0
				// Missing lootbox1
			},
		}

		// Should fail due to missing material
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox2, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInsufficientQuantity)
	})
}

// Phase 7: Multi-Output Disassemble Tests

func TestDisassembleItem_MultipleOutputs(t *testing.T) {
	t.Parallel()

	t.Run("Multiple Outputs Added Correctly", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // No perfect salvage
		ctx := context.Background()

		// Create disassemble recipe with multiple outputs: lootbox2 -> 2x lootbox0 + 1x lootbox1
		repo.Lock()
		repo.disassembleRecipes[99] = &domain.DisassembleRecipe{
			ID:               99,
			SourceItemID:     TestItemID3, // lootbox_tier2
			QuantityConsumed: 1,
			Outputs: []domain.RecipeOutput{
				{ItemID: TestItemID1, Quantity: 2}, // 2x lootbox_tier0
				{ItemID: TestItemID2, Quantity: 1}, // 1x lootbox_tier1
			},
		}
		repo.recipeAssociations[99] = 1 // Associate with upgrade recipe 1
		repo.Unlock()
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Give user the item to disassemble
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID3, Quantity: 2}, // 2x lootbox_tier2
			},
		}

		// Disassemble 2 items
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox2, 2)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.QuantityProcessed)
		assert.False(t, result.IsPerfectSalvage)

		// Verify all outputs in result map
		assert.Contains(t, result.Outputs, domain.ItemLootbox0)
		assert.Contains(t, result.Outputs, domain.ItemLootbox1)
		assert.Equal(t, 4, result.Outputs[domain.ItemLootbox0], "should get 4x lootbox0 (2 per disassemble)")
		assert.Equal(t, 2, result.Outputs[domain.ItemLootbox1], "should get 2x lootbox1 (1 per disassemble)")
	})

	t.Run("Perfect Salvage Applied To All Outputs", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 0.0 } // Always perfect salvage
		ctx := context.Background()

		// Create disassemble recipe with multiple outputs
		repo.Lock()
		repo.disassembleRecipes[99] = &domain.DisassembleRecipe{
			ID:               99,
			SourceItemID:     TestItemID3,
			QuantityConsumed: 1,
			Outputs: []domain.RecipeOutput{
				{ItemID: TestItemID1, Quantity: 2}, // 2x lootbox_tier0
				{ItemID: TestItemID2, Quantity: 1}, // 1x lootbox_tier1
			},
		}
		repo.recipeAssociations[99] = 1
		repo.Unlock()
		repo.UnlockRecipe(ctx, "user-alice", 1)

		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: TestItemID3, Quantity: 1},
			},
		}

		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox2, 1)
		assert.NoError(t, err)
		assert.True(t, result.IsPerfectSalvage)
		assert.Equal(t, PerfectSalvageMultiplier, result.Multiplier)

		// Verify perfect salvage multiplier applied to all outputs
		// For lootbox0: base 2 * 1.5 = 3 (ceil)
		// For lootbox1: base 1 * 1.5 = 2 (ceil)
		assert.Equal(t, 3, result.Outputs[domain.ItemLootbox0], "perfect salvage should apply 1.5x multiplier to lootbox0")
		assert.Equal(t, 2, result.Outputs[domain.ItemLootbox1], "perfect salvage should apply 1.5x multiplier to lootbox1")
	})
}

// Phase 8: Integration with other services

func TestUpgradeItem_WithXP(t *testing.T) {
	t.Parallel()

	t.Run("Awards XP on Success", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)

		// Setup mock job service
		mockJob := &MockJobService{
			calls: []struct {
				UserID   string
				JobKey   string
				Amount   int
				Source   string
				Metadata map[string]interface{}
			}{},
		}

		svc := NewService(repo, mockJob, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // No masterwork
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 2},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
		assert.NoError(t, err)

		// Wait for async operations
		svc.Shutdown(ctx)

		// Assert
		assert.NotEmpty(t, mockJob.calls)
		assert.Equal(t, "user-alice", mockJob.calls[0].UserID)
		assert.Equal(t, "blacksmith", mockJob.calls[0].JobKey)
		assert.Greater(t, mockJob.calls[0].Amount, 0)
	})
}

func TestUpgradeItem_WithProgression(t *testing.T) {
	t.Parallel()

	t.Run("Applies Masterwork Modifier", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)

		mockProg := &MockProgressionService{
			modifiers: map[string]float64{
				"crafting_success_rate": 1.0, // 100% chance
			},
		}

		svc := NewService(repo, nil, nil, nil, mockProg, nil).(*service)
		svc.rnd = func() float64 { return 0.5 } // Would fail base 10%, but passes 100%
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.NoError(t, err)
		assert.True(t, result.IsMasterwork)
	})
}

func TestUpgradeItem_WithNamingResolution(t *testing.T) {
	t.Parallel()

	t.Run("Resolves Public Name", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)

		mockNaming := &MockNamingResolver{
			publicToInternal: map[string]string{
				"junkbox": domain.ItemLootbox1,
			},
		}

		svc := NewService(repo, nil, nil, mockNaming, nil, nil)
		ctx := context.Background()

		// We need 1 lootbox0 to make 1 lootbox1
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act using public name "junkbox"
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", "junkbox", 1)
		assert.NoError(t, err)
		assert.Equal(t, domain.ItemLootbox1, result.ItemName)
	})
}

func TestShutdown_WaitsForAsync(t *testing.T) {
	t.Parallel()
	repo := NewMockRepository()
	setupTestData(repo)

	// Setup blocking mock job service
	blockChan := make(chan struct{})
	mockJob := &MockJobService{
		blockChan: blockChan,
		calls: []struct {
			UserID   string
			JobKey   string
			Amount   int
			Source   string
			Metadata map[string]interface{}
		}{},
	}

	svc := NewService(repo, mockJob, nil, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }
	ctx := context.Background()

	// Arrange: Unlock recipe and give items
	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: TestItemID1, Quantity: 2},
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Act: Trigger upgrade which triggers async AwardXP
	_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
	assert.NoError(t, err)

	// Shutdown in a goroutine so we can check if it blocks
	shutdownDone := make(chan struct{})
	go func() {
		_ = svc.Shutdown(ctx)
		close(shutdownDone)
	}()

	// Assert: Shutdown should NOT complete yet because blockChan is open
	select {
	case <-shutdownDone:
		t.Fatal("Shutdown completed before async job finished")
	case <-time.After(100 * time.Millisecond):
		// This is good, it's blocked
	}

	// Release the block
	close(blockChan)

	// Now shutdown should complete
	select {
	case <-shutdownDone:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Shutdown timed out after async job finished")
	}
}

func TestDisassembleItem_QualityInheritance(t *testing.T) {
	t.Parallel()

	t.Run("Inherits Average Quality From Consumed Items", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // No perfect salvage
		ctx := context.Background()

		// Setup: Disassemble Lootbox1 -> Lootbox0
		// We have 2 Lootbox1 items to disassemble (requires 1 per disassemble).
		// Item 1: Quality Common (3)
		// Item 2: Quality Rare (5)
		// Average Quality = (3 + 5) / 2 = 4 (Uncommon)
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID2, Quantity: 1, QualityLevel: domain.QualityCommon},
			{ItemID: TestItemID2, Quantity: 1, QualityLevel: domain.QualityRare},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act: Disassemble 2 items
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
		assert.NoError(t, err)
		assert.Equal(t, 2, result.QuantityProcessed)

		// Assert: Output items should be Uncommon
		inv, _ := repo.GetInventory(ctx, "user-alice")
		found := false
		for _, slot := range inv.Slots {
			if slot.ItemID == TestItemID1 {
				found = true
				assert.Equal(t, domain.QualityUncommon, slot.QualityLevel, "Output should average to UNCOMMON")
			}
		}
		assert.True(t, found, "Should have disassembled into lootbox0")
	})
}

func TestDisassembleItem_WithProgression(t *testing.T) {
	t.Parallel()

	t.Run("Applies Perfect Salvage Modifier", func(t *testing.T) {
		t.Parallel()
		repo := NewMockRepository()
		setupTestData(repo)

		mockProg := &MockProgressionService{
			modifiers: map[string]float64{
				"crafting_success_rate": 1.0, // 100% chance
			},
		}

		svc := NewService(repo, nil, nil, nil, mockProg, nil).(*service)
		svc.rnd = func() float64 { return 0.5 } // Would fail base 10%, but passes 100%
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: TestItemID2, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.NoError(t, err)
		assert.True(t, result.IsPerfectSalvage)

		// Verify progression service was called
		mockProg.mu.Lock()
		defer mockProg.mu.Unlock()
		assert.NotEmpty(t, mockProg.calls)
		assert.Equal(t, "crafting_success_rate", mockProg.calls[0].featureKey)
	})
}
