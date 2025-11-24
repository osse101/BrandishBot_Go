# Database Migrations Guide

## What Are Migrations?

Database migrations are incremental SQL files that evolve your database schema over time. Instead of having one large SQL file, you have multiple smaller files that represent individual changes.

## Migration Files

Migrations are located in the `migrations/` directory and follow this naming pattern:

```
0001_initial_schema.up.sql       # Creates users, platforms, links
0001_initial_schema.down.sql     # Removes users, platforms, links
0002_create_items.up.sql         # Creates items table
0002_create_items.down.sql       # Removes items table
...
```

- **`.up.sql`**: Applies the change (forward migration)
- **`.down.sql`**: Reverses the change (rollback migration)

## How Migrations Work

1. **Sequential Order**: Migrations are numbered (0001, 0002, 0003, etc.)
2. **Applied Once**: Each migration is applied in order when you run setup
3. **Idempotent**: Migrations use `IF NOT EXISTS` and `ON CONFLICT` to be safe to run multiple times

## Running Migrations

### Apply All Migrations

```bash
go run cmd/setup/main.go
```

This will:
1. Create the database if it doesn't exist
2. Read all `.up.sql` files from `migrations/` directory
3. Apply them in alphabetical order (0001, 0002, 0003, etc.)

### Verify Database State

```bash
go run cmd/debug/main.go
```

This shows the contents of all tables.

## Creating New Migrations

When you need to add a new table or modify the schema:

1. **Create a new migration file** with the next number:
   ```
   migrations/0006_add_feature.up.sql
   migrations/0006_add_feature.down.sql
   ```

2. **Write the SQL**:
   - `.up.sql` - Add your new table/column/index
   - `.down.sql` - Remove what you added

3. **Run setup** to apply the new migration:
   ```bash
   go run cmd/setup/main.go
   ```

## Example: Adding a New Table

**migrations/0006_add_stats.up.sql**:
```sql
CREATE TABLE IF NOT EXISTS user_stats (
    user_id UUID PRIMARY KEY REFERENCES users(user_id),
    total_items INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**migrations/0006_add_stats.down.sql**:
```sql
DROP TABLE IF EXISTS user_stats;
```

## Current Migrations

1. **0001_initial_schema**: Creates users, platforms, and user_platform_links tables
2. **0002_create_items**: Creates items table
3. **0003_item_types**: Creates item_types and item_type_assignments tables
4. **0004_user_inventory**: Creates user_inventory table with JSONB
5. **0005_seed_items**: Seeds lootbox items and types

## Benefits

- ✅ **Version Control**: Track database changes in Git
- ✅ **Team Collaboration**: Everyone applies the same changes
- ✅ **Incremental**: Add features one step at a time
- ✅ **Rollback**: Can reverse changes with .down.sql files
- ✅ **Production Safety**: Only new migrations are applied

## Important Notes

- **Never edit** existing migration files after they've been applied
- **Always create** new migration files for changes
- **Test migrations** before applying to production
- **Keep migrations small** - one logical change per migration
