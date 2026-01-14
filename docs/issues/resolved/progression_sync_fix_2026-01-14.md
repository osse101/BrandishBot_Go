# Progression Tree Sync Fix - 2026-01-14

## Issues Resolved

### 1. Tree Loader Not Syncing on Config File Changes

**Symptom:** Log message "Progression tree config file unchanged, skipping sync" appeared even after editing `configs/progression_tree.json`

**Root Cause:** The `configs` directory was not mounted as a volume in the app container. The app was reading a stale copy baked into the Docker image at build time.

**Evidence:**
- Host file: Hash `0667f115...`, Modified 2026-01-14, Size 7472 bytes
- Container file: Hash `3200dc35...`, Modified 2026-01-11, Size 6893 bytes
- Discord container had the volume mount, but app container didn't

**Fix:** Added volume mount to `docker-compose.yml`:

```yaml
app:
  volumes:
    - ./logs:/app/logs
    - ./configs:/app/configs:ro  # <-- Added this line
```

### 2. Existing Nodes Missing New Structure Fields

**Symptom:** All nodes in database had `tier=0`, `size=NULL`, `category=NULL` even though the schema supported these fields

**Root Causes:**
1. The `needsUpdate` comparison in `internal/progression/tree_loader.go:287-291` didn't check tier, size, or category
2. The `insertNode()` and `updateNode()` functions had these fields commented out (lines 355-358, 383-386)

**Fixes:**

**File: `internal/progression/tree_loader.go`**

1. Updated comparison logic (line 286-294):
```go
if existing, ok := existingByKey[nodeConfig.Key]; ok {
    needsUpdate := existing.DisplayName != nodeConfig.Name ||
        existing.Description != nodeConfig.Description ||
        existing.MaxLevel != nodeConfig.MaxLevel ||
        existing.SortOrder != nodeConfig.SortOrder ||
        existing.NodeType != nodeConfig.Type ||
        existing.Tier != nodeConfig.Tier ||        // Added
        existing.Size != nodeConfig.Size ||        // Added
        existing.Category != nodeConfig.Category   // Added
```

2. Uncommented fields in `insertNode()` (line 347-358):
```go
return inserter.InsertNode(ctx, &domain.ProgressionNode{
    NodeKey:     config.Key,
    NodeType:    config.Type,
    DisplayName: config.Name,
    Description: config.Description,
    MaxLevel:    config.MaxLevel,
    UnlockCost:  unlockCost,
    SortOrder:   config.SortOrder,
    Tier:        config.Tier,      // Uncommented
    Size:        config.Size,      // Uncommented
    Category:    config.Category,  // Uncommented
})
```

3. Uncommented fields in `updateNode()` (line 374-385):
```go
return updater.UpdateNode(ctx, nodeID, &domain.ProgressionNode{
    NodeKey:     config.Key,
    NodeType:    config.Type,
    DisplayName: config.Name,
    Description: config.Description,
    MaxLevel:    config.MaxLevel,
    UnlockCost:  unlockCost,
    SortOrder:   config.SortOrder,
    Tier:        config.Tier,      // Uncommented
    Size:        config.Size,      // Uncommented
    Category:    config.Category,  // Uncommented
})
```

## Verification Results

### Initial State (Before Fixes)
```sql
SELECT node_key, tier, size, category FROM progression_nodes LIMIT 3;
```
```
      node_key      | tier | size | category
--------------------+------+------+----------
 progression_system |    0 |      |
 item_money         |    0 |      |
 item_lootbox0      |    0 |      |
```

### After Fixes + Rebuild
```
Progression tree sync completed: inserted=0 updated=18 skipped=0 auto_unlocked=0
```

```sql
SELECT node_key, tier, size, category, unlock_cost FROM progression_nodes ORDER BY sort_order LIMIT 6;
```
```
      node_key       | tier |  size  | category | unlock_cost
---------------------+------+--------+----------+-------------
 progression_system  |    0 | medium | core     |           0
 item_money          |    1 | small  | items    |         500
 item_lootbox0       |    1 | small  | items    |         500
 feature_economy     |    1 | large  | economy  |        2000
 feature_buy         |    2 | small  | economy  |        1000
 feature_sell        |    2 | small  | economy  |        1000
```

### Sync Detection Test
Changed `item_money` description in config file, restarted app:

```
Updated progression node: key=item_money
Progression tree sync completed: inserted=0 updated=1 skipped=17 auto_unlocked=0
```

✅ Only the changed node was updated (1 updated, 17 skipped)

## Database Schema Confirmation

The schema was already correct (no migration needed):

```sql
-- From migrations/0001_initial_schema_v1.sql:214-216
CREATE TABLE progression_nodes (
    ...
    tier integer DEFAULT 1 NOT NULL,
    size character varying(20) DEFAULT 'medium'::character varying NOT NULL,
    category character varying(50) DEFAULT 'uncategorized'::character varying NOT NULL
);
```

SQLC queries also correctly handled these fields in:
- `internal/database/queries/progression.sql` (lines 21-28)
- Domain model in `internal/domain/progression.go` (lines 16-18)

## Related Files Modified

1. `docker-compose.yml` - Added configs volume mount
2. `internal/progression/tree_loader.go` - Fixed comparison + uncommented fields

## Lessons Learned

1. **Volume Mounts Are Critical:** Config files must be mounted as volumes for live-reload to work
2. **Check All Comparison Fields:** When adding new fields to a domain model, update the `needsUpdate` logic
3. **Don't Leave TODOs in Production Code:** The commented-out fields should have been uncommented when the schema migration was applied
4. **Verify Both Insert and Update Paths:** Node insertion worked, but updates didn't - both code paths needed fixing

## Testing Commands

```bash
# Check current sync status
docker exec brandishbot_go-db-1 bash -c 'psql -U $POSTGRES_USER -d $POSTGRES_DB -c "SELECT config_name, last_sync_time FROM config_sync_metadata WHERE config_name = '\''progression_tree.json'\'';"'

# Verify node structure
docker exec brandishbot_go-db-1 bash -c 'psql -U $POSTGRES_USER -d $POSTGRES_DB -c "SELECT node_key, tier, size, category FROM progression_nodes LIMIT 5;"'

# Check file hash in container vs host
docker exec brandishbot_go-app-1 sha256sum configs/progression_tree.json
sha256sum configs/progression_tree.json

# Force sync by touching file
touch configs/progression_tree.json && docker restart brandishbot_go-app-1
```
