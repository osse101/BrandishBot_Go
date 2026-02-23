# Devtool Utility

`cmd/devtool` is the central command-line utility for BrandishBot development, maintenance, and deployment. It aggregates various scripts and helpers into a single Go binary using a Command pattern and Registry, replacing scattered shell scripts.

## Overview

The `devtool` binary is designed to be the single entry point for:

- Development tasks (building, testing, coverage)
- Database management (migrations, seeding)
- Deployment workflows (build, push, deploy, rollback)
- Runtime operations (health checks, entrypoint logic)

Most `Makefile` targets delegate to this tool under the hood.

## Usage

```bash
go run ./cmd/devtool <command> [flags]
```

Or via `make`:

```bash
make migrate-up  # Runs: go run ./cmd/devtool migrate up
```

## Commands

### Development Workflow

- **`build`**: Compiles `cmd/app` and `cmd/discord` into `bin/app` and `bin/discord_bot`. Injects build metadata (Version, BuildTime, GitCommit) via ldflags.
- **`check-coverage`**: Runs tests with coverage, generates HTML reports, and verifies coverage thresholds.
  - **Flags**:
    - `-file <path>`: Check coverage for a specific file.
    - `-threshold <int>`: Set minimum coverage percentage (default 80).
    - `-html`: Generate `coverage.html`.
    - `-watch`: Watch for file changes and re-run tests automatically (uses `fsnotify`).
    - `-smart`: Only run tests for changed packages and their dependents (uses `package_selector.go`).
- **`check-deps`**: Verifies that required system dependencies (Go, Docker, etc.) are installed.
- **`bench`**: Runs benchmarks.

### Database & Migrations

- **`migrate`**: Manages database migrations.
  - `up`: Apply all pending migrations.
  - `down`: Rollback the last migration.
  - `status`: Show migration status.
  - `create`: Create a new migration file.
- **`check-db`**: Checks if the database is reachable.
- **`wait-for-db`**: Blocks until the database is ready (useful in CI/CD or startup scripts).
- **`test-migrations`**: Verifies migration idempotency (up/down cycles).

### Deployment

- **`deploy`**: Orchestrates the deployment process.
- **`rollback`**: Rolls back to a previous version.
- **`push`**: Pushes build artifacts to the registry.

### Runtime & Operations

- **`entrypoint`**: Replaces the legacy `docker-entrypoint.sh`. Handles:
  - Setting `DB_HOST` to "db" if missing (for Docker Compose compatibility).
  - Database readiness checks.
  - Conditional backups and migrations on startup.
  - Starting the application.
- **`health-check`**: Performs a health check against the running service.
- **`doctor`**: Diagnoses common environment issues.

## Architecture

The tool uses a **Command Registry** pattern. Commands are registered in `cmd/devtool/main.go` and implemented in separate files within `cmd/devtool/`. This allows for easy extensibility and shared logic (like logging or configuration loading) across all commands.

### Key Components

- **Package Selector (`package_selector.go`)**: Centralized logic for determining which packages to test based on git changes (`-smart` mode) or explicit selection.
- **File Watcher (`fsnotify`)**: Recursive file watching for the `-watch` flag, enabling rapid test feedback loops.
- **UI Helpers (`ui.go`)**: Cross-platform utilities, such as `OpenBrowser` (using `xdg-open`, `open`, or `start`) for opening coverage reports.
- **Strict Flags**: Commands use strict flag parsing to prevent ambiguous behavior.

### Ad-hoc Scripts

The `scripts/` directory has been removed. All ad-hoc utility scripts (e.g., debug tools, one-off fixes) are now consolidated into `cmd/devtool` commands to leverage the shared Go environment and type safety.
