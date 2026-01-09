package crafting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Sentinel errors for recipe loader
var (
	ErrDuplicateRecipeKey = errors.New("duplicate recipe key")
	ErrInvalidItem        = errors.New("invalid item reference")
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrOrphanedRecipe     = errors.New("orphaned recipe in database")
)

// RecipeConfig represents the complete recipe configuration
type RecipeConfig struct {
	CraftingConfig   *CraftingRecipeConfig
	DisassembleConfig *DisassembleRecipeConfig
}

// CraftingRecipeConfig represents the JSON configuration for crafting recipes
type CraftingRecipeConfig struct {
	Version     string                `json:"version"`
	Description string                `json:"description"`
	Recipes     []CraftingRecipeDef   `json:"recipes"`
}

// DisassembleRecipeConfig represents the JSON configuration for disassemble recipes
type DisassembleRecipeConfig struct {
	Version     string                  `json:"version"`
	Description string                  `json:"description"`
	Recipes     []DisassembleRecipeDef  `json:"recipes"`
}

// CraftingRecipeDef represents a single crafting recipe in the JSON
type CraftingRecipeDef struct {
	RecipeKey  string       `json:"recipe_key"`
	TargetItem string       `json:"target_item"`
	Costs      []RecipeCost `json:"costs"`
}

// DisassembleRecipeDef represents a single disassemble recipe in the JSON
type DisassembleRecipeDef struct {
	RecipeKey        string         `json:"recipe_key"`
	QuantityConsumed int            `json:"quantity_consumed"`
	Outputs          []RecipeOutput `json:"outputs"`
	AssociatedUpgrade string        `json:"associated_upgrade"`
}

// RecipeCost represents a cost item in a recipe
type RecipeCost struct {
	Item     string `json:"item"`
	Quantity int    `json:"quantity"`
}

// RecipeOutput represents an output item from disassembly
type RecipeOutput struct {
	Item     string `json:"item"`
	Quantity int    `json:"quantity"`
}

// RecipeLoader handles loading and validating recipe configuration
type RecipeLoader interface {
	Load(craftingPath, disassemblePath string) (*RecipeConfig, error)
	Validate(config *RecipeConfig, itemRepo repository.Item) error
	SyncToDatabase(ctx context.Context, config *RecipeConfig, craftingRepo repository.Crafting, itemRepo repository.Item, configDir string) (*SyncResult, error)
}

// SyncResult contains the result of syncing recipes to the database
type SyncResult struct {
	CraftingInserted    int
	CraftingUpdated     int
	CraftingSkipped     int
	DisassembleInserted int
	DisassembleUpdated  int
	DisassembleSkipped  int
	OrphanedRecipes     []string
}

type recipeLoader struct{}

// NewRecipeLoader creates a new RecipeLoader instance
func NewRecipeLoader() RecipeLoader {
	return &recipeLoader{}
}

// Load reads and parses both recipe JSON files
func (l *recipeLoader) Load(craftingPath, disassemblePath string) (*RecipeConfig, error) {
	// Load crafting recipes
	craftingData, err := os.ReadFile(craftingPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read crafting config file: %w", err)
	}

	var craftingConfig CraftingRecipeConfig
	if err := json.Unmarshal(craftingData, &craftingConfig); err != nil {
		return nil, fmt.Errorf("failed to parse crafting config: %w", err)
	}

	// Load disassemble recipes
	disassembleData, err := os.ReadFile(disassemblePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read disassemble config file: %w", err)
	}

	var disassembleConfig DisassembleRecipeConfig
	if err := json.Unmarshal(disassembleData, &disassembleConfig); err != nil {
		return nil, fmt.Errorf("failed to parse disassemble config: %w", err)
	}

	return &RecipeConfig{
		CraftingConfig:    &craftingConfig,
		DisassembleConfig: &disassembleConfig,
	}, nil
}

