# Test Database Setup & Management

This document explains how to set up and use test databases for BrandishBot_Go.

## Quick Start

```bash
# Run all integration tests (uses testcontainers)
make test-integration

# Load test database with seed data
make db-seed-test

# Export production data for testing
make db-export

# Import data into test database
make db-import
```

## Test Database Strategies

### 1. Integration Tests (Recommended for CI/CD)

Uses `testcontainers` to spin up temporary Postgres containers. This is what the existing tests use.

**Pros**:
- Clean state for each test
- No manual setup required
- Works in CI/CD
- Isolated from production

**Cons**:
- Requires Docker running
- Slower than mocks

**Usage**: Run `go test ./internal/database/postgres -v`

---

### 2. Local Test Database (For Manual Testing)

A persistent test database using Docker Compose.

**Setup**:
```bash
# Start test database
docker compose -f docker compose.test.yml up -d

# Run migrations
make migrate-up-test

# Seed with test data
make db-seed-test
```

**Pros**:
- Persistent data for manual testing
- Fast startup after initial setup
- Can inspect data between tests

**Cons**:
- Requires manual setup
- Need to clean up manually

---

## Copying Production Data to Test

### Method 1: Using pg_dump (SQL export)

```bash
# 1. Export production database
docker exec brandishbot_go-db-1 pg_dump -U $DB_USER -d $DB_NAME > backup.sql

# 2. Copy to test database
docker exec -i brandishbot_go-test-db-1 psql -U testuser -d testdb < backup.sql
```

### Method 2: Using Docker Volume Copy

```bash
# 1. Stop databases
docker compose down
docker compose -f docker compose.test.yml down

# 2. Copy volume data
docker run --rm \
  -v brandishbot_go_pgdata:/from \
  -v brandishbot_go_test_pgdata:/to \
  alpine sh -c "cd /from && cp -av . /to"

# 3. Restart
docker compose -f docker compose.test.yml up -d
```

### Method 3: Using Makefile Commands (Easiest)

```bash
# Export production data
make db-export  # Creates backup.sql

# Import into test database
make db-import  # Loads backup.sql into test DB
```

---

## Makefile Commands Reference

| Command | Description |
|---------|-------------|
| `make test-integration` | Run integration tests with testcontainers |
| `make db-test-up` | Start test database |
| `make db-test-down` | Stop test database |
| `make migrate-up-test` | Run migrations on test database |
| `make db-seed-test` | Load test seed data |
| `make db-export` | Export production database to backup.sql |
| `make db-import` | Import backup.sql into test database |
| `make db-clean-test` | Drop and recreate test database |

---

## Seed Data Files

Test seed data is located in `scripts/`:
- `setup_test_user.sql` - Creates test users
- `seed_test_recipe.sql` - Adds test recipes

To add more seed data, create SQL files in `scripts/` and update `db-seed-test` target in Makefile.

---

## Best Practices

1. **For Unit Tests**: Use mocks (like the existing `MockRepository` in crafting tests)
2. **For Integration Tests**: Use testcontainers (auto-managed)
3. **For Manual Testing**: Use local test database with seed data
4. **Never test against production database directly**
5. **Use transactions in tests and rollback after each test when possible**

---

## Troubleshooting

**Integration tests failing with Docker error?**
- Ensure Docker is running
- Run `docker ps` to verify
- Integration tests will skip if Docker is unavailable

**Test database won't start?**
- Check if port 5433 is available
- Run `docker compose -f docker compose.test.yml logs`

**Data not persisting in test database?**
- Check volume is created: `docker volume ls | grep test`
- Verify migrations ran: `make migrate-status-test`
