# Pre-v1 Migration Archive

This directory contains the original 29 development migrations that were squashed into `0001_initial_schema_v1.sql` on January 5, 2026.

## Migration History

These migrations represent the development history from November 2025 through January 2026:

- **0001-0015**: Initial schema, items, inventory, crafting, progression system
- **0016-0019**: Cooldowns, events, gambling, jobs
- **0020-0024**: Voting sessions, linking, item naming, constraint fixes
- **0025-0027**: Gamble concurrency, progression tuning, inventory filters
- **0028-0029**: Prerequisites junction table (v2.0), parent_node_id cleanup

## Why They Were Squashed

- Faster test database setup (1 migration vs 29)
- Cleaner baseline for v1.0 production release
- Easier onboarding for new developers
- No production databases existed at squash time

## Recovery

If you need to recreate the migration history:
1. These files are preserved exactly as they were
2. The goose version table was at version 29 before squashing
3. Database schema dump taken: January 5, 2026 02:56 UTC

## Note

Do NOT apply these migrations to databases created after the squash! Use `0001_initial_schema_v1.sql` instead.
