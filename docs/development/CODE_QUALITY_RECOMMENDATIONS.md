# Code Quality Recommendations - BrandishBot_Go

Based on review of the Feature Development Guide against current codebase practices, here are recommendations to improve code quality and maintainability.

## ✅ Current Strengths

The codebase already follows many best practices:

1. **Good File Size Management**
   - Most files are within recommended limits
   - `inventory.go`: 400 lines (within 300-400 guideline)
   - `user.go` (postgres): 500 lines (within 400-500 guideline)
   - `service.go` (user): 657 lines (within 500-700 guideline)

2. **Proper Domain Organization**
   - Constants centralized in `internal/domain/constants.go`
   - Separate files for distinct entities (user, item, inventory, etc.)
   - Good use of typed constants

3. **Handler Patterns**
   - Consistent use of context-aware logging
   - Proper validation order (decode → validate → process)
   - Appropriate HTTP status codes

4. **Testing**
   - Good test file organization
   - Separate test files for distinct features (`search_test.go`, `lootbox_test.go`)

## 🔧 Recommended Improvements

### 1. Domain Layer Enhancements

**Issue**: Constants are split across `constants.go` and `user.go`, creating potential confusion.

**Current State:**
```go
// internal/domain/constants.go
const (
    ItemMoney = "money"
    ItemLootbox0 = "lootbox0"
    // ...
)

// internal/domain/user.go
const (
    ActionSearch = "search"
)

const (
    SearchCooldownDuration = 30 * time.Minute
)
```

**Recommendation**: Consolidate all constants into `constants.go` with clear grouping:

```go
// internal/domain/constants.go
package domain

import "time"

// Item name constants
const (
    ItemMoney    = "money"
    ItemLootbox0 = "lootbox0"
    ItemLootbox1 = "lootbox1"
    ItemLootbox2 = "lootbox2"
    ItemBlaster  = "blaster"
)

// Action name constants for cooldown tracking
const (
    ActionSearch = "search"
    ActionDaily  = "daily"  // Future feature
)

// Duration constants
const (
    SearchCooldownDuration = 30 * time.Minute
    // Add future cooldowns here
)
```

**Benefits:**
- Single source of truth for all domain constants
- Easier to find and update related constants
- Reduces import confusion
- Follows guide recommendation (line 86-113)

---

### 2. Handler File Organization

**Issue**: `inventory.go` at 400 lines is at the upper limit and handles multiple distinct operations.

**Current State:**
- `inventory.go` contains: Add, Remove, Give, Sell, Buy, Use, GetInventory (7 endpoints)

**Recommendation**: Split into logical groups:

```
internal/handler/
├── inventory.go          # GetInventory, AddItem, RemoveItem (~150 lines)
├── trading.go            # GiveItem (~100 lines)
├── economy.go            # SellItem, BuyItem (~200 lines)
├── item_usage.go         # UseItem, UpgradeItem, DisassembleItem (~250 lines)
├── search.go             # Search endpoint
└── ...
```

**Alternative** (if keeping together):
Add clear section comments in `inventory.go`:

```go
// ============================================================================
// Inventory Management Endpoints
// ============================================================================

func HandleAddItem...
func HandleRemoveItem...
func HandleGetInventory...

// ============================================================================
// Item Trading Endpoints
// ============================================================================

func HandleGiveItem...

// ============================================================================
// Economy Endpoints
// ============================================================================

func HandleSellItem...
func HandleBuyItem...
```

**Benefits:**
- Better code navigation
- Easier to locate specific handlers
- Prevents hitting 400-line limit
- Follows guide recommendation (line 316-319)

---

### 3. Test Coverage Documentation

**Issue**: No explicit test coverage metrics or goals documented in codebase.

**Recommendation**: Add test coverage tracking

1. **Create `Makefile` target**:
```makefile
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out | grep total | awk '{print "Total Coverage: " $$3}'

.PHONY: test-coverage-check
test-coverage-check:
	@echo "Checking coverage threshold (80%)..."
	@go test -coverprofile=coverage.out ./... >/dev/null 2>&1
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < 80" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below 80% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets 80% threshold"; \
	fi
```

2. **Add to CI/CD** (if applicable):
```yaml
- name: Test Coverage
  run: make test-coverage-check
```

