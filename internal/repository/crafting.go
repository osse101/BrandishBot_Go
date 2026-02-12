package repository

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Crafting defines the interface for crafting persistence
type Crafting interface {
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	GetItemByName(ctx context.Context, itemName string) (*domain.Item, error)
	GetItemByID(ctx context.Context, id int) (*domain.Item, error)
	GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error)
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
	GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error)
	IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error)
	UnlockRecipe(ctx context.Context, userID string, recipeID int) error
	GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]UnlockedRecipeInfo, error)
	// BeginTx starts a transaction for crafting operations
	BeginTx(ctx context.Context) (CraftingTx, error)

	GetDisassembleRecipeBySourceItemID(ctx context.Context, itemID int) (*domain.DisassembleRecipe, error)
	GetAssociatedUpgradeRecipeID(ctx context.Context, disassembleRecipeID int) (int, error)
	GetAllRecipes(ctx context.Context) ([]RecipeListItem, error)

	// Recipe loader operations
	GetAllCraftingRecipes(ctx context.Context) ([]domain.Recipe, error)
	GetAllDisassembleRecipes(ctx context.Context) ([]domain.DisassembleRecipe, error)
	GetCraftingRecipeByKey(ctx context.Context, recipeKey string) (*domain.Recipe, error)
	GetDisassembleRecipeByKey(ctx context.Context, recipeKey string) (*domain.DisassembleRecipe, error)
	InsertCraftingRecipe(ctx context.Context, recipe *domain.Recipe) (int, error)
	InsertDisassembleRecipe(ctx context.Context, recipe *domain.DisassembleRecipe) (int, error)
	UpdateCraftingRecipe(ctx context.Context, recipeID int, recipe *domain.Recipe) error
	UpdateDisassembleRecipe(ctx context.Context, recipeID int, recipe *domain.DisassembleRecipe) error
	ClearDisassembleOutputs(ctx context.Context, recipeID int) error
	InsertDisassembleOutput(ctx context.Context, recipeID int, output domain.RecipeOutput) error
	UpsertRecipeAssociation(ctx context.Context, upgradeRecipeID, disassembleRecipeID int) error
}

// CraftingTx defines the interface for crafting transactions
type CraftingTx interface {
	Tx
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
}

// Structs needed for the interface (formerly in crafting package)
type UnlockedRecipeInfo struct {
	ItemName string `json:"item_name"`
	ItemID   int    `json:"item_id"`
}

type RecipeListItem struct {
	ItemID           int    `json:"item_id"`
	ItemName         string `json:"item_name"`
	Description      string `json:"description"`
	RequiredJobLevel int    `json:"required_job_level,omitempty"`
}
