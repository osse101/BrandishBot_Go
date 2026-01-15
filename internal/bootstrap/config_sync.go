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
	slog.Info(LogMsgSyncingProgressionTree)
	treeLoader := progression.NewTreeLoader()

	treeConfig, err := treeLoader.Load(config.ConfigPathProgressionTree)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrMsgFailedLoadProgressionTree, err)
	}

	if err := treeLoader.Validate(treeConfig); err != nil {
		return fmt.Errorf("%s: %w", ErrMsgInvalidProgressionTree, err)
	}

	syncResult, err := treeLoader.SyncToDatabase(ctx, treeConfig, progressionRepo, config.ConfigPathProgressionTree)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrMsgFailedSyncProgressionTree, err)
	}

	if syncResult.NodesInserted > 0 || syncResult.NodesUpdated > 0 || syncResult.AutoUnlocked > 0 {
		slog.Info(LogMsgProgressionTreeSynced,
			"inserted", syncResult.NodesInserted,
			"updated", syncResult.NodesUpdated,
			"skipped", syncResult.NodesSkipped,
			"auto_unlocked", syncResult.AutoUnlocked)
	} else {
		slog.Info(LogMsgProgressionTreeUnchanged)
	}

	return nil
}

// SyncItems loads, validates, and syncs the items configuration to database.
// It handles the complete lifecycle: load JSON → validate → sync to DB → log results.
// Uses intelligent hash-based change detection to skip sync if file is unchanged.
// Returns the item repository for use in subsequent sync operations.
func SyncItems(ctx context.Context, dbPool *pgxpool.Pool) (repository.Item, error) {
	slog.Info(LogMsgSyncingItems)
	itemLoader := item.NewLoader()
	itemRepo := postgres.NewItemRepository(dbPool)

	itemConfig, err := itemLoader.Load(config.ConfigPathItems)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrMsgFailedLoadItems, err)
	}

	if err := itemLoader.Validate(itemConfig); err != nil {
		return nil, fmt.Errorf("%s: %w", ErrMsgInvalidItems, err)
	}

	itemSyncResult, err := itemLoader.SyncToDatabase(ctx, itemConfig, itemRepo, config.ConfigPathItems)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrMsgFailedSyncItems, err)
	}

	if itemSyncResult.ItemsInserted > 0 || itemSyncResult.ItemsUpdated > 0 {
		slog.Info(LogMsgItemsSynced,
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
	slog.Info(LogMsgSyncingRecipes)
	recipeLoader := crafting.NewRecipeLoader()

	recipeConfig, err := recipeLoader.Load(config.ConfigPathRecipesCrafting, config.ConfigPathRecipesDisassemble)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrMsgFailedLoadRecipes, err)
	}

	if err := recipeLoader.Validate(recipeConfig, itemRepo); err != nil {
		return fmt.Errorf("%s: %w", ErrMsgInvalidRecipes, err)
	}

	recipeSyncResult, err := recipeLoader.SyncToDatabase(ctx, recipeConfig, craftingRepo, itemRepo, config.ConfigPathRecipesDir)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrMsgFailedSyncRecipes, err)
	}

	if recipeSyncResult.CraftingInserted > 0 || recipeSyncResult.DisassembleInserted > 0 ||
		recipeSyncResult.CraftingUpdated > 0 || recipeSyncResult.DisassembleUpdated > 0 {
		slog.Info(LogMsgRecipesSynced,
			"crafting_inserted", recipeSyncResult.CraftingInserted,
			"crafting_updated", recipeSyncResult.CraftingUpdated,
			"disassemble_inserted", recipeSyncResult.DisassembleInserted,
			"disassemble_updated", recipeSyncResult.DisassembleUpdated)
	}

	return nil
}