**Benefits:**
- Enforces 80% coverage guideline (line 471)
- Provides visibility into coverage gaps
- Enables coverage tracking over time

---

### 4. Error Handling Consistency

**Issue**: Mix of error handling patterns across codebase.

**Current patterns observed:**
```go
// Pattern 1: Return nil for not found
if err == pgx.ErrNoRows {
    return nil, nil
}

// Pattern 2: Return custom error (newer code)
if user == nil {
    return nil, fmt.Errorf("%w: %s", domain.ErrUserNotFound, username)
}
```

**Recommendation**: Standardize on custom errors with wrapping

**Add to `internal/domain/errors.go`**:
```go
package domain

import "errors"

// Common errors
var (
    ErrUserNotFound         = errors.New("user not found")
    ErrItemNotFound         = errors.New("item not found")
    ErrInsufficientQuantity = errors.New("insufficient quantity")
    ErrInsufficientFunds    = errors.New("insufficient funds")
    ErrOnCooldown           = errors.New("action on cooldown")
    ErrFeatureLocked        = errors.New("feature is locked")
)
```

**Update repository pattern**:
```go
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
    // ...
    err := r.db.QueryRow(ctx, query, username).Scan(...)
    if err != nil {
        if err == pgx.ErrNoRows {
            return nil, domain.ErrUserNotFound
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return &user, nil
}
```

**Benefits:**
- Consistent error handling across layers
- Easier to test specific error cases
- Better error messages for debugging
- Follows guide recommendation (line 571-584)

---

### 5. Service Layer Documentation

**Issue**: Complex service methods lack step-by-step comments.

**Current state** (some methods):
```go
func (s *service) GiveItem(...) error {
    // Some logic...
}
```

**Recommendation**: Add numbered step comments for complex operations:

```go
// GiveItem transfers items from one user to another atomically
func (s *service) GiveItem(ctx context.Context, ownerUsername, receiverUsername, platform, itemName string, quantity int) error {
    log := logger.FromContext(ctx)
    log.Info("GiveItem called", "owner", ownerUsername, "receiver", receiverUsername)
    
    // 1. Validate users exist
    owner, err := s.validateUser(ctx, ownerUsername)
    if err != nil {
        return err
    }
    receiver, err := s.validateUser(ctx, receiverUsername)
    if err != nil {
        return err
    }
    
    // 2. Validate item exists
    item, err := s.validateItem(ctx, itemName)
    if err != nil {
        return err
    }
    
    // 3. Acquire locks in consistent order (prevent deadlocks)
    firstLock, secondLock := s.getUserLock(owner.ID), s.getUserLock(receiver.ID)
    if owner.ID > receiver.ID {
        firstLock, secondLock = secondLock, firstLock
    }
    
    firstLock.Lock()
    defer firstLock.Unlock()
    
    if owner.ID != receiver.ID {
        secondLock.Lock()
        defer secondLock.Unlock()
    }
    
    // 4. Execute transfer within transaction
    return s.executeGiveItemTx(ctx, owner, receiver, item, quantity)
}
```

**Benefits:**
- Easier to understand complex logic flow
- Matches guide pattern (line 240-277)
- Helps new contributors
- Makes code review easier

---

### 6. Repository Interface Location

**Issue**: Repository interfaces are defined in service packages (`internal/user/service.go`), which creates coupling.

**Current structure:**
```
internal/user/service.go - Contains Repository interface
internal/database/postgres/user.go - Implements interface
```

**Recommendation**: Move to dedicated package (Optional - breaking change)

```
internal/repository/
├── user.go        # type UserRepository interface
├── stats.go       # type StatsRepository interface
└── progression.go # type ProgressionRepository interface
```

**Alternative** (Non-breaking): Add documentation clarifying the pattern:

```go
// internal/user/service.go

// Repository defines the interface for user persistence.
// 
// Implementation location: internal/database/postgres/user.go
// 
// This interface is defined here (in the service package) to follow the
// dependency inversion principle - the service layer defines what it needs,
// and the database layer implements it.
type Repository interface {
    // ...
}
```

**Benefits:**
- Better separation of concerns
- Follows dependency inversion principle
- Makes testing boundaries clearer
- Guide mentions this at line 143

