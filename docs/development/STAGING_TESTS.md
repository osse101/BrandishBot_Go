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

- `main_test.go`: Setup and configuration. Reads `STAGING_URL`.
- `health_test.go`: Basic health check (`/healthz`).
- `smoke_test.go`: Verifies core functionality (e.g., progression tree availability).

## Adding New Tests

1. Create a new file in `tests/staging/` (e.g., `user_flow_test.go`).
2. Add `//go:build staging` to the top of the file.
3. Use `makeRequest` helper to interact with the API.
4. Focus on black-box testing (test behavior, not implementation).