// Validate checks the recipe configuration for errors
func (l *recipeLoader) Validate(config *RecipeConfig, itemRepo repository.Item) error {
	ctx := context.Background()

	if config == nil || config.CraftingConfig == nil || config.DisassembleConfig == nil {
		return fmt.Errorf("%w: config is nil", ErrInvalidConfig)
	}

	// Get all items from database for validation
	items, err := itemRepo.GetAllItems(ctx)
	if err != nil {
		return fmt.Errorf("failed to get items for validation: %w", err)
	}

	itemsByInternalName := make(map[string]bool, len(items))
	for _, item := range items {
		itemsByInternalName[item.InternalName] = true
	}

	// Validate crafting recipes
	craftingKeys := make(map[string]bool)
	for i, recipe := range config.CraftingConfig.Recipes {
		if recipe.RecipeKey == "" {
			return fmt.Errorf("%w: crafting recipe at index %d has empty recipe_key", ErrInvalidConfig, i)
		}

		if craftingKeys[recipe.RecipeKey] {
			return fmt.Errorf("%w: '%s' in crafting recipes", ErrDuplicateRecipeKey, recipe.RecipeKey)
		}
		craftingKeys[recipe.RecipeKey] = true

		// Validate target item exists
		if !itemsByInternalName[recipe.TargetItem] {
			return fmt.Errorf("%w: crafting recipe '%s' references non-existent target_item '%s'", ErrInvalidItem, recipe.RecipeKey, recipe.TargetItem)
		}

		// Validate costs
		if len(recipe.Costs) == 0 {
			return fmt.Errorf("%w: crafting recipe '%s' has no costs", ErrInvalidConfig, recipe.RecipeKey)
		}

		for j, cost := range recipe.Costs {
			if !itemsByInternalName[cost.Item] {
				return fmt.Errorf("%w: crafting recipe '%s' cost[%d] references non-existent item '%s'", ErrInvalidItem, recipe.RecipeKey, j, cost.Item)
			}
			if cost.Quantity <= 0 {
				return fmt.Errorf("%w: crafting recipe '%s' cost[%d] has non-positive quantity", ErrInvalidConfig, recipe.RecipeKey, j)
			}
		}
	}

	// Validate disassemble recipes
	disassembleKeys := make(map[string]bool)
	for i, recipe := range config.DisassembleConfig.Recipes {
		if recipe.RecipeKey == "" {
			return fmt.Errorf("%w: disassemble recipe at index %d has empty recipe_key", ErrInvalidConfig, i)
		}

		if disassembleKeys[recipe.RecipeKey] {
			return fmt.Errorf("%w: '%s' in disassemble recipes", ErrDuplicateRecipeKey, recipe.RecipeKey)
		}
		disassembleKeys[recipe.RecipeKey] = true

		// Validate recipe_key (the item being disassembled) exists
		if !itemsByInternalName[recipe.RecipeKey] {
			return fmt.Errorf("%w: disassemble recipe '%s' references non-existent source item", ErrInvalidItem, recipe.RecipeKey)
		}

		// Validate quantity consumed
		if recipe.QuantityConsumed <= 0 {
			return fmt.Errorf("%w: disassemble recipe '%s' has non-positive quantity_consumed", ErrInvalidConfig, recipe.RecipeKey)
		}

		// Validate outputs
		if len(recipe.Outputs) == 0 {
			return fmt.Errorf("%w: disassemble recipe '%s' has no outputs", ErrInvalidConfig, recipe.RecipeKey)
		}

		for j, output := range recipe.Outputs {
			if !itemsByInternalName[output.Item] {
				return fmt.Errorf("%w: disassemble recipe '%s' output[%d] references non-existent item '%s'", ErrInvalidItem, recipe.RecipeKey, j, output.Item)
			}
			if output.Quantity <= 0 {
				return fmt.Errorf("%w: disassemble recipe '%s' output[%d] has non-positive quantity", ErrInvalidConfig, recipe.RecipeKey, j)
			}
		}

		// Validate associated_upgrade exists in crafting recipes
		if recipe.AssociatedUpgrade != "" && !craftingKeys[recipe.AssociatedUpgrade] {
			return fmt.Errorf("%w: disassemble recipe '%s' references non-existent associated_upgrade '%s'", ErrInvalidItem, recipe.RecipeKey, recipe.AssociatedUpgrade)
		}
	}

	return nil
}

