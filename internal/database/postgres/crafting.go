package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// CraftingRepository implements the crafting repository for PostgreSQL
type CraftingRepository struct {
	db *pgxpool.Pool
	q  *generated.Queries
	// Helpers for shared logic could be injected or just duplicated if simple
}

// NewCraftingRepository creates a new CraftingRepository
func NewCraftingRepository(db *pgxpool.Pool) *CraftingRepository {
	return &CraftingRepository{
		db: db,
		q:  generated.New(db),
	}
}

// CraftingTx implements repository.CraftingTx
type CraftingTx struct {
	tx pgx.Tx
	q  *generated.Queries
}

// BeginTx starts a new transaction
func (r *CraftingRepository) BeginTx(ctx context.Context) (repository.CraftingTx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &CraftingTx{
		tx: tx,
		q:  r.q.WithTx(tx),
	}, nil
}

// Commit commits the transaction
func (t *CraftingTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction
func (t *CraftingTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// GetUserByPlatformID implementation
func (r *CraftingRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	row, err := r.q.GetUserByPlatformID(ctx, generated.GetUserByPlatformIDParams{
		Name:           platform,
		PlatformUserID: platformID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user core data: %w", err)
	}
	return mapUserAndLinks(ctx, r.q, row.UserID, row.Username)
}

// GetItemByName retrieves an item by its internal name
func (r *CraftingRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	row, err := r.q.GetItemByName(ctx, itemName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Return nil if item not found
		}
		return nil, fmt.Errorf("failed to get item by name: %w", err)
	}

	return &domain.Item{
		ID:             int(row.ItemID),
		InternalName:   row.InternalName,
		PublicName:     row.PublicName.String,
		DefaultDisplay: row.DefaultDisplay.String,
		Description:    row.ItemDescription.String,
		BaseValue:      int(row.BaseValue.Int32),
		Handler:        textToPtr(row.Handler),
		Types:          row.Types,
	}, nil
}

// GetItemByID retrieves an item by its ID
func (r *CraftingRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	row, err := r.q.GetItemByID(ctx, int32(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get item by id: %w", err)
	}

	return &domain.Item{
		ID:             int(row.ItemID),
		InternalName:   row.InternalName,
		PublicName:     row.PublicName.String,
		DefaultDisplay: row.DefaultDisplay.String,
		Description:    row.ItemDescription.String,
		BaseValue:      int(row.BaseValue.Int32),
		Handler:        textToPtr(row.Handler),
		Types:          row.Types,
	}, nil
}

// GetItemsByIDs retrieves multiple items by their IDs
func (r *CraftingRepository) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	if len(itemIDs) == 0 {
		return []domain.Item{}, nil
	}

	ids := make([]int32, len(itemIDs))
	for i, id := range itemIDs {
		ids[i] = int32(id)
	}

	rows, err := r.q.GetItemsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get items by ids: %w", err)
	}

	var items []domain.Item
	for _, row := range rows {
		items = append(items, domain.Item{
			ID:             int(row.ItemID),
			InternalName:   row.InternalName,
			PublicName:     row.PublicName.String,
			DefaultDisplay: row.DefaultDisplay.String,
			Description:    row.ItemDescription.String,
			BaseValue:      int(row.BaseValue.Int32),
			Handler:        textToPtr(row.Handler),
			Types:          row.Types,
		})
	}
	return items, nil
}

// GetInventory retrieves the user's inventory
func (r *CraftingRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return getInventory(ctx, r.q, userID)
}

// UpdateInventory updates the user's inventory
func (r *CraftingRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return updateInventory(ctx, r.q, userID, inventory)
}

// GetInventory for Tx
func (t *CraftingTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return getInventory(ctx, t.q, userID)
}

// UpdateInventory for Tx
func (t *CraftingTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return updateInventory(ctx, t.q, userID, inventory)
}

// GetRecipeByTargetItemID retrieves a recipe by its target item ID
func (r *CraftingRepository) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	row, err := r.q.GetRecipeByTargetItemID(ctx, int32(itemID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get recipe by target item id: %w", err)
	}

	recipe := domain.Recipe{
		ID:           int(row.RecipeID),
		TargetItemID: int(row.TargetItemID),
		CreatedAt:    row.CreatedAt.Time,
	}

	if len(row.BaseCost) > 0 {
		if err := json.Unmarshal(row.BaseCost, &recipe.BaseCost); err != nil {
			return nil, fmt.Errorf("failed to unmarshal base cost: %w", err)
		}
	} else {
		recipe.BaseCost = []domain.RecipeCost{}
	}

	return &recipe, nil
}

// IsRecipeUnlocked checks if a user has unlocked a specific recipe
func (r *CraftingRepository) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user id: %w", err)
	}
	return r.q.IsRecipeUnlocked(ctx, generated.IsRecipeUnlockedParams{
		UserID:   userUUID,
		RecipeID: int32(recipeID),
	})
}

