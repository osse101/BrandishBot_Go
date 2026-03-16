package crafting

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestAutoUnlockRecipe(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)

	mockProg := &MockProgressionService{}
	mockJobs := NewMockJobService()
	mockNaming := &MockNamingResolver{
		publicToInternal: map[string]string{"junkbox": "lootbox_tier0"},
	}

	svc := NewService(repo, &MockEventPublisher{}, mockNaming, mockProg, mockJobs).(*service)
	ctx := context.Background()

	// 1. Setup Tier 0 Item (Junkbox)
	junkboxID := 100
	junkboxTargetID := 101

	repo.items["lootbox_tier0"] = &domain.Item{ID: junkboxID, InternalName: "lootbox_tier0", PublicName: "junkbox"}
	repo.itemsByID[junkboxID] = repo.items["lootbox_tier0"]
	repo.items["lootbox_tier1"] = &domain.Item{ID: junkboxTargetID, InternalName: "lootbox_tier1", PublicName: "decent lootbox"}
	repo.itemsByID[junkboxTargetID] = repo.items["lootbox_tier1"]

	// Create recipe with IsAutoUnlock = true
	repo.recipes[100] = &domain.Recipe{
		ID:           100,
		RecipeKey:    "lootbox_tier0",
		TargetItemID: junkboxTargetID,
		IsAutoUnlock: true,
		BaseCost: []domain.RecipeCost{
			{ItemID: junkboxID, Quantity: 1},
		},
	}

	// Setup user
	userID := "user-alice"
	repo.users["alice"] = &domain.User{ID: userID, Username: "alice", TwitchID: "twitch-alice"}
	repo.inventories[userID] = &domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: junkboxID, Quantity: 5},
	}}

	// ACT 1: Try upgrade BEFORE unlocking feature_upgrade
	mockJobs.SetFeatureUnlocked(userID, "feature_upgrade", false)
	_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", "junkbox", 1)

	// ASSERT 1: Should fail with feature locked
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires job progression")
	assert.ErrorIs(t, err, domain.ErrFeatureLocked)

	// ACT 2: Unlock feature_upgrade and try upgrade (recipe is auto-unlocked)
	mockJobs.SetFeatureUnlocked(userID, "feature_upgrade", true)
	result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", "junkbox", 1)

	// ASSERT 2: Should succeed because recipe is auto-unlocked
	require.NoError(t, err)
	assert.Equal(t, 1, result.Quantity)

	// Verify it's actually not in the unlocked_recipes table
	unlocked, _ := repo.IsRecipeUnlocked(ctx, userID, 100)
	assert.True(t, unlocked, "Should be considered unlocked because of IsAutoUnlock flag")

	repo.RLock()
	_, foundInTable := repo.unlockedRecipes[userID][100]
	repo.RUnlock()
	assert.False(t, foundInTable, "Should not be present in the manual unlock table")
}

func TestDisassembleAutoUnlock(t *testing.T) {
	// ARRANGE
	repo := NewMockRepository()
	setupTestData(repo)

	mockProg := &MockProgressionService{}
	mockJobs := NewMockJobService()
	mockNaming := &MockNamingResolver{
		publicToInternal: map[string]string{"junkbox": "lootbox_tier0"},
	}

	svc := NewService(repo, &MockEventPublisher{}, mockNaming, mockProg, mockJobs).(*service)
	ctx := context.Background()

	// Setup Tier 0 Item
	itemID := 100
	repo.items["lootbox_tier0"] = &domain.Item{ID: itemID, InternalName: "lootbox_tier0", PublicName: "junkbox"}
	repo.itemsByID[itemID] = repo.items["lootbox_tier0"]

	// Create upgrade recipe with IsAutoUnlock = true
	upgradeID := 100
	repo.recipes[upgradeID] = &domain.Recipe{
		ID:           upgradeID,
		RecipeKey:    "lootbox_tier0",
		IsAutoUnlock: true,
	}

	// Create disassemble recipe
	disassembleID := 200
	repo.disassembleRecipes[disassembleID] = &domain.DisassembleRecipe{
		ID:               disassembleID,
		SourceItemID:     itemID,
		RecipeKey:        "lootbox_tier0",
		QuantityConsumed: 1,
		Outputs: []domain.RecipeOutput{
			{ItemID: 1, Quantity: 1}, // gives ItemID 1
		},
	}
	repo.recipeAssociations[disassembleID] = upgradeID

	// Setup user
	userID := "user-alice"
	repo.users["alice"] = &domain.User{ID: userID, Username: "alice", TwitchID: "twitch-alice"}
	repo.inventories[userID] = &domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: itemID, Quantity: 5},
	}}

	// ACT 1: Try disassemble WITHOUT feature_disassemble
	mockJobs.SetFeatureUnlocked(userID, "feature_disassemble", false)
	_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", "junkbox", 1)

	// ASSERT 1: Should fail
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires job progression")

	// ACT 2: Unlock feature_disassemble and try disassemble
	mockJobs.SetFeatureUnlocked(userID, "feature_disassemble", true)
	result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", "junkbox", 1)

	// ASSERT 2: Should succeed because associated upgrade is auto-unlocked
	require.NoError(t, err)
	assert.Equal(t, 1, result.QuantityProcessed)
}
