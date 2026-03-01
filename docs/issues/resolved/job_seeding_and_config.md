# Job Seeding and Bonus Configuration (Resolved)

## Overview

Previously, job definitions were hardcoded in Go constants, making them difficult to modify or extend without code changes. Additionally, job bonuses were handled separately from the main progression system.

## Resolution

- **Migration 0027 (`seed_jobs.sql`)**: Job definitions and display names are now seeded into the database. This allows for dynamic job management and potential future admin UI integration.
- **Migration 0028 (`add_bonus_and_job_unlock_configs.sql`)**: Introduced the `bonus_config` table. This unifies progression bonuses and job bonuses into a single configuration structure, simplifying the bonus calculation logic.

## Impact

- **Flexibility**: New jobs can be added via SQL migrations or potentially an admin UI without redeploying the application code.
- **Unified Logic**: The `bonus_config` table allows for a consistent way to apply modifiers (e.g., XP multipliers, drop rates) regardless of whether they come from a progression node or a job level.
- **Maintenance**: Reduced hardcoded values in the codebase.

## Status

**Resolved**: Migrations are applied and the system is functioning using the new database-driven configuration.
