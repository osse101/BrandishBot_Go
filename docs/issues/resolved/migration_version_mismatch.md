RESOLVED

# Migration Version Mismatch Issue

**Status**: Open  
**Priority**: Medium  
**Category**: Database/Infrastructure  
**Created**: 2025-12-31  
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

## Root Cause

Database `goose_db_version` table contains mixed migration formats:

**Current State**:
```sql
version_id   | is_applied
-------------+------------
0 - 25       | t           -- Numbered migrations ✓
20251228...  | t           -- Timestamp migration ❌ (incompatible)
20251228...  | t           -- Timestamp migration ❌ (incompatible)
-- Missing: 26, 27         -- Numbered migrations not registered
```

**Expected State**:
```sql
version_id   | is_applied
-------------+------------
0 - 27       | t           -- All numbered migrations ✓
```

## Impact

- **Severity**: HIGH - Blocks all deployments
- **Affected**: Any environment with timestamp migrations
- **Workaround**: Manual migration registration (see fix below)

## Fix for Existing Environments

### Step 1: Manually Apply Missing Migrations

```bash
# Apply migration 26
docker exec <db-container> psql -U $DB_USER -d $DB_NAME < migrations/0026_tune_progression_system.sql

# Apply migration 27
docker exec <db-container> psql -U $DB_USER -d $DB_NAME < migrations/0027_inventory_filters.sql
```

### Step 2: Register Migrations in Goose Tracking Table

```bash
docker exec <db-container> psql -U $DB_USER -d $DB_NAME -c \
  "INSERT INTO goose_db_version (version_id, is_applied) VALUES (26, true), (27, true) ON CONFLICT DO NOTHING;"
```

### Step 3: Restart Application

```bash
docker-compose restart app
```

### Step 4: Verify

```bash
# Check migrations are registered
docker exec <db-container> psql -U $DB_USER -d $DB_NAME -c \
  "SELECT version_id, is_applied FROM goose_db_version ORDER BY id;"

# Check app started successfully
docker logs <app-container> --tail 20
```

## Staging Fix Script

For convenience, created as `scripts/fix_migrations_staging.sh`:

```bash
#!/bin/bash
# Run this on staging server to fix the migration issue

set -e

echo "Fixing migration version mismatch..."

# Load environment variables
source .env

DB_CONTAINER="brandishbot_go-db-1"

# Apply migrations manually
echo "Applying migrations 26 and 27..."
docker exec $DB_CONTAINER psql -U $DB_USER -d $DB_NAME < migrations/0026_tune_progression_system.sql
docker exec $DB_CONTAINER psql -U $DB_USER -d $DB_NAME < migrations/0027_inventory_filters.sql

# Register in goose table
echo "Registering migrations in goose tracking table..."
docker exec $DB_CONTAINER psql -U $DB_USER -d $DB_NAME -c \
  "INSERT INTO goose_db_version (version_id, is_applied) VALUES (26, true), (27, true) ON CONFLICT DO NOTHING;"

# Restart app
echo "Restarting application..."
docker-compose restart app

# Wait for startup
sleep 5

# Verify
echo "Verifying application status..."
docker logs brandishbot_go-app-1 --tail 10

echo "✅ Migration fix complete!"
echo "Check logs above to verify app started successfully"
```

## Prevention

### Option 1: Clean Up Timestamp Migrations (Recommended)

Remove timestamp migrations from database and migrate them to numbered format:

```sql
-- CAUTION: Only run this if you're sure these migrations are obsolete
-- or have been re-applied as numbered migrations

-- Remove timestamp migrations
DELETE FROM goose_db_version WHERE version_id > 10000000;

-- Verify only numbered migrations remain
SELECT version_id, is_applied FROM goose_db_version ORDER BY id;
```

### Option 2: Standardize on Timestamp Format

Convert all numbered migrations to timestamp format (not recommended - requires migration file renaming).

### Option 3: Database Reset (Nuclear Option)

For development environments only:

```bash
# WARNING: This deletes all data!
make db-clean-test
make migrate-up-test
```

## Long-term Solution

1. **Standardize**: Choose ONE migration format (numbered recommended)
2. **Validate**: Add pre-deployment check for migration consistency
3. **Document**: Update migration creation process in docs
4. **Test**: Include migration validation in CI/CD

## Related Files

- [scripts/fix_migrations_staging.sh](../../scripts/fix_migrations_staging.sh) - Fix script
- [migrations/0026_tune_progression_system.sql](../../migrations/0026_tune_progression_system.sql)
- [migrations/0027_inventory_filters.sql](../../migrations/0027_inventory_filters.sql)
- [docker-entrypoint.sh](../../scripts/docker-entrypoint.sh) - Where migration check happens

## Notes

This issue prevented the search bug fix from deploying because the app container never successfully started. Always check container health after deployment if features aren't working as expected.

**Lesson**: Migration schema inconsistencies can silently prevent deployments while appearing to succeed (containers restart but fail to initialize).
