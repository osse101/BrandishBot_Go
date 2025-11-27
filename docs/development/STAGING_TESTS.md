# Staging Integration Tests

This directory contains integration tests designed to run against a deployed staging environment (or a locally running instance that mimics staging).

## Prerequisites

- A running instance of BrandishBot (staging or local)
- Network access to the instance

## Running Tests

### Using Makefile

```bash
# Run against default (http://localhost:8080)
make test-staging

# Run against specific URL with API Key
STAGING_URL=http://staging.brandishbot.com:8080 API_KEY=my-secret-key make test-staging
```

### Manual Execution

```bash
export STAGING_URL=http://staging.brandishbot.com:8080
export API_KEY=my-secret-key
go test -tags=staging -v ./tests/staging
```

## Test Structure

- **`main_test.go`**: Setup and configuration. Reads `STAGING_URL` and `API_KEY`.
- **`health_test.go`**: Basic health check (`/healthz`).
- **`smoke_test.go`**: Verifies core functionality (progression tree availability).
- **`progression_test.go`**: Comprehensive progression endpoint tests (tree, voting, engagement).
- **`user_test.go`**: User and economy tests (registration, inventory, prices, recipes).
- **`stats_test.go`**: Stats endpoint tests (system stats, leaderboard, event recording).

### Test Coverage

The staging test suite includes **11 tests** covering:

1. **Health & Smoke Tests** (2 tests)
   - `/healthz` - Health check
   - `/progression/tree` - Basic progression tree

2. **Progression Tests** (4 tests)
   - `/progression/tree` - Full tree endpoint
   - `/progression/available` - Available unlocks
   - `/progression/status` - Voting status
   - `/progression/vote` - Voting flow
   - `/progression/engagement` - Engagement tracking

3. **User & Economy Tests** (4 tests)
   - `/user/register` - User registration
   - `/user/inventory` - Inventory retrieval
   - `/prices` - Price information
   - `/recipes` - Crafting recipes

4. **Stats Tests** (4 tests)
   - `/stats/system` - System statistics
   - `/stats/leaderboard` - Leaderboard
   - `/stats/user` - User-specific stats
   - `/stats/event` - Event recording

## Adding New Tests

1. Create a new file in `tests/staging/` (e.g., `user_flow_test.go`).
2. Add `//go:build staging` to the top of the file.
3. Use `makeRequest` helper to interact with the API.
4. Focus on black-box testing (test behavior, not implementation).
