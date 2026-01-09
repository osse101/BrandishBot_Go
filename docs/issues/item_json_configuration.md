# JSON-Based Item Configuration System

**Status**: Open  
**Priority**: Medium  
**Category**: Feature Enhancement  
**Created**: 2026-01-09

## Problem

Currently, items are defined solely in database migrations (SQL). This makes it difficult to:
- Add new items without writing SQL migrations
- modify item properties (tags, display names)
- Visualize all available items in one place
- Maintain consistency between environments

This is similar to the problem solved for progression nodes (`progression_tree.json`) and proposed for recipes (`recipes/*.json`).

## Proposed Solution

Create a `configs/items.json` file to serve as the source of truth for all items in the game. A sync mechanism will update the database on startup.

### Reference Implementations

- **Progression Tree**: [`configs/progression_tree.json`](file:///home/osse1/projects/BrandishBot_Go/configs/progression_tree.json) and [`internal/progression/tree_loader.go`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/tree_loader.go)
- **Recipe Config**: Proposed in [`docs/issues/recipe_json_configuration.md`](file:///home/osse1/projects/BrandishBot_Go/docs/issues/recipe_json_configuration.md)

## Proposed JSON Structure

### `configs/items.json`

```json
{
  "version": "1.0",
  "description": "Item definitions for BrandishBot",
  "items": [
    {
      "internal_name": "money",
      "public_name": "money",
      "description": "Standard currency",
      "tier": 0,
      "max_stack": 1000000,
      "tags": ["currency", "tradeable"],
      "default_display": "💰"
    },
    {
      "internal_name": "lootbox_tier0",
      "public_name": "junkbox",
      "description": "A dusty old box containing basic items",
      "tier": 0,
      "max_stack": 100,
      "tags": ["consumable", "tradeable", "can_open", "upgradeable"],
      "default_display": "A dingy box"
    },
    {
      "internal_name": "lootbox_tier1",
      "public_name": "lootbox",
      "description": "Standard lootbox with decent rewards",
      "tier": 1,
      "max_stack": 100,
      "tags": ["consumable", "tradeable", "can_open", "upgradeable", "disassembleable"],
      "default_display": "A sturdy chest"
    }
  ]
}
```

## Implementation Plan

### 1. Create Item Loader (`internal/item/loader.go`)

Create a loader service that follows the `TreeLoader` pattern:

```go
type ItemLoader interface {
    Load(path string) (*ItemConfig, error)
    Validate(config *ItemConfig) error
    SyncToDatabase(ctx context.Context, config *ItemConfig, repo repository.Item) (*SyncResult, error)
}
```

### 2. Define Configuration Structs

```go
type ItemConfig struct {
    Version     string       `json:"version"`
    Description string       `json:"description"`
    Items       []ItemDef    `json:"items"`
}

type ItemDef struct {
    InternalName   string   `json:"internal_name"`
    PublicName     string   `json:"public_name"`
    Description    string   `json:"description"`
    Tier           int      `json:"tier"`
    MaxStack       int      `json:"max_stack"`
    Tags           []string `json:"tags"` // Maps to item_types/tags in DB
    DefaultDisplay string   `json:"default_display"` // Emoji or icon
}
```

### 3. Implement Sync Logic

The `SyncToDatabase` method should:
1. Load all existing items from the DB
2. Iterate through `items.json`:
   - If item doesn't exist → **Insert** into `items` table
   - If item exists → **Update** properties (public_name, description, etc.)
   - Sync tags: Clear existing tags and insert from `Tags` list (or smarter diff)
3. Return statistics (Inserted, Updated, Skipped)

**Tag Handling**:
You will need to ensure referenced tags exist in the `item_types` table or create them if missing.

### 4. Main Initialization

Add to `cmd/app/main.go`:

```go
if cfg.SyncItems {
    itemLoader := item.NewLoader()
    itemConfig, err := itemLoader.Load("configs/items.json")
    if err != nil {
        // handle error
    }
    
    result, err := itemLoader.SyncToDatabase(ctx, itemConfig, itemRepo)
    slog.Info("Items synced", "inserted", result.Inserted, "updated", result.Updated)
}
```

## Database Considerations

- **Table**: `items`
- **Columns**: `internal_name` (Unique Key), `public_name`, `description`, `rarity`, `tier`, `max_stack`, `default_display`
- **Tags Table**: `item_type_assignments` linking to `item_types`

## Benefits

✅ **Centralized Config**: All items in one readable file  
✅ **Easy Updates**: Change public names or descriptions without SQL  
✅ **Tag Management**: Easily see and modify item behavior tags  
✅ **Version Control**: Track item evolution in Git  
✅ **Validation**: Catch duplicate names or invalid values at startup  

## Related Files

- Reference: [`configs/progression_tree.json`](file:///home/osse1/projects/BrandishBot_Go/configs/progression_tree.json)
- Reference: [`internal/progression/tree_loader.go`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/tree_loader.go)
- Domain: [`internal/domain/item.go`](file:///home/osse1/projects/BrandishBot_Go/internal/domain/item.go)
- Migration: [`migrations/0001_initial_schema_v1.sql`](file:///home/osse1/projects/BrandishBot_Go/migrations/0001_initial_schema_v1.sql) (Item tables)
