# JSON-Based Progression Tree Configuration

## Problem

Currently, the progression tree is defined via SQL migrations, which makes it:
- Hard to visualize the full tree structure
- Difficult to modify (requires new migration)
- Error-prone (manual ID references, easy to break parent/child relationships)
- Impossible to version control meaningfully (just SQL dumps)

**Example of Current Approach:**
```sql
-- Migration 0015
INSERT INTO progression_nodes (node_key, unlock_cost, parent_node_id) 
VALUES ('feature_upgrade', 50, 12);  -- What's 12? Have to look it up!
```

## Proposed Solution

Define the progression tree in a **JSON configuration file** that gets loaded/validated on server startup.

### File Location
`configs/progression_tree.json`

### Example Structure

```json
{
  "version": "1.0",
  "description": "BrandishBot Progression Tree",
  "nodes": [
    {
      "key": "progression_system",
      "name": "Progression System",
      "description": "Unlock the community progression system",
      "unlock_cost": 50,
      "max_level": 1,
      "parent": null,
      "auto_unlock": true,
      "children": [
        "item_money",
        "feature_economy"
      ]
    },
    {
      "key": "item_money",
      "name": "Money System",
      "description": "Enable currency for the economy",
      "unlock_cost": 100,
      "max_level": 1,
      "parent": "progression_system",
      "children": [
        "feature_economy"
      ]
    },
    {
      "key": "feature_economy",
      "name": "Economy Features",
      "description": "Unlock buy/sell commands",
      "unlock_cost": 150,
      "max_level": 1,
      "parent": "item_money",
      "children": [
        "feature_upgrade",
        "feature_sell"
      ]
    },
    {
      "key": "feature_upgrade",
      "name": "Upgrade System",
      "description": "Allow upgrading items",
      "unlock_cost": 200,
      "max_level": 3,
      "parent": "feature_economy",
      "children": []
    }
  ]
}
```

## Benefits

1. **Visual Structure**: Easy to see parent-child relationships
2. **Validation**: Can validate tree structure (no cycles, valid parents, etc.) before applying
3. **Version Control**: Meaningful git diffs when tree changes
4. **Documentation**: Self-documenting with descriptions
5. **Tooling**: Can generate visualizations (Graphviz, Mermaid diagrams)
6. **Flexibility**: Can add metadata (icons, colors, categories) without schema changes

## Implementation Plan

### Phase 1: Loader
- Create `internal/progression/tree_loader.go`
- Parse JSON file
- Validate structure (no cycles, all parents exist, unique keys)
- Convert to database inserts (idempotent)

### Phase 2: Migration
- Keep existing migrations for backward compatibility
- Add new migration that reads JSON and populates DB
- Tool to export current DB tree to JSON (one-time migration helper)

### Phase 3: Validation
- Startup check: Load JSON, compare to DB, warn if mismatch
- Admin command: `/progression reload-tree` (dev only)

### Phase 4: Features
- Support for `auto_unlock: true` (skips voting)
- Support for `hidden: true` (admin-only unlocks)
- Support for `requires_all_children: true` (AND vs OR logic)

## Example Code

```go
type TreeConfig struct {
    Version     string `json:"version"`
    Description string `json:"description"`
    Nodes       []NodeConfig `json:"nodes"`
}

type NodeConfig struct {
    Key         string   `json:"key"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    UnlockCost  int      `json:"unlock_cost"`
    MaxLevel    int      `json:"max_level"`
    Parent      *string  `json:"parent"`      // null for root
    Children    []string `json:"children"`
    AutoUnlock  bool     `json:"auto_unlock"` // Skip voting
}

func LoadTreeConfig(path string) (*TreeConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var config TreeConfig
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, err
    }
    
    // Validate
    if err := validateTree(&config); err != nil {
        return nil, fmt.Errorf("invalid tree: %w", err)
    }
    
    return &config, nil
}

func validateTree(config *TreeConfig) error {
    // Check for duplicate keys
    // Check all parent references exist
    // Check for cycles
    // Check unlock costs are positive
    return nil
}
```

## Migration Strategy

1. Export current DB tree to JSON (one-time script)
2. Review/edit JSON for clarity
3. Create new migration that uses JSON loader
4. Deploy with both old migrations (for schema) + JSON loader
5. Future changes only update JSON

## Tooling Ideas

### Visualization
```bash
$ make progression-tree-diagram
# Generates docs/progression_tree.svg using Graphviz
```

### Validation
```bash
$ make progression-tree-validate
# Checks JSON for errors before deploy
```

### Diff Viewer
```bash
$ make progression-tree-diff
# Shows what changed between DB and JSON
```

## Priority

**High** - Current tree is hard to maintain, this will save significant time and reduce errors

## Files to Create

- `configs/progression_tree.json` - The tree definition
- `internal/progression/tree_loader.go` - Loader and validator
- `internal/progression/tree_loader_test.go` - Tests
- `scripts/export_tree_to_json.go` - One-time migration helper
- `migrations/00XX_load_tree_from_json.sql` - Calls Go loader

## Related Issues

- Auto-skip single option votes (can use `auto_unlock` flag)
- Tree visualization (easier with JSON)
- Admin tree editing UI (can edit JSON via API)
