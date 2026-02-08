# Test Database Quick Start

## Setup (First Time)

```bash
# 1. Start test database
make db-test-up

# 2. Run migrations
make migrate-up-test

# 3. Load seed data
make db-seed-test
```

Test database is now ready at: `postgres://testuser:testpass@localhost:5433/testdb`

## Daily Usage

```bash
# Run integration tests
make test-integration

# Or manually test against test DB
DB_HOST=localhost DB_PORT=5433 DB_USER=testuser DB_PASSWORD=testpass DB_NAME=testdb make run
```

## Copy Production Data

```bash
# Export from production
make db-export

# Clean test DB and import
make db-clean-test
make migrate-up-test  
make db-import
```

## Commands

| Command | Purpose |
|---------|---------|
| `make db-test-up` | Start test DB (port 5433) |
| `make db-test-down` | Stop test DB |
| `make migrate-up-test` | Apply migrations |
| `make db-seed-test` | Load seed data |
| `make test-integration` | Run all integration tests |
| `make db-export` | Export production to backup.sql |
| `make db-import` | Import backup.sql to test |
| `make db-clean-test` | Drop all tables |

See [DATABASE_TESTING.md](./DATABASE_TESTING.md) for detailed documentation.
