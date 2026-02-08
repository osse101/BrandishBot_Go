# Running Tests with Coverage

## Quick Commands

### Using Makefile (Recommended)
```bash
# Generate coverage report in logs/
make test-coverage

# View coverage in browser
open logs/coverage.html  # macOS
xdg-open logs/coverage.html  # Linux
```

### Manual Commands
```bash
# Create logs directory
mkdir -p logs

# Run tests with coverage
go test ./... -coverprofile=logs/coverage.out -covermode=atomic

# Generate HTML report
go tool cover -html=logs/coverage.out -o logs/coverage.html

# View specific package coverage
go test ./internal/logger -coverprofile=logs/logger_coverage.out
go tool cover -func=logs/logger_coverage.out
```

## Coverage Output Locations

**✅ Preferred**: `logs/coverage.out`, `logs/coverage.html`
- Organized with other generated files
- Already gitignored via `logs/` directory
- Easy to find and clean up

**❌ Avoid**: Root directory (`coverage.out`, `coverage.html`)
- Clutters repository root
- Requires explicit gitignore entries

## Current Test Coverage

Run `make test-coverage` to see coverage for:
- `internal/crafting`: ~76%
- `internal/database/postgres`: ~62%
- `internal/progression`: ~68%
- `internal/stats`: ~69%
- `internal/user`: ~72%
- `internal/logger`: ~85%+ (with new tests)

## CI/CD Integration

For automated testing, use:
```bash
go test ./... -coverprofile=logs/coverage.out -covermode=atomic -race
```

Upload `logs/coverage.out` to coverage services like Codecov or Coveralls.