---

### 7. Handler Response Types

**Issue**: Some handlers don't define explicit response types for simple cases.

**Current pattern** (mixed):
```go
// With response type
type SearchResponse struct {
    Message string `json:"message"`
}
json.NewEncoder(w).Encode(SearchResponse{Message: message})

// Without response type
w.Write([]byte("Item added successfully"))
```

**Recommendation**: Always use typed responses for consistency:

```go
type SuccessResponse struct {
    Message string `json:"message"`
}

type ErrorResponse struct {
    Error string `json:"error"`
}

// Use consistently
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(SuccessResponse{Message: "Item added successfully"})
```

**Benefits:**
- Consistent API responses
- Easier to parse for clients
- Better API documentation
- Matches guide recommendation (line 412-419)

---

### 8. Add Linting Configuration

**Issue**: No documented linting configuration or standards.

**Recommendation**: Add `.golangci.yml`:

```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - unused
    - ineffassign
    - misspell
    - gosec
    - goconst

linters-settings:
  goconst:
    min-len: 3
    min-occurrences: 3
  
  gosec:
    excludes:
      - G104  # Audit errors not checked (we handle explicitly)

run:
  timeout: 5m
  tests: true

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

**Add Makefile target**:
```makefile
.PHONY: lint
lint:
	@echo "Running linters..."
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix:
	@echo "Running linters with auto-fix..."
	golangci-lint run --fix ./...
```

**Benefits:**
- Catches common issues early
- Enforces code quality standards
- Reduces code review burden
- Identifies potential bugs

---

### 9. Migration Testing

**Issue**: No automated migration testing documented.

**Recommendation**: Add migration testing script:

**File**: `scripts/test_migrations.sh`
```bash
#!/bin/bash
set -e

echo "Testing database migrations..."

# Use test database
export DATABASE_NAME="brandish_test_migrations"
export DATABASE_URL="postgres://localhost:5432/${DATABASE_NAME}?sslmode=disable"

# Reset test database
psql -c "DROP DATABASE IF EXISTS ${DATABASE_NAME};" || true
psql -c "CREATE DATABASE ${DATABASE_NAME};"

# Run migrations up
echo "Testing UP migrations..."
goose -dir migrations postgres "${DATABASE_URL}" up

# Run migrations down
echo "Testing DOWN migrations..."
goose -dir migrations postgres "${DATABASE_URL}" down

# Run migrations up again
echo "Testing UP migrations again..."
goose -dir migrations postgres "${DATABASE_URL}" up

echo "✅ All migrations passed!"

# Cleanup
psql -c "DROP DATABASE ${DATABASE_NAME};"
```

**Benefits:**
- Ensures migrations are reversible
- Catches migration errors early
- Follows guide recommendation (line 79)

---

## 📊 Implementation Priority

| Priority | Item | Effort | Impact |
|----------|------|--------|--------|
| **High** | 1. Consolidate domain constants | Low | High |
| **High** | 4. Standardize error handling | Medium | High |
| **High** | 8. Add linting configuration | Low | High |
| **Medium** | 3. Add test coverage tracking | Low | Medium |
| **Medium** | 5. Improve service documentation | Medium | Medium |
| **Medium** | 7. Standardize response types | Medium | Medium |
| **Low** | 2. Split handler files | Medium | Low |
| **Low** | 6. Repository interface location | High | Low |
| **Low** | 9. Migration testing script | Low | Medium |

## 🎯 Quick Wins (Start Here)

1. **Consolidate constants** (`internal/domain/constants.go`) - 30 minutes
2. **Add lint configuration** (`.golangci.yml`) - 30 minutes  
3. **Add test coverage Makefile targets** - 15 minutes
4. **Define standard error types** (`internal/domain/errors.go`) - 30 minutes

Total time for quick wins: ~1.5-2 hours
Total impact: Significant improvement in code quality and maintainability

## 📝 Next Steps

1. Review recommendations with team
2. Prioritize based on current sprint goals
3. Implement quick wins first
4. Create issues for larger refactorings
5. Update Feature Development Guide as patterns evolve

---

**Note**: All recommendations maintain backward compatibility except where marked as "breaking change". Always run full test suite after implementing changes.
