# Database Backups

This directory is intended for storing manual database backups.

## Usage

The project includes `Makefile` commands to help with database backup and restoration for development and testing purposes.

### Creating a Backup

To export the current production database (from the database container), run:

```bash
make db-export
```

This will create a `backup.sql` file in the project root. You can move this file into the `backups/` directory for safe keeping:

```bash
mv backup.sql backups/my-backup-$(date +%Y%m%d).sql
```

**Note:** All `.sql` files in this directory are git-ignored to prevent accidental commitment of sensitive data.

### Restoring to Test Database

You can import a `backup.sql` file from the project root into the **test database** (`brandishbot_test_db`) to debug with production data:

```bash
# Ensure backup.sql is in the project root first
cp backups/my-backup.sql backup.sql
make db-import
```

**Warning:** `make db-import` overwrites the test database. It does **not** restore to the production database.
