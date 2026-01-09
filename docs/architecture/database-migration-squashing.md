# Database Migration Squashing Guide

## Overview

Migration squashing consolidates multiple development migrations into a single baseline migration. This improves performance for fresh database setups and reduces maintenance overhead.

## When to Squash Migrations

Consider squashing when:
- You have accumulated many development migrations (20+)
- Setting up fresh databases is slow
- You're preparing for a major version release (v1.0, v2.0)
- The migration history has become difficult to maintain

**Do NOT squash if:**
- You have active production databases with the old migrations
- The project is less than 6 months old
- You have fewer than 15 migrations

## Prerequisites

Before starting, ensure:
- ✅ All current migrations are applied to production
- ✅ All databases are backed up
- ✅ Team is notified of upcoming changes
- ✅ You have tested the squash process in a non-production environment

## Squashing Process

### Phase 1: Preparation

1. **Create a git tag** for the pre-squash state:
   ```bash
   git tag pre-migration-squash
   git push origin pre-migration-squash
   ```

2. **Archive original migrations**:
   ```bash
   mkdir -p migrations/archive/pre-v1
   cp migrations/*.sql migrations/archive/pre-v1/
   ```

3. **Apply all migrations** to a test database:
   ```bash
   docker-compose -f docker-compose.test.yml up -d
   goose -dir migrations postgres "CONNECTION_STRING" up
   ```

### Phase 2: Schema Extraction

4. **Extract the complete schema**:
   ```bash
   pg_dump \
     --schema-only \
     --no-owner \
     --no-acl \
     -U user -d database > schema_dump.sql
   ```

5. **Clean the extracted schema**:
   - Remove psql meta-commands (lines starting with `\`)
   - Remove `SET` statements
   - Remove `SELECT pg_catalog.*` calls
   - **CRITICAL**: Remove any `CREATE TABLE` or constraints for `goose_db_version`
   
   Goose manages its own version table. Including it in migrations will cause conflicts.

6. **Extract seed data** from the database:
   ```sql
   -- Query production for actual values
   SELECT * FROM items WHERE internal_name LIKE 'lootbox%';
   SELECT * FROM platforms;
   ```

### Phase 3: Create Squashed Migration

7. **Create the squashed migration file**:
   ```bash
   goose -dir migrations create initial_schema_v1 sql
   ```

8. **Assemble the migration**:
   ```sql
   -- +goose Up
   -- ProjectName v1.0 - Initial Schema
   -- Squashed from N development migrations (date range)
   
   [DDL from cleaned schema dump - excluding goose_db_version]
   
   -- Seed data
   INSERT INTO platforms (name) VALUES ('twitch'), ('youtube') 
   ON CONFLICT DO NOTHING;
   
   INSERT INTO items (internal_name, public_name, base_value) VALUES
       ('item1', 'Item 1', 100),
       ('item2', 'Item 2', 200)
   ON CONFLICT DO NOTHING;
   
   -- +goose Down
   DROP TABLE IF EXISTS [tables in reverse dependency order];
   ```

9. **Important**: Verify all INSERT statements include:
   - All non-nullable columns
   - `ON CONFLICT DO NOTHING` for idempotency
   - Actual production values (not placeholder data)

### Phase 4: Testing

10. **Test on fresh database**:
    ```bash
    # Remove old migrations (keep archive)
    rm migrations/[old_migration_files].sql
    
    # Reset test database
    docker-compose -f docker-compose.test.yml down
    docker volume rm [test_volume]
    docker-compose -f docker-compose.test.yml up -d
    
    # Apply squashed migration
    goose -dir migrations postgres "CONNECTION_STRING" up
    ```

11. **Verify the database**:
    ```bash
    # Check migration status
    goose -dir migrations postgres "CONNECTION_STRING" status
    
    # Count tables
    psql -c "SELECT COUNT(*) FROM information_schema.tables 
             WHERE table_schema='public' AND table_type='BASE TABLE';"
    
    # Verify seed data
    psql -c "SELECT COUNT(*) FROM items;"
    psql -c "SELECT COUNT(*) FROM platforms;"
    ```

12. **Run integration tests**:
    ```bash
    make test-integration
    ```
    
    If tests fail, investigate and fix before proceeding.

### Phase 5: Cleanup

13. **Update documentation**:
    - Update `migrations/README.md`
    - Document the squash in changelog
    - Update contributor guidelines

14. **Commit the changes**:
    ```bash
    git add migrations/
    git commit -m "Squash migrations to v1.0 baseline"
    git tag v1.0-migration-baseline
    git push origin main --tags
    ```

## Common Issues & Solutions

### Issue: "relation 'goose_db_version' already exists"

**Cause**: Migration tries to create goose_db_version table, but goose creates this automatically.

**Solution**: Remove the CREATE TABLE statement and any constraints for `goose_db_version` from your squashed migration.

### Issue: NULL scan errors

**Cause**: INSERT statements missing columns that Go structs expect to scan.

**Example**:
```sql
-- ❌ Wrong
INSERT INTO items (internal_name, public_name) VALUES ('box', 'Box');

-- ✅ Correct  
INSERT INTO items (internal_name, public_name, default_display) 
VALUES ('box', 'Box', 'Treasure Box');
```

**Solution**: Include ALL columns that your application reads, even if they have default values.

### Issue: Seed data doesn't match production

**Cause**: Using placeholder values instead of actual production data.

**Solution**: Query production database and copy exact values:
```sql
-- Extract actual values
SELECT internal_name, public_name, default_display, base_value 
FROM items WHERE internal_name IN ('item1', 'item2');
```

## Best Practices

### DO ✅

- Always backup before squashing
- Test on fresh database multiple times
- Include all required columns in seed data
- Use `ON CONFLICT DO NOTHING` for idempotency
- Archive original migrations with documentation
- Tag git commits for easy rollback
- Test in staging before production
- Verify column names against actual schema

### DON'T ❌

- Don't include `goose_db_version` in migrations
- Don't assume column names without verification
- Don't use invented seed data values
- Don't delete original migrations without archiving
- Don't ignore test failures
- Don't squash with pending migrations
- Don't skip backup steps

## Rollback Procedure

If something goes wrong:

1. **Restore from git**:
   ```bash
   git checkout pre-migration-squash
   ```

2. **Restore database** from backup:
   ```bash
   psql -U user -d database < backup.sql
   ```

3. **Investigate** the issue before trying again

## Post-Squash Workflow

After successful squashing:

- New migrations continue from the next number (0002, 0003, etc.)
- The squashed migration becomes the new baseline
- Fresh databases start with the squashed migration
- Existing production databases are unaffected (already have all migrations applied)

## Architecture Decision

Migrations are numbered sequentially. After squashing:
- `0001_initial_schema_v1.sql` - Squashed baseline
- `0002_add_new_feature.sql` - New migration after squash
- `0003_another_feature.sql` - Next migration

This maintains compatibility with goose's versioning system.

## References

- [goose Documentation](https://github.com/pressly/goose)
- [PostgreSQL pg_dump Documentation](https://www.postgresql.org/docs/current/app-pgdump.html)
- Project-specific migration reports in `docs/archived/`
