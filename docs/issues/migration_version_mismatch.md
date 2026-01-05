# Migration Version Mismatch Issue

**Status**: RESOLVED (Prevention Script Added)
**Priority**: Medium  
**Category**: Database/Infrastructure  
**Created**: 2025-12-31  
**Resolved**: 2025-01-05
**Environment**: All (staging, production, local)

## Summary

The database migration tracking table (`goose_db_version`) contains timestamp-based migration entries (e.g., `20251228162349`) that conflict with numbered migrations (e.g., `0026`, `0027`). This causes the application container to crash on startup with "missing migrations" errors.

## Background

Goose migration tool expects migrations to be numbered sequentially (0001, 0002, etc.) or timestamp-based consistently. Mixing both formats causes goose to be unable to determine which migrations have been applied.

## Symptoms

### Container Crash Loop
```
goose run: error: found 2 missing migrations:
    version 26: migrations/0026_tune_progression_system.sql
    version 27: migrations/0027_inventory_filters.sql
ERROR: Migrations failed after 3 attempts
```

### Application Never Starts
- Container continuously restarts
- Old code keeps running (if any containers were already running)
- New fixes/features never deploy

## Resolution

A cleanup script has been added to remove the conflicting timestamp migrations from the `goose_db_version` table.

### Run the Cleanup Script

```bash
./scripts/cleanup_migrations.sh
```

This script will:
1. Connect to the database container.
2. Delete migration entries with `version_id > 10000000` (timestamp versions).
3. Display the remaining applied migrations.

### Manual Fix (Legacy)

If the script cannot be run, you can manually execute:

```bash
docker exec <db-container> psql -U $DB_USER -d $DB_NAME -c "DELETE FROM goose_db_version WHERE version_id > 10000000;"
```

After cleaning up, restart the application:

```bash
docker-compose restart app
```

## Root Cause

Database `goose_db_version` table contains mixed migration formats:

**Previous State**:
```sql
version_id   | is_applied
-------------+------------
0 - 25       | t           -- Numbered migrations ✓
20251228...  | t           -- Timestamp migration ❌ (incompatible)
20251228...  | t           -- Timestamp migration ❌ (incompatible)
```

**Correct State**:
```sql
version_id   | is_applied
-------------+------------
0 - 27       | t           -- All numbered migrations ✓
```

## Related Files

- [scripts/cleanup_migrations.sh](../../scripts/cleanup_migrations.sh) - Resolution script
- [scripts/fix_migrations_staging.sh](../../scripts/fix_migrations_staging.sh) - Legacy fix script
- [migrations/](../../migrations/) - Migration files