// UnlockRecipe unlocks a recipe for a user
func (r *CraftingRepository) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	err = r.q.UnlockRecipe(ctx, generated.UnlockRecipeParams{
		UserID:   userUUID,
		RecipeID: int32(recipeID),
	})
	if err != nil {
		return fmt.Errorf("failed to unlock recipe: %w", err)
	}
	return nil
}

// GetUnlockedRecipesForUser retrieves all recipes unlocked by a specific user
func (r *CraftingRepository) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]repository.UnlockedRecipeInfo, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := r.q.GetUnlockedRecipesForUser(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unlocked recipes: %w", err)
	}

	var recipes []repository.UnlockedRecipeInfo
	for _, row := range rows {
		recipes = append(recipes, repository.UnlockedRecipeInfo{
			ItemName: row.ItemName,
			ItemID:   int(row.ItemID),
		})
	}
	return recipes, nil
}

// GetAllRecipes retrieves all crafting recipes
func (r *CraftingRepository) GetAllRecipes(ctx context.Context) ([]repository.RecipeListItem, error) {
	rows, err := r.q.GetAllRecipes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all recipes: %w", err)
	}

	var recipes []repository.RecipeListItem
	for _, row := range rows {
		recipes = append(recipes, repository.RecipeListItem{
			ItemName:    row.ItemName,
			ItemID:      int(row.ItemID),
			Description: row.ItemDescription.String,
		})
	}
	return recipes, nil
}

// GetDisassembleRecipeBySourceItemID retrieves a disassemble recipe for a given source item
func (r *CraftingRepository) GetDisassembleRecipeBySourceItemID(ctx context.Context, itemID int) (*domain.DisassembleRecipe, error) {
	row, err := r.q.GetDisassembleRecipeBySourceItemID(ctx, int32(itemID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query disassemble recipe: %w", err)
	}

	recipe := domain.DisassembleRecipe{
		ID:               int(row.RecipeID),
		SourceItemID:     int(row.SourceItemID),
		QuantityConsumed: int(row.QuantityConsumed),
		CreatedAt:        row.CreatedAt.Time,
	}

	outputs, err := r.q.GetDisassembleOutputs(ctx, row.RecipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query disassemble outputs: %w", err)
	}

	for _, out := range outputs {
		recipe.Outputs = append(recipe.Outputs, domain.RecipeOutput{
			ItemID:   int(out.ItemID),
			Quantity: int(out.Quantity),
		})
	}

	return &recipe, nil
}

// GetAssociatedUpgradeRecipeID retrieves the upgrade recipe ID associated with a disassemble recipe
func (r *CraftingRepository) GetAssociatedUpgradeRecipeID(ctx context.Context, disassembleRecipeID int) (int, error) {
	id, err := r.q.GetAssociatedUpgradeRecipeID(ctx, int32(disassembleRecipeID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("no associated upgrade recipe found for disassemble recipe %d", disassembleRecipeID)
		}
		return 0, fmt.Errorf("failed to query associated upgrade recipe: %w", err)
	}
	return int(id), nil
}
