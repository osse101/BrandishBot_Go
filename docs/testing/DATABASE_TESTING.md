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

### 1. Integration Tests with Shared Containers (Recommended)

Uses `testcontainers` with **TestMain pattern** to share a single container across all tests in a package.

**Pros**:

- Clean state for each test (with cleanup helpers)
- No manual setup required
- Works in CI/CD
- Isolated from production
- **85% faster** than per-test containers (postgres package: 37s → 5.5s)

**Cons**:

- Requires Docker running
- Slower than mocks
- Tests share database (use unique test data)

**Implementation Pattern:**

See comprehensive guide in [`TEST_GUIDANCE.md`](file:///home/osse1/projects/BrandishBot_Go/docs/testing/TEST_GUIDANCE.md#shared-container-infrastructure-recommended)

**Quick Example:**

```go
// TestMain in a file with tests
func TestMain(m *testing.M) {
    flag.Parse()
    if !testing.Short() {
        testDBConnString, terminate = setupContainer(context.Background())
        testPool, _ = database.NewPool(testDBConnString, 20, 30*time.Minute, time.Hour)
    }
    code := m.Run()
    if testPool != nil { testPool.Close() }
    if terminate != nil { terminate() }
    os.Exit(code)
}

func TestSomething(t *testing.T) {
    if testDBConnString == "" {
        t.Skip("database not available")
    }
    ensureMigrations(t) // Thread-safe, runs once
    // Use testPool...
}
```

**Real Examples:**

- [`internal/database/postgres/`](file:///home/osse1/projects/BrandishBot_Go/internal/database/postgres/integration_test.go) - 9 tests, shared container
- [`internal/progression/`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/service_integration_test.go) - 4 tests, shared container

**Performance:**

- postgres package: 37s → 5.5s (85% faster)
- progression package: ~14s → 7.2s (~50% faster)

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

| Command                 | Description                               |
| ----------------------- | ----------------------------------------- |
| `make test-integration` | Run integration tests with testcontainers |
| `make db-test-up`       | Start test database                       |
| `make db-test-down`     | Stop test database                        |
| `make migrate-up-test`  | Run migrations on test database           |
| `make db-seed-test`     | Load test seed data                       |
| `make db-export`        | Export production database to backup.sql  |
| `make db-import`        | Import backup.sql into test database      |
| `make db-clean-test`    | Drop and recreate test database           |

---

## Seed Data Files

Test seed data is located in `internal/database/seeds/`:

- `test_user.sql` - Creates test users
- `test_recipe.sql` - Adds test recipes

To add more seed data, create SQL files in `internal/database/seeds/` and update `cmd/devtool/seed.go`.

---

## Best Practices

1. **For Unit Tests**: Use mocks (like the existing `MockRepository` in crafting tests)
2. **For Integration Tests**:
   - **Multiple tests in package**: Use shared TestMain pattern (see [`TEST_GUIDANCE.md`](file:///home/osse1/projects/BrandishBot_Go/docs/testing/TEST_GUIDANCE.md))
   - **Single test file**: Use testcontainers per-test (auto-managed)
3. **For Manual Testing**: Use local test database with seed data
4. **Never test against production database directly**
5. **Use unique test data** when sharing containers:
   ```go
   userID := fmt.Sprintf("test-user-%d", time.Now().UnixNano())
   ```
6. **Use cleanup helpers** to reset state between tests sharing a database

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
