# Staging Integration Tests

This directory contains integration tests for running against a deployed staging environment or local instance.

## Quick Start

```bash
# Default (localhost:8080 with test-api-key)
make test-staging

# Against staging server
STAGING_URL=https://staging.example.com:8080 API_KEY=your-key make test-staging
```

## Environment Variables

- **`STAGING_URL`**: Target server URL (default: `http://localhost:8080`)
- **`API_KEY`**: API key for authentication (default: `test-api-key`)

## Test Files

| File | Description | Test Count |
|------|-------------|------------|
| `main_test.go` | Test setup and configuration | - |
| `health_test.go` | Health check endpoint | 1 |
| `smoke_test.go` | Basic progression tree | 1 |
| `progression_test.go` | Progression system tests | 4 |
| `user_test.go` | User and economy tests | 4 |
| `stats_test.go` | Statistics endpoints | 4 |

**Total: 11 tests**

## Build Tag

All files use `//go:build staging` to exclude them from normal test runs.

```bash
# These tests are NOT included:
go test ./...

# These tests ARE included:
go test -tags=staging ./tests/staging
```

## Documentation

See [`docs/development/STAGING_TESTS.md`](../../docs/development/STAGING_TESTS.md) for detailed documentation.