// SyncToDatabase syncs the recipe configuration to the database idempotently
func (l *recipeLoader) SyncToDatabase(ctx context.Context, config *RecipeConfig, craftingRepo repository.Crafting, itemRepo repository.Item, configDir string) (*SyncResult, error) {
	log := logger.FromContext(ctx)

	// Check if files have changed since last sync
	craftingPath := configDir + "crafting.json"
	disassemblePath := configDir + "disassemble.json"

	craftingChanged, err := hasFileChanged(ctx, itemRepo, craftingPath, "recipes_crafting.json")
	if err != nil {
		return nil, fmt.Errorf("failed to check crafting file change: %w", err)
	}

	disassembleChanged, err := hasFileChanged(ctx, itemRepo, disassemblePath, "recipes_disassemble.json")
	if err != nil {
		return nil, fmt.Errorf("failed to check disassemble file change: %w", err)
	}

	if !craftingChanged && !disassembleChanged {
		log.Info("Recipe config files unchanged, skipping sync")
		return &SyncResult{}, nil
	}

	result := &SyncResult{
		OrphanedRecipes: make([]string, 0),
	}

	// Get all items for ID lookup
	items, err := itemRepo.GetAllItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}

	itemIDsByInternalName := make(map[string]int, len(items))
	for _, item := range items {
		itemIDsByInternalName[item.InternalName] = item.ID
	}

	// Sync crafting recipes first
	if craftingChanged {
		craftingResult, err := l.syncCraftingRecipes(ctx, config.CraftingConfig, craftingRepo, itemIDsByInternalName)
		if err != nil {
			return nil, fmt.Errorf("failed to sync crafting recipes: %w", err)
		}
		result.CraftingInserted = craftingResult.Inserted
		result.CraftingUpdated = craftingResult.Updated
		result.CraftingSkipped = craftingResult.Skipped
		result.OrphanedRecipes = append(result.OrphanedRecipes, craftingResult.Orphaned...)

		// Update sync metadata for crafting
		if err := updateSyncMetadata(ctx, itemRepo, craftingPath, "recipes_crafting.json"); err != nil {
			log.Warn("Failed to update crafting sync metadata", "error", err)
		}
	}

	// Sync disassemble recipes second
	if disassembleChanged {
		disassembleResult, err := l.syncDisassembleRecipes(ctx, config, craftingRepo, itemIDsByInternalName)
		if err != nil {
			return nil, fmt.Errorf("failed to sync disassemble recipes: %w", err)
		}
		result.DisassembleInserted = disassembleResult.Inserted
		result.DisassembleUpdated = disassembleResult.Updated
		result.DisassembleSkipped = disassembleResult.Skipped
		result.OrphanedRecipes = append(result.OrphanedRecipes, disassembleResult.Orphaned...)

		// Update sync metadata for disassemble
		if err := updateSyncMetadata(ctx, itemRepo, disassemblePath, "recipes_disassemble.json"); err != nil {
			log.Warn("Failed to update disassemble sync metadata", "error", err)
		}
	}

	// Log orphaned recipes
	if len(result.OrphanedRecipes) > 0 {
		log.Warn("Found orphaned recipes in database (in DB but not in config)", "count", len(result.OrphanedRecipes), "recipes", result.OrphanedRecipes)
	}

	log.Info("Recipe sync completed",
		"crafting_inserted", result.CraftingInserted,
		"crafting_updated", result.CraftingUpdated,
		"crafting_skipped", result.CraftingSkipped,
		"disassemble_inserted", result.DisassembleInserted,
		"disassemble_updated", result.DisassembleUpdated,
		"disassemble_skipped", result.DisassembleSkipped)

	return result, nil
}

type recipeSyncResult struct {
	Inserted int
	Updated  int
	Skipped  int
	Orphaned []string
}

