package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/config"
	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/database/postgres"
	"github.com/osse101/BrandishBot_Go/internal/item"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// SyncProgressionTree loads, validates, and syncs the progression tree configuration to database.
// It handles the complete lifecycle: load JSON → validate → sync to DB → log results.
// Uses intelligent hash-based change detection to skip sync if file is unchanged.
func SyncProgressionTree(ctx context.Context, progressionRepo repository.Progression) error {
	slog.Info("Syncing progression tree from JSON config...")
	treeLoader := progression.NewTreeLoader()

	treeConfig, err := treeLoader.Load(config.ConfigPathProgressionTree)
	if err != nil {
		return fmt.Errorf("failed to load progression tree config: %w", err)
	}

	if err := treeLoader.Validate(treeConfig); err != nil {
		return fmt.Errorf("invalid progression tree config: %w", err)
	}

	syncResult, err := treeLoader.SyncToDatabase(ctx, treeConfig, progressionRepo, config.ConfigPathProgressionTree)
	if err != nil {
		return fmt.Errorf("failed to sync progression tree to database: %w", err)
	}

	if syncResult.NodesInserted > 0 || syncResult.NodesUpdated > 0 || syncResult.AutoUnlocked > 0 {
		slog.Info("Progression tree synced successfully",
			"inserted", syncResult.NodesInserted,
			"updated", syncResult.NodesUpdated,
			"skipped", syncResult.NodesSkipped,
			"auto_unlocked", syncResult.AutoUnlocked)
	} else {
		slog.Info("Progression tree config unchanged, sync skipped")
	}

	return nil
}

// SyncItems loads, validates, and syncs the items configuration to database.
// It handles the complete lifecycle: load JSON → validate → sync to DB → log results.
// Uses intelligent hash-based change detection to skip sync if file is unchanged.
// Returns the item repository for use in subsequent sync operations.
func SyncItems(ctx context.Context, dbPool *pgxpool.Pool) (repository.Item, error) {
	slog.Info("Syncing items from JSON config...")
	itemLoader := item.NewLoader()
	itemRepo := postgres.NewItemRepository(dbPool)

	itemConfig, err := itemLoader.Load(config.ConfigPathItems)
	if err != nil {
		return nil, fmt.Errorf("failed to load items config: %w", err)
	}

	if err := itemLoader.Validate(itemConfig); err != nil {
		return nil, fmt.Errorf("invalid items config: %w", err)
	}

	itemSyncResult, err := itemLoader.SyncToDatabase(ctx, itemConfig, itemRepo, config.ConfigPathItems)
	if err != nil {
		return nil, fmt.Errorf("failed to sync items to database: %w", err)
	}

	if itemSyncResult.ItemsInserted > 0 || itemSyncResult.ItemsUpdated > 0 {
		slog.Info("Items synced successfully",
			"inserted", itemSyncResult.ItemsInserted,
			"updated", itemSyncResult.ItemsUpdated,
			"skipped", itemSyncResult.ItemsSkipped)
	}

	return itemRepo, nil
}

// SyncRecipes loads, validates, and syncs the crafting and disassemble recipes to database.
// It handles the complete lifecycle: load JSONs → validate → sync to DB → log results.
// Uses intelligent hash-based change detection to skip sync if files are unchanged.
// Requires itemRepo for validating recipe item references.
func SyncRecipes(ctx context.Context, craftingRepo repository.Crafting, itemRepo repository.Item) error {
	slog.Info("Syncing recipes from JSON config...")
	recipeLoader := crafting.NewRecipeLoader()

	recipeConfig, err := recipeLoader.Load(config.ConfigPathRecipesCrafting, config.ConfigPathRecipesDisassemble)
	if err != nil {
		return fmt.Errorf("failed to load recipe config: %w", err)
	}

	if err := recipeLoader.Validate(recipeConfig, itemRepo); err != nil {
		return fmt.Errorf("invalid recipe configuration: %w", err)
	}

	recipeSyncResult, err := recipeLoader.SyncToDatabase(ctx, recipeConfig, craftingRepo, itemRepo, config.ConfigPathRecipesDir)
	if err != nil {
		return fmt.Errorf("failed to sync recipes to database: %w", err)
	}

	if recipeSyncResult.CraftingInserted > 0 || recipeSyncResult.DisassembleInserted > 0 ||
		recipeSyncResult.CraftingUpdated > 0 || recipeSyncResult.DisassembleUpdated > 0 {
		slog.Info("Recipes synced successfully",
			"crafting_inserted", recipeSyncResult.CraftingInserted,
			"crafting_updated", recipeSyncResult.CraftingUpdated,
			"disassemble_inserted", recipeSyncResult.DisassembleInserted,
			"disassemble_updated", recipeSyncResult.DisassembleUpdated)
	}

	return nil
}
