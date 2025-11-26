# Goose Migration Issue - Fixed

## Problem
Goose v3.26.0 panicked with:
```
panic: goose: duplicate version 1 detected:
    migrations/0001_initial_schema.up.sql
    migrations/0001_initial_schema.down.sql
```

## Root Cause
Goose v3.26+ changed how it handles migration files:
- **Old behavior**: Separate `.up.sql` and `.down.sql` files worked fine
- **New behavior**: Detects separate files as duplicates (expects combined files with `-- +goose Up/Down` markers)

## Solution Applied
Moved `.down.sql` files to `migrations/archive/`:
```bash
mkdir -p migrations/archive
mv migrations/*.down.sql migrations/archive/
```

## Why This Works
- Goose only sees `.up.sql` files now (no duplicates detected)
- Down migrations are preserved in archive for reference
- Can still manually rollback if needed by applying archive files

## Alternative Solutions

### Option 1: Downgrade Goose
```bash
go install github.com/pressly/goose/v3/cmd/goose@v3.11.0
```

### Option 2: Combine Files (Future Migrations)
For new migrations, use single files with markers:
```sql
-- +goose Up
CREATE TABLE users (...);

-- +goose Down  
DROP TABLE users;
```

## Current Status
✅ `.up.sql` files: Active in `migrations/`  
✅ `.down.sql` files: Archived in `migrations/archive/`  
✅ Migrations working with goose v3.26.0

## Commands
```bash
# Check status
make migrate-status

# Run migrations
make migrate-up

# Rollback (manual - use archive files if needed)
# OR downgrade goose version
```