// syncCraftingRecipes syncs crafting recipes to the database
func (l *recipeLoader) syncCraftingRecipes(ctx context.Context, config *CraftingRecipeConfig, repo repository.Crafting, itemIDs map[string]int) (*recipeSyncResult, error) {
	log := logger.FromContext(ctx)
	result := &recipeSyncResult{
		Orphaned: make([]string, 0),
	}

	// Get existing recipes
	existing, err := repo.GetAllCraftingRecipes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing crafting recipes: %w", err)
	}

	existingByKey := make(map[string]*domain.Recipe, len(existing))
	for i := range existing {
		existingByKey[existing[i].RecipeKey] = &existing[i]
	}

	// Track which keys we've seen from config
	seenKeys := make(map[string]bool)

	// Process each recipe from config
	for _, recipeDef := range config.Recipes {
		seenKeys[recipeDef.RecipeKey] = true

		targetItemID := itemIDs[recipeDef.TargetItem]

		// Convert costs to domain format
		costs := make([]domain.RecipeCost, len(recipeDef.Costs))
		for i, cost := range recipeDef.Costs {
			costs[i] = domain.RecipeCost{
				ItemID:   itemIDs[cost.Item],
				Quantity: cost.Quantity,
			}
		}

		if existingRecipe, ok := existingByKey[recipeDef.RecipeKey]; ok {
			// Recipe exists - check if update needed
			needsUpdate := existingRecipe.TargetItemID != targetItemID || !costsEqual(existingRecipe.BaseCost, costs)

			if needsUpdate {
				// Update existing recipe
				if err := repo.UpdateCraftingRecipe(ctx, existingRecipe.ID, &domain.Recipe{
					RecipeKey:    recipeDef.RecipeKey,
					TargetItemID: targetItemID,
					BaseCost:     costs,
				}); err != nil {
					return nil, fmt.Errorf("failed to update crafting recipe '%s': %w", recipeDef.RecipeKey, err)
				}
				result.Updated++
				log.Info("Updated crafting recipe", "recipe_key", recipeDef.RecipeKey)
			} else {
				result.Skipped++
			}
		} else {
			// Insert new recipe
			recipeID, err := repo.InsertCraftingRecipe(ctx, &domain.Recipe{
				RecipeKey:    recipeDef.RecipeKey,
				TargetItemID: targetItemID,
				BaseCost:     costs,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to insert crafting recipe '%s': %w", recipeDef.RecipeKey, err)
			}
			result.Inserted++
			log.Info("Inserted crafting recipe", "recipe_key", recipeDef.RecipeKey, "id", recipeID)
		}
	}

	// Find orphaned recipes
	for key := range existingByKey {
		if !seenKeys[key] {
			result.Orphaned = append(result.Orphaned, "crafting:"+key)
		}
	}

	return result, nil
}

