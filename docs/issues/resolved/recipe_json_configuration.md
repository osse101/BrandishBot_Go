# JSON-Based Recipe Configuration System

**Status**: RESOLVED  
**Priority**: Medium  
**Category**: Feature Enhancement  
**Created**: 2026-01-09  
**Resolved**: 2026-01-09

## Problem

Currently, crafting recipes (upgrade and disassemble) are stored directly in the database, making them difficult to:
- View and understand at a glance
- Version control and track changes
- Modify without writing SQL
- Share and document
- Sync across environments

This is similar to the problem that was solved for progression nodes with `progression_tree.json`.

## Proposed Solution

Create JSON configuration files for recipes with automatic database synchronization:
- `configs/recipes/upgrades.json` - Upgrade/crafting recipes
- `configs/recipes/disassembles.json` - Disassemble recipes

### Reference Implementation

Follow the **progression tree pattern** already implemented:

**Config File**: [`configs/progression_tree.json`](file:///home/osse1/projects/BrandishBot_Go/configs/progression_tree.json)
- JSON structure with version, description, and array of nodes
- Each node has key, name, type, description, prerequisites, etc.
- Support for modifiers and configuration options

**Loader**: [`internal/progression/tree_loader.go`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/tree_loader.go)
- `Load()` - Reads and parses JSON
- `Validate()` - Validates structure and business rules
- `SyncToDatabase()` - Idempotent sync to database (insert/update/skip)

**Initialization**: [`cmd/app/main.go:168-195`](file:///home/osse1/projects/BrandishBot_Go/cmd/app/main.go#L168-L195)
```go
if cfg.SyncProgressionTree {
    treeLoader := progression.NewTreeLoader()
    treeConfig, err := treeLoader.Load("configs/progression_tree.json")
    // ... validation ...
    syncResult, err := treeLoader.SyncToDatabase(context.Background(), treeConfig, progressionRepo)
    slog.Info("Progression tree synced",
        "inserted", syncResult.NodesInserted,
        "updated", syncResult.NodesUpdated,
        "skipped", syncResult.NodesSkipped)
}
```

## Proposed JSON Structure

### `configs/recipes/upgrades.json`

```json
{
  "version": "1.0",
  "description": "Upgrade/crafting recipes for BrandishBot",
  "recipes": [
    {
      "key": "upgrade_lootbox_tier1",
      "target_item": "lootbox_tier1",
      "description": "Upgrade junkboxes to lootbox",
      "costs": [
        {
          "item": "lootbox_tier0",
          "quantity": 5
        },
        {
          "item": "money",
          "quantity": 100
        }
      ]
    },
    {
      "key": "upgrade_lootbox_tier2",
      "target_item": "lootbox_tier2",
      "description": "Upgrade lootboxes to goldbox",
      "costs": [
        {
          "item": "lootbox_tier1",
          "quantity": 3
        },
        {
          "item": "money",
          "quantity": 500
        }
      ]
    }
  ]
}
```

### `configs/recipes/disassembles.json`

```json
{
  "version": "1.0",
  "description": "Disassemble recipes for BrandishBot",
  "recipes": [
    {
      "key": "disassemble_lootbox_tier1",
      "source_item": "lootbox_tier1",
      "quantity_consumed": 1,
      "description": "Break down lootbox for materials",
      "outputs": [
        {
          "item": "lootbox_tier0",
          "quantity": 1
        },
        {
          "item": "money",
          "quantity": 25
        }
      ],
      "associated_upgrade": "upgrade_lootbox_tier1"
    }
  ]
}
```

## Implementation Plan

### 1. Create Recipe Loader (`internal/crafting/recipe_loader.go`)

Following the `tree_loader.go` pattern:

```go
type RecipeLoader interface {
    LoadUpgrades(path string) (*UpgradeRecipeConfig, error)
    LoadDisassembles(path string) (*DisassembleRecipeConfig, error)
    ValidateUpgrades(config *UpgradeRecipeConfig) error
    ValidateDisassembles(config *DisassembleRecipeConfig) error
    SyncUpgradesToDatabase(ctx context.Context, config *UpgradeRecipeConfig, repo repository.Crafting) (*SyncResult, error)
    SyncDisassemblesToDatabase(ctx context.Context, config *DisassembleRecipeConfig, repo repository.Crafting) (*SyncResult, error)
}
```

### 2. Add Config Structures

```go
type UpgradeRecipeConfig struct {
    Version     string          `json:"version"`
    Description string          `json:"description"`
    Recipes     []UpgradeRecipe `json:"recipes"`
}

type UpgradeRecipe struct {
    Key        string       `json:"key"`
    TargetItem string       `json:"target_item"` // Internal name
    Description string      `json:"description"`
    Costs      []RecipeCost `json:"costs"`
}

type RecipeCost struct {
    Item     string `json:"item"`     // Internal name
    Quantity int    `json:"quantity"`
}
```

### 3. Implement Sync Logic

- Query existing recipes by key/target_item
- Insert new recipes
- Update existing recipes if changed
- Mark orphaned recipes (in DB but not in config)
- **Sync Associations**: Link disassemble recipes to upgrade recipes using `associated_upgrade` field. This writes to the `recipe_associations` table.
   > [!IMPORTANT]
   > Ensure that upgrade recipes are synced BEFORE disassemble recipes to ensure the `associated_upgrade` key works.
- Return sync statistics

### 4. Add to Main Initialization

```go
if cfg.SyncRecipes {
    recipeLoader := crafting.NewRecipeLoader()
    
    // Sync upgrade recipes
    upgradeConfig, err := recipeLoader.LoadUpgrades("configs/recipes/upgrades.json")
    // ... validation ...
    upgradeResult, err := recipeLoader.SyncUpgradesToDatabase(ctx, upgradeConfig, craftingRepo)
    
    // Sync disassemble recipes
    disassembleConfig, err := recipeLoader.LoadDisassembles("configs/recipes/disassembles.json")
    // ... validation ...
    disassembleResult, err := recipeLoader.SyncDisassemblesToDatabase(ctx, disassembleConfig, craftingRepo)
    
    slog.Info("Recipes synced",
        "upgrades_inserted", upgradeResult.RecipesInserted,
        "disassembles_inserted", disassembleResult.RecipesInserted)
}
```

### 5. Add Environment Variable

Add `SYNC_RECIPES=true` to `.env` (similar to `SYNC_PROGRESSION_TREE`)

## Benefits

✅ **Easy to manage** - Edit JSON files instead of writing SQL  
✅ **Version controlled** - Track recipe changes in git  
✅ **Declarative** - Clear structure shows all recipes at once  
✅ **Idempotent** - Safe to run sync multiple times  
✅ **Validated** - Catch errors in config before database ops  
✅ **Documented** - JSON structure is self-documenting  
✅ **Familiar pattern** - Reuses progression tree sync approach

## Related Files

- Reference: [`configs/progression_tree.json`](file:///home/osse1/projects/BrandishBot_Go/configs/progression_tree.json)
- Reference: [`internal/progression/tree_loader.go`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/tree_loader.go)
- Database: [`migrations/0001_initial_schema_v1.sql`](file:///home/osse1/projects/BrandishBot_Go/migrations/0001_initial_schema_v1.sql) (recipe tables)
- Repository: [`internal/database/postgres/crafting.go`](file:///home/osse1/projects/BrandishBot_Go/internal/database/postgres/crafting.go)

## Notes

- Item names in JSON should use **internal names** (e.g., `lootbox_tier0`, not `junkbox`)
- Consider adding a `GetItemByPublicName` helper if you want to allow public names in config
- Sync should validate that all referenced items exist in the database
- Recipe keys should be unique and used for idempotent updates
