# Database Migrations Guide

## Overview

BrandishBot uses [goose](https://github.com/pressly/goose) for database migrations. Migrations are managed as single `.sql` files containing both Up and Down logic, separated by goose markers.

## Migration Files

Migrations are located in the `migrations/` directory:

```
migrations/
├── 0001_initial_schema_v1.sql
├── 0002_add_modifier_config.sql
└── ...
```

### File Format

Each migration file uses the following format:

```sql
-- +goose Up
-- SQL for applying the change
CREATE TABLE example (id SERIAL PRIMARY KEY);

-- +goose Down
-- SQL for rolling back the change
DROP TABLE example;
```

## Running Migrations

The `Makefile` provides the primary interface for migration tasks:

| Command                      | Description                          |
| :--------------------------- | :----------------------------------- |
| `make migrate-status`        | Show the current migration status    |
| `make migrate-up`            | Apply all pending migrations         |
| `make migrate-down`          | Rollback the last applied migration  |
| `make migrate-create NAME=x` | Create a new migration file template |

### Local Development Setup

When setting up your environment for the first time:

1.  Ensure Docker is running: `make check-db`
2.  Run all migrations: `make migrate-up`

## Creating New Migrations

To add a new table or modify the schema:

1.  **Generate a template**:
    ```bash
    make migrate-create NAME=add_my_new_feature
    ```
2.  **Edit the file**: Open the newly created file in `migrations/` and fill in the `-- +goose Up` and `-- +goose Down` sections.
3.  **Test locally**:
    ```bash
    make migrate-up
    make migrate-down
    make migrate-up
    ```

## Best Practices

- **Immutability**: Never edit a migration file that has already been committed and applied to shared environments. Always create a new migration for changes.
- **Safety**: Use `IF NOT EXISTS` for table/index creation and `IF EXISTS` for dropping when possible, although goose handles versioning automatically.
- **Transactions**: Goose runs each migration in its own transaction by default.
- **Naming**: Use descriptive names (e.g., `add_user_preferences`) rather than generic ones.

## Troubleshooting

### Version Mismatch

If you encounter a version mismatch or "out of order" error, check the `goose_db_version` table in your database:

```bash
go run cmd/debug/main.go
# Or manually:
# SELECT * FROM goose_db_version;
```

### Resetting the Database

If you need to completely wipe and restart the database:

```bash
go run cmd/reset/main.go
make migrate-up
```