// syncDisassembleRecipes syncs disassemble recipes to the database
func (l *recipeLoader) syncDisassembleRecipes(ctx context.Context, config *RecipeConfig, repo repository.Crafting, itemIDs map[string]int) (*recipeSyncResult, error) {
	log := logger.FromContext(ctx)
	result := &recipeSyncResult{
		Orphaned: make([]string, 0),
	}

	// Get existing recipes
	existing, err := repo.GetAllDisassembleRecipes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing disassemble recipes: %w", err)
	}

	existingByKey := make(map[string]*domain.DisassembleRecipe, len(existing))
	for i := range existing {
		existingByKey[existing[i].RecipeKey] = &existing[i]
	}

	// Get all crafting recipes for association lookup
	craftingRecipes, err := repo.GetAllCraftingRecipes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get crafting recipes for associations: %w", err)
	}

	craftingIDsByKey := make(map[string]int, len(craftingRecipes))
	for _, recipe := range craftingRecipes {
		craftingIDsByKey[recipe.RecipeKey] = recipe.ID
	}

	// Track which keys we've seen from config
	seenKeys := make(map[string]bool)

	// Process each recipe from config
	for _, recipeDef := range config.DisassembleConfig.Recipes {
		seenKeys[recipeDef.RecipeKey] = true

		sourceItemID := itemIDs[recipeDef.RecipeKey]

		// Convert outputs to domain format
		outputs := make([]domain.RecipeOutput, len(recipeDef.Outputs))
		for i, output := range recipeDef.Outputs {
			outputs[i] = domain.RecipeOutput{
				ItemID:   itemIDs[output.Item],
				Quantity: output.Quantity,
			}
		}

		if existingRecipe, ok := existingByKey[recipeDef.RecipeKey]; ok {
			// Recipe exists - check if update needed
			needsUpdate := existingRecipe.SourceItemID != sourceItemID ||
				existingRecipe.QuantityConsumed != recipeDef.QuantityConsumed ||
				!outputsEqual(existingRecipe.Outputs, outputs)

			if needsUpdate {
				// Update existing recipe
				if err := repo.UpdateDisassembleRecipe(ctx, existingRecipe.ID, &domain.DisassembleRecipe{
					RecipeKey:        recipeDef.RecipeKey,
					SourceItemID:     sourceItemID,
					QuantityConsumed: recipeDef.QuantityConsumed,
				}); err != nil {
					return nil, fmt.Errorf("failed to update disassemble recipe '%s': %w", recipeDef.RecipeKey, err)
				}

				// Update outputs
				if err := repo.ClearDisassembleOutputs(ctx, existingRecipe.ID); err != nil {
					return nil, fmt.Errorf("failed to clear outputs for recipe '%s': %w", recipeDef.RecipeKey, err)
				}

				for _, output := range outputs {
					if err := repo.InsertDisassembleOutput(ctx, existingRecipe.ID, output); err != nil {
						return nil, fmt.Errorf("failed to insert output for recipe '%s': %w", recipeDef.RecipeKey, err)
					}
				}

				result.Updated++
				log.Info("Updated disassemble recipe", "recipe_key", recipeDef.RecipeKey)
			} else {
				result.Skipped++
			}

			// Handle association
			if recipeDef.AssociatedUpgrade != "" {
				if upgradeID, ok := craftingIDsByKey[recipeDef.AssociatedUpgrade]; ok {
					if err := repo.UpsertRecipeAssociation(ctx, upgradeID, existingRecipe.ID); err != nil {
						return nil, fmt.Errorf("failed to upsert association for '%s': %w", recipeDef.RecipeKey, err)
					}
				}
			}
		} else {
			// Insert new recipe
			recipeID, err := repo.InsertDisassembleRecipe(ctx, &domain.DisassembleRecipe{
				RecipeKey:        recipeDef.RecipeKey,
				SourceItemID:     sourceItemID,
				QuantityConsumed: recipeDef.QuantityConsumed,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to insert disassemble recipe '%s': %w", recipeDef.RecipeKey, err)
			}

			// Insert outputs
			for _, output := range outputs {
				if err := repo.InsertDisassembleOutput(ctx, recipeID, output); err != nil {
					return nil, fmt.Errorf("failed to insert output for recipe '%s': %w", recipeDef.RecipeKey, err)
				}
			}

			// Handle association
			if recipeDef.AssociatedUpgrade != "" {
				if upgradeID, ok := craftingIDsByKey[recipeDef.AssociatedUpgrade]; ok {
					if err := repo.UpsertRecipeAssociation(ctx, upgradeID, recipeID); err != nil {
						return nil, fmt.Errorf("failed to upsert association for '%s': %w", recipeDef.RecipeKey, err)
					}
				}
			}

			result.Inserted++
			log.Info("Inserted disassemble recipe", "recipe_key", recipeDef.RecipeKey, "id", recipeID)
		}
	}

	// Find orphaned recipes
	for key := range existingByKey {
		if !seenKeys[key] {
			result.Orphaned = append(result.Orphaned, "disassemble:"+key)
		}
	}

	return result, nil
}

// Helper functions

func hasFileChanged(ctx context.Context, repo repository.Item, configPath, metadataName string) (bool, error) {
	// Get file info
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return false, fmt.Errorf("failed to stat config file: %w", err)
	}

	// Calculate file hash
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, fmt.Errorf("failed to read config file: %w", err)
	}

	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	// Get last sync metadata
	syncMeta, err := repo.GetSyncMetadata(ctx, metadataName)
	if err != nil {
		// First sync - no metadata exists
		return true, nil
	}

	// Compare hash and mod time
	if syncMeta.FileHash != fileHash || !syncMeta.FileModTime.Equal(fileInfo.ModTime()) {
		return true, nil
	}

	return false, nil
}

func updateSyncMetadata(ctx context.Context, repo repository.Item, configPath, metadataName string) error {
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	return repo.UpsertSyncMetadata(ctx, &domain.SyncMetadata{
		ConfigName:   metadataName,
		LastSyncTime: time.Now(),
		FileHash:     fileHash,
		FileModTime:  fileInfo.ModTime(),
	})
}

func costsEqual(a, b []domain.RecipeCost) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].ItemID != b[i].ItemID || a[i].Quantity != b[i].Quantity {
			return false
		}
	}

	return true
}

func outputsEqual(a, b []domain.RecipeOutput) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].ItemID != b[i].ItemID || a[i].Quantity != b[i].Quantity {
			return false
		}
	}

	return true
}
