package crafting

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// StubRepository is a minimal, high-performance mock for benchmarks
// It avoids map lookups where possible and allocates minimally
type StubRepository struct {
	recipes []*domain.Recipe
}

func (r *StubRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return &domain.User{ID: "bench-user", Username: "bench"}, nil
}
func (r *StubRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	return &domain.Item{ID: 1, InternalName: itemName}, nil
}
func (r *StubRepository) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	return r.recipes[0], nil
}
func (r *StubRepository) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	return true, nil
}
func (r *StubRepository) BeginTx(ctx context.Context) (repository.CraftingTx, error) {
	return &StubTx{repo: r}, nil
}
func (r *StubRepository) GetDisassembleRecipeBySourceItemID(ctx context.Context, itemID int) (*domain.DisassembleRecipe, error) {
	return &domain.DisassembleRecipe{
		ID:               1,
		SourceItemID:     1,
		QuantityConsumed: 1,
		Outputs: []domain.RecipeOutput{
			{ItemID: 2, Quantity: 1},
		},
	}, nil
}
func (r *StubRepository) GetAssociatedUpgradeRecipeID(ctx context.Context, disassembleRecipeID int) (int, error) {
	return 1, nil
}
func (r *StubRepository) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	return []domain.Item{{ID: 2, InternalName: "material"}}, nil
}

// Unused interface methods stubbed
func (r *StubRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) { return nil, nil }
func (r *StubRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return nil, nil
}
func (r *StubRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return nil
}
func (r *StubRepository) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	return nil
}
func (r *StubRepository) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]repository.UnlockedRecipeInfo, error) {
	return nil, nil
}
func (r *StubRepository) GetAllRecipes(ctx context.Context) ([]repository.RecipeListItem, error) {
	return nil, nil
}

type StubTx struct {
	repo *StubRepository
}

func (tx *StubTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	// Return inventory with plenty of materials
	return &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 10000},
			{ItemID: 2, Quantity: 10000},
		},
	}, nil
}
func (tx *StubTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return nil
}
func (tx *StubTx) Commit(ctx context.Context) error { return nil }
func (tx *StubTx) Rollback(ctx context.Context) error { return nil }

// Unused Tx methods
func (tx *StubTx) UpsertUser(ctx context.Context, user *domain.User) error { return nil }
func (tx *StubTx) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	return nil, nil
}
func (tx *StubTx) UpdateUser(ctx context.Context, user domain.User) error   { return nil }
func (tx *StubTx) DeleteUser(ctx context.Context, userID string) error      { return nil }
func (tx *StubTx) DeleteInventory(ctx context.Context, userID string) error { return nil }
func (tx *StubTx) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	return tx.repo.GetItemsByIDs(ctx, itemIDs)
}
func (tx *StubTx) GetSellablePrices(ctx context.Context) ([]domain.Item, error) { return nil, nil }
func (tx *StubTx) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	return false, nil
}
func (tx *StubTx) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	return nil, nil
}
func (tx *StubTx) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	return nil
}
func (tx *StubTx) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
	return nil
}
func (tx *StubTx) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return tx.repo.GetUserByPlatformID(ctx, platform, platformID)
}

func BenchmarkUpgradeItem(b *testing.B) {
	repo := &StubRepository{
		recipes: []*domain.Recipe{
			{
				ID:           1,
				TargetItemID: 1,
				BaseCost: []domain.RecipeCost{
					{ItemID: 2, Quantity: 1},
				},
			},
		},
	}
	svc := NewService(repo, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 } // Disable masterwork
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.UpgradeItem(ctx, "twitch", "123", "bench", "target_item", 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDisassembleItem(b *testing.B) {
	repo := &StubRepository{
		recipes: []*domain.Recipe{{}}, // Placeholder
	}
	svc := NewService(repo, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 } // Disable perfect salvage
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.DisassembleItem(ctx, "twitch", "123", "bench", "source_item", 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}
