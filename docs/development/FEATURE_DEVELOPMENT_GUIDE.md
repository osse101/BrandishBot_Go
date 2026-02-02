# Adding New Features to BrandishBot_Go

This guide outlines the step-by-step process for adding new features to BrandishBot_Go, organized by system component. It incorporates best practices observed in the codebase and lessons learned from recent feature implementations.

## Table of Contents

1. [Planning Phase](#planning-phase)
2. [Database Layer](#database-layer)
3. [Domain Layer](#domain-layer)
4. [Repository Layer](#repository-layer)
5. [Service Layer](#service-layer)
6. [Progression Modifiers](#progression-modifiers)
7. [Handler Layer](#handler-layer)
8. [Server/Routing Layer](#serverrouting-layer)
9. [Testing](#testing)
10. [Best Practices](#best-practices)

---

## Planning Phase

### 1. Define Requirements

- [ ] Document the feature's purpose and behavior
- [ ] Identify all inputs and outputs
- [ ] Determine if feature requires progression system unlock
- [ ] **Identify numeric values that could use progression modifiers** ⭐
- [ ] Identify cooldowns or rate limits needed
- [ ] Define success and error states

> 💡 **Deep Dive**: For detailed architecture and rules of the progression system, see [PROGRESSION_GUIDANCE.md](PROGRESSION_GUIDANCE.md).

### 2. Check Integration Points

- [ ] Does it need database persistence?
- [ ] Does it interact with inventory?
- [ ] Does it involve multiple users (transactions needed)?
- [ ] Does it need engagement tracking?
- [ ] Are there external dependencies?

### 3. Create Implementation Plan

- [ ] List all files to be created/modified
- [ ] Identify dependencies and order of implementation
- [ ] Plan verification strategy (unit tests, integration tests)

---

## Database Layer

### Migration Files (`migrations/`)

**When to create a migration:**

- New tables needed
- Adding/modifying columns
- Creating indexes
- Seeding required data

**File naming:**

```
XXXX_descriptive_name.sql
```

Example: `0016_create_user_cooldowns.sql`

**Migration structure:**

```sql
-- +goose Up
-- Create statements with IF NOT EXISTS for safety
CREATE TABLE IF NOT EXISTS table_name (
    column_id SERIAL PRIMARY KEY,
    -- columns here
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_name ON table_name(column);

-- +goose Down
-- Cleanup in reverse order
DROP TABLE IF EXISTS table_name;
```

**Best practices:**

- ✅ Always use `IF NOT EXISTS` and `IF EXISTS`
- ✅ Create indexes for foreign keys and frequently queried columns
- ✅ Use `TIMESTAMP WITH TIME ZONE` for timestamps
- ✅ Include comments explaining complex constraints
- ✅ Test both Up and Down migrations
- ❌ Never modify existing migrations that have been deployed

---

## Domain Layer

### Constants (`internal/domain/`)

**File organization:**

- Keep related constants together
- Use separate files when a domain grows beyond ~150 lines

**Constant naming patterns:**

```go
// Action names for cooldowns
const (
    ActionSearch = "search"
    ActionDaily  = "daily"
)

// Durations
const (
    SearchCooldownDuration = 30 * time.Minute
    DailyCooldownDuration  = 24 * time.Hour
)

// Item names (centralized)
const (
    ItemMoney    = "money"
    ItemLootbox0 = "lootbox0"
    ItemLootbox1 = "lootbox1"
)
```

**Best practices:**

- ✅ Use descriptive names that indicate type/purpose
- ✅ Group related constants together
- ✅ Export constants that are used across packages
- ✅ Use typed constants where appropriate
- ❌ Avoid magic strings/numbers scattered in code

### Domain Models

**When to create a new domain file:**

- New entity type (User, Item, Inventory, etc.)
- File exceeds 200 lines
- Logically distinct concepts

**Model structure:**

```go
// MyEntity represents...
type MyEntity struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}
```

---

## Repository Layer

### Repository Interface (`internal/user/service.go` or similar)

**Adding methods to Repository interface:**

```go
type Repository interface {
    // Existing methods...

    // New feature methods - group logically
    GetMyData(ctx context.Context, userID string) (*MyData, error)
    UpdateMyData(ctx context.Context, userID string, data MyData) error
}
```

**Best practices:**

- ✅ Accept `context.Context` as first parameter
- ✅ Use pointer receivers for output structs
- ✅ Return `error` as last return value
- ✅ Group related methods together with comments

### Repository Implementation (`internal/database/postgres/`)

**File organization:**

```
internal/database/postgres/
├── user.go           # User-related queries (< 500 lines)
├── stats.go          # Stats-related queries
├── progression.go    # Progression queries
└── [feature].go      # New feature if complex enough
```

**When to create a new file:**

- Feature has 5+ database methods
- File would exceed 500 lines
- Logically distinct from existing files

**Query implementation pattern:**

```go
// GetMyData retrieves my data from the database
func (r *MyRepository) GetMyData(ctx context.Context, userID string) (*domain.MyData, error) {
    query := `
        SELECT id, name, value
        FROM my_table
        WHERE user_id = $1
    `

    var data domain.MyData
    err := r.db.QueryRow(ctx, query, userID).Scan(
        &data.ID,
        &data.Name,
        &data.Value,
    )
    if err != nil {
        if err == pgx.ErrNoRows {
            return nil, nil // or domain.ErrNotFound
        }
        return nil, fmt.Errorf("failed to get my data: %w", err)
    }
    return &data, nil
}
```

**Best practices:**

- ✅ Use parameterized queries ($1, $2) to prevent SQL injection
- ✅ Handle `pgx.ErrNoRows` explicitly
- ✅ Wrap errors with context using `fmt.Errorf`
- ✅ Use transactions for multi-table operations
- ✅ Include descriptive comments
- ❌ Never use string concatenation for SQL

---

## Service Layer

### Service Interface (`internal/[feature]/service.go`)

**Adding service methods:**

```go
type Service interface {
    // Existing methods...

    // HandleMyFeature performs the feature action
    HandleMyFeature(ctx context.Context, username string, params MyParams) (string, error)
}
```

### Service Implementation

**File size guidelines:**

- Keep service.go under 700 lines
- Extract helpers when file grows large
- Consider splitting by feature area if needed

**Implementation pattern:**

```go
// HandleMyFeature implements the feature logic
func (s *service) HandleMyFeature(ctx context.Context, username string, params MyParams) (string, error) {
    log := logger.FromContext(ctx)
    log.Info("HandleMyFeature called", "username", username)

    // 1. Get/validate user
    user, err := s.validateUser(ctx, username)
    if err != nil {
        return "", err
    }

    // 2. Acquire lock for thread-safety
    lock := s.getUserLock(user.ID)
    lock.Lock()
    defer lock.Unlock()

    // 3. Check business rules (cooldowns, prerequisites, etc.)
    if err := s.checkMyRules(ctx, user); err != nil {
        return "", err
    }

    // 4. Perform core logic
    result, err := s.performMyLogic(ctx, user, params)
    if err != nil {
        log.Error("Failed to perform logic", "error", err)
        return "", err
    }

    // 5. Update state
    if err := s.updateMyState(ctx, user, result); err != nil {
        log.Error("Failed to update state", "error", err)
        return "", err
    }

    log.Info("Feature completed", "username", username, "result", result)
    return result, nil
}
```

**Best practices:**

- ✅ Use user-level locking for concurrent operations on same user
- ✅ Log at INFO level for key operations
- ✅ Log at ERROR level with full context
- ✅ Break complex logic into helper methods
- ✅ Use transactions for atomic multi-resource updates
- ✅ Always defer unlock/rollback calls
- ❌ Don't hold locks during external API calls

**Helper method pattern:**

```go
// Private helpers use lowercase
func (s *service) validateMyParams(params MyParams) error {
    if params.Value < 0 {
        return fmt.Errorf("value must be positive")
    }
    return nil
}
```

---

## Progression Modifiers

**When to use progression modifiers:**
Features with numeric values that should scale with player progression should use the progression modifier system instead of hardcoded values.

### Identifying Modifier Candidates

Ask these questions:

- ✅ Does this feature have a numeric value that could increase with progression?
- ✅ Would players benefit from unlocking upgrades to this value?
- ✅ Is this a core game mechanic (XP, rewards, cooldowns, rates)?

**Good candidates:**

- XP multipliers
- Reward bonuses
- Cooldown reductions
- Success rates
- Resource caps

**Not good candidates:**

- UI display values
- Internal IDs
- Boolean flags

### Adding ProgressionService to Your Feature

**1. Add to service interface:**

```go
// ProgressionService defines required progression methods
type ProgressionService interface {
    GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
}
```

**2. Add to service struct:**

```go
type service struct {
    repo           Repository
    progressionSvc ProgressionService  // Add this
    // ... other dependencies
}
```

**3. Update constructor:**

```go
func NewService(repo Repository, progressionSvc ProgressionService) Service {
    return &service{
        repo:           repo,
        progressionSvc: progressionSvc,
    }
}
```

### Using GetModifiedValue()

**Pattern with fallback (recommended):**

```go
func (s *service) calculateReward(ctx context.Context, baseReward int) int {
    // Apply modifier if available
    modified, err := s.progressionSvc.GetModifiedValue(ctx, "my_feature_reward_bonus", float64(baseReward))
    if err != nil {
        // Fallback to base value on error
        log.Warn("Failed to get modifier, using base value", "error", err)
        return baseReward
    }
    return int(modified)
}
```

**For cooldowns:**

```go
baseDuration := 5 * time.Minute

// Apply reduction modifier
if s.progressionSvc != nil {
    modifiedDuration, err := s.progressionSvc.GetModifiedValue(ctx, "my_cooldown_reduction", float64(baseDuration))
    if err == nil {
        baseDuration = time.Duration(modifiedDuration)
    }
}
```

### Real-World Examples

**Example 1: Job System XP Multiplier**

```go
// internal/job/service.go
func (s *service) getXPMultiplier(ctx context.Context) float64 {
    modified, err := s.progressionSvc.GetModifiedValue(ctx, "job_xp_multiplier", 1.0)
    if err != nil {
        return 1.0  // Default multiplier
    }
    return modified
}

// Usage: xp = baseXP * getXPMultiplier(ctx)
```

**Example 2: Gamble Win Bonus**

```go
// internal/gamble/service.go - in ExecuteGamble()
totalValue := int64(drop.Value * drop.Quantity)

if s.progressionSvc != nil {
    modifiedValue, err := s.progressionSvc.GetModifiedValue(ctx, "gamble_win_bonus", float64(totalValue))
    if err == nil {
        totalValue = int64(modifiedValue)
    }
}
```

**Example 3: Cooldown Reduction**

```go
// internal/cooldown/postgres.go
cooldownDuration := b.config.GetCooldownDuration(action)

if b.progressionSvc != nil && action == "search" {
    modifiedDuration, err := b.progressionSvc.GetModifiedValue(ctx, "search_cooldown_reduction", float64(cooldownDuration))
    if err == nil {
        cooldownDuration = time.Duration(modifiedDuration)
    }
}
```

### Adding Modifier Nodes to Progression Tree

**1. Add to `configs/progression_tree.json`:**

```json
{
  "node_key": "upgrade_my_feature_bonus",
  "name": "My Feature Bonus",
  "type": "upgrade",
  "description": "Increase my feature bonus by 10% per level",
  "tier": 3,
  "max_level": 5,
  "prerequisites": ["feature_my_feature"],
  "modifier_config": {
    "feature_key": "my_feature_bonus",
    "modifier_type": "multiplicative",
    "base_value": 1.0,
    "per_level_value": 0.1
  }
}
```

**2. Modifier types:**

| Type             | Formula                         | Use Case                       |
| ---------------- | ------------------------------- | ------------------------------ |
| `multiplicative` | `base * (1 + level * perLevel)` | XP boost, reward bonus         |
| `linear`         | `base + (level * perLevel)`     | Daily caps, absolute increases |
| `fixed`          | `perLevel` (ignores base)       | Fixed values at each level     |
| `percentage`     | `base * (perLevel / 100)`       | Percentage-based changes       |

**Example calculations:**

```
Multiplicative (base=1.0, perLevel=0.10):
  Level 0: 1.0
  Level 3: 1.3  (30% boost)
  Level 5: 1.5  (50% boost)

Linear (base=3, perLevel=1):
  Level 0: 3
  Level 3: 6
  Level 5: 8
```

### Testing with Modifiers

**Mock ProgressionService in tests:**

```go
type MockProgressionService struct{}

func (m *MockProgressionService) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
    return baseValue, nil  // Return unmodified for testing
}
```

**Or test with specific values:**

```go
mockProg.On("GetModifiedValue", mock.Anything, "my_feature_bonus", 100.0).Return(150.0, nil)
```

### Checklist for Modifier Integration

- [ ] Add `ProgressionService` to your service interface
- [ ] Inject `progressionSvc` in constructor and `main.go`
- [ ] Identify numeric values to modify
- [ ] Replace hardcoded values with `GetModifiedValue()` calls
- [ ] Add fallback handling for errors
- [ ] Create modifier node in `progression_tree.json`
- [ ] Update test mocks to include `GetModifiedValue`
- [ ] Document feature key in code comments
- [ ] Test with different progression levels

---

## Handler Layer

### Handler Files (`internal/handler/`)

**File organization:**

```
internal/handler/
├── inventory.go      # Inventory endpoints
├── user.go           # User management
├── stats.go          # Stats endpoints
├── search.go         # Search feature
└── [myfeature].go    # New feature
```

**When to create a new file:**

- Feature has multiple related endpoints
- Handler file would exceed 400 lines
- Logically distinct from existing handlers

**Handler implementation pattern:**

```go
package handler

import (
    "encoding/json"
    "net/http"

    "github.com/osse101/BrandishBot_Go/internal/logger"
    "github.com/osse101/BrandishBot_Go/internal/middleware"
    "github.com/osse101/BrandishBot_Go/internal/progression"
    "github.com/osse101/BrandishBot_Go/internal/[feature]"
)

type MyFeatureRequest struct {
    Username string `json:"username"`
    Param1   string `json:"param1"`
    Param2   int    `json:"param2"`
}

type MyFeatureResponse struct {
    Message string `json:"message"`
    Data    string `json:"data"`
}

func HandleMyFeature(svc feature.Service, progressionSvc progression.Service) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log := logger.FromContext(r.Context())

        // 1. Check feature unlock (if applicable)
        if progressionSvc != nil {
            unlocked, err := progressionSvc.IsFeatureUnlocked(r.Context(), progression.FeatureMyFeature)
            if err != nil {
                log.Error("Failed to check feature unlock", "error", err)
                http.Error(w, "Failed to check feature availability", http.StatusInternalServerError)
                return
            }
            if !unlocked {
                log.Warn("Feature is locked")
                http.Error(w, "Feature is not yet unlocked", http.StatusForbidden)
                return
            }
        }

        // 2. Decode request
        var req MyFeatureRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            log.Error("Failed to decode request", "error", err)
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        log.Debug("Feature request", "username", req.Username, "param1", req.Param1)

        // 3. Validate inputs
        if err := ValidateUsername(req.Username); err != nil {
            log.Warn("Invalid username", "error", err)
            http.Error(w, "Invalid username", http.StatusBadRequest)
            return
        }

        // 4. Call service
        result, err := svc.HandleMyFeature(r.Context(), req.Username, req.Param1, req.Param2)
        if err != nil {
            log.Error("Feature failed", "error", err, "username", req.Username)
            http.Error(w, "Failed to perform feature", http.StatusInternalServerError)
            return
        }

        log.Info("Feature completed", "username", req.Username, "result", result)

        // 5. Track engagement (if applicable)
        middleware.TrackEngagementFromContext(
            middleware.WithUserID(r.Context(), req.Username),
            progressionSvc,
            "feature_used",
            1,
        )

        // 6. Return response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(MyFeatureResponse{
            Message: result,
            Data:    "additional data",
        })
    }
}
```

**Best practices:**

- ✅ Always get logger from context
- ✅ Validate all inputs before processing
- ✅ Use appropriate HTTP status codes
- ✅ Return JSON for data responses
- ✅ Track engagement for progression features
- ✅ Log at DEBUG for request details, INFO for completion, ERROR for failures
- ❌ Don't expose internal errors to clients

---

## Server/Routing Layer

### Adding Routes (`internal/server/server.go`)

**Route organization:**

```go
// Group routes by feature area
// User routes
mux.HandleFunc("/user/register", handler.HandleRegisterUser(userService))
mux.HandleFunc("/user/inventory", handler.HandleGetInventory(userService))
mux.HandleFunc("/user/search", handler.HandleSearch(userService, progressionService))

// Economy routes
mux.HandleFunc("/user/item/sell", handler.HandleSellItem(economyService, progressionService))
mux.HandleFunc("/user/item/buy", handler.HandleBuyItem(economyService, progressionService))

// Stats routes
mux.HandleFunc("/stats/user", handler.HandleGetUserStats(statsService))
```

**Best practices:**

- ✅ Group related routes together with comments
- ✅ Use RESTful naming conventions
- ✅ Pass required services to handlers
- ✅ Keep routes alphabetically sorted within groups
- ❌ Don't duplicate route paths

---

## Testing

### Unit Tests

**Test file organization:**

```
internal/[package]/
├── service.go
├── service_test.go        # Main service tests
├── [feature]_test.go      # Feature-specific tests
└── concurrency_test.go    # Concurrency tests
```

**When to create separate test file:**

- Feature-specific tests exceed 200 lines
- Testing requires unique mocks/fixtures
- Logically distinct test group (e.g., concurrency, integration)

**Test coverage requirements:**

- ✅ **Minimum 80%** coverage for new features
- ✅ Test happy path
- ✅ Test error cases
- ✅ Test edge cases (empty inputs, boundaries)
- ✅ Test concurrency if applicable

**Test naming pattern:**

```go
func TestFeatureName_Scenario(t *testing.T) {
    // Arrange
    repo := NewMockRepository()
    setupTestData(repo)
    svc := NewService(repo, lockManager)

    // Act
    result, err := svc.MyMethod(ctx, params)

    // Assert
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

**Mock repository pattern:**

````go
type MockRepository struct {
    data map[string]*MyData
}

func (m *MockRepository) GetMyData(ctx context.Context, id string) (*MyData, error) {
    if data, ok := m.data[id]; ok {
        return data, nil
    }
    return nil, nil
}
}\n```

### Mock Architecture: Why Two Mock Systems Exist

The project has **two types of mocks** that serve **different, complementary purposes**:

#### Generated Mocks (`mocks/` package)

**Purpose**: Cross-package testing (mocking dependencies)
**Location**: `mocks/mock_*.go`
**Generated via**: `make mocks` (using `mockery`)
**Used by**: Packages that consume the interface

**Example:**
```go
// internal/handler/user_test.go
import \"github.com/osse101/BrandishBot_Go/mocks\"

func TestHandleGetInventory(t *testing.T) {
    mockUserService := new(mocks.MockUserService)  // Mocking dependency
    mockUserService.On(\"GetInventory\", ...).Return(...)

    handler := HandleGetInventory(mockUserService)
    // Test handler logic
}
````

**When to use**:

- Testing code that _uses_ an interface defined elsewhere
- Example: Handler tests mocking services
- Example: Service tests mocking external dependencies

**Important**: Run `make mocks` after changing any interface to regenerate.

#### In-Package Mocks

**Purpose**: Same-package testing (testing the package's own implementation)  
**Location**: `internal/[package]/mock_*.go`  
**Maintained**: Manually (or within test files)  
**Used by**: Tests in the same package

**Example:**

```go
// internal/user/service_test.go
func TestAddItem(t *testing.T) {
    repo := NewMockRepository()  // In-package mock
    setupTestData(repo)
    svc := NewService(repo, nil, nil)

    // Test service logic
}
```

**Why they must exist**:  
These mocks **cannot be in the `mocks/` package** due to Go's import cycle restrictions:

```
internal/user -> mocks -> internal/user (IMPORT CYCLE!)
```

**When to use**:

- Testing code in the _same package_ as the interface
- Example: `user.Service` tests mocking `user.Repository`
- Example: `eventlog.Service` tests mocking `eventlog.Repository`

#### Two Styles of In-Package Mocks

**1. Stateful "Fake" Implementations**  
Store actual state in memory (maps, slices):

```go
// internal/user/mock_repository.go
type MockRepository struct {
    users map[string]*domain.User
    inventories map[string]*domain.Inventory
}

func (m *MockRepository) GetUser(ctx context.Context, id string) (*domain.User, error) {
    return m.users[id], nil  // Returns stored state
}
```

**Use when**:

- Integration-style unit tests need realistic behavior
- Tests manipulate state directly: `repo.users["alice"] = ...`
- Easier to read than lots of expectations

**2. Expectation-Based Mocks**  
Use `testify/mock` for verification:

```go
// internal/eventlog/mock_repository.go
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) LogEvent(ctx context.Context, ...) error {
    args := m.Called(ctx, ...)
    return args.Error(0)
}
```

**Use when**:

- Verifying specific method calls
- Testing error paths
- Need to assert call counts

#### Summary: When to Use Which Mock

| Scenario                     | Mock Type            | Example                        |
| ---------------------------- | -------------------- | ------------------------------ |
| Testing handlers             | Generated (`mocks/`) | `mocks.MockUserService`        |
| Testing services (cross-pkg) | Generated (`mocks/`) | `mocks.MockProgressionService` |
| Testing service (same pkg)   | In-package           | `user.MockRepository`          |
| Integration-style unit tests | In-package stateful  | `user.MockRepository`          |
| Verification/error paths     | In-package mock      | `eventlog.MockRepository`      |

**Best practices:**

- ✅ Use generated mocks for cross-package dependencies
- ✅ Keep in-package mocks for same-package testing
- ✅ Add comments explaining why in-package mocks exist (avoid confusion)
- ✅ Run `make mocks` after interface changes
- ❌ Don't try to move in-package mocks to `mocks/` (creates import cycles)
- ❌ Don't delete in-package mocks thinking they're "duplicate"

- ✅ Use table-driven tests for multiple scenarios
- ✅ Create reusable test fixtures
- ✅ Test with `-race` flag for concurrency
- ✅ Use subtests (`t.Run()`) for related cases
- ✅ Mock external dependencies
- ❌ Don't test implementation details
- ❌ Avoid brittle tests that break on refactoring

### Integration Tests

**Staging tests** (`tests/staging/`):

- Test full HTTP request/response cycle
- Verify database persistence
- Test progression system integration

---

## Best Practices

### Code Organization

**File size guidelines:**

- **Domain files**: 150-200 lines per file
- **Repository files**: 400-500 lines per file
- **Service files**: 500-700 lines per file
- **Handler files**: 300-400 lines per file
- **Test files**: 200-300 lines per feature

**When to split:**

- File exceeds size guidelines
- Multiple distinct responsibilities
- Code becomes hard to navigate

### Constants and Configuration

**Where to define constants:**

| Type             | Location                       | Example                  |
| ---------------- | ------------------------------ | ------------------------ |
| Domain constants | `internal/domain/`             | Item names, action names |
| Progression keys | `internal/progression/keys.go` | Feature keys             |
| HTTP status      | Use `http.Status*`             | `http.StatusOK`          |
| Durations        | Domain or service              | `30 * time.Minute`       |

**Avoid:**

- ❌ Magic numbers scattered in code
- ❌ Hardcoded strings
- ❌ Configuration in multiple places

### Error Handling

**Error wrapping:**

```go
if err != nil {
    return fmt.Errorf("failed to perform action: %w", err)
}
```

**Custom errors:**

```go
var (
    ErrUserNotFound = errors.New("user not found")
    ErrInsufficientQuantity = errors.New("insufficient quantity")
)
```

**Error checking:**

```go
if errors.Is(err, domain.ErrUserNotFound) {
    // Handle specific error
}
```

### Logging

**Log levels:**

- **DEBUG**: Request details, internal state
- **INFO**: Key operations, completions
- **WARN**: Validation failures, recoverable errors
- **ERROR**: Failures requiring attention

**Logging pattern:**

```go
log := logger.FromContext(ctx)
log.Info("Operation started", "username", username, "action", action)
log.Error("Operation failed", "error", err, "username", username)
```

### Concurrency

**When to use locks:**

- Multiple operations on same user
- Race conditions possible
- Modifying shared state

**Lock ordering (prevent deadlocks):**

```go
// Consistent ordering by ID
firstLock := s.getUserLock(id1)
secondLock := s.getUserLock(id2)

if id1 > id2 {
    firstLock, secondLock = secondLock, firstLock
}

firstLock.Lock()
defer firstLock.Unlock()

if id1 != id2 {
    secondLock.Lock()
    defer secondLock.Unlock()
}
```

**Testing concurrency:**

```bash
go test -race ./...
```

### Database Best Practices

**Use transactions when:**

- Updating multiple tables
- Operations must be atomic
- Transferring resources between users

**Transaction pattern:**

```go
tx, err := s.repo.BeginTx(ctx)
if err != nil {
    return fmt.Errorf("failed to begin transaction: %w", err)
}
defer tx.Rollback(ctx)

// Perform operations...

return tx.Commit(ctx)
```

---

## Feature Checklist

Use this checklist when implementing a new feature:

### Planning

- [ ] Requirements documented
- [ ] Integration points identified
- [ ] Implementation plan created

### Implementation

- [ ] Migration created and tested
- [ ] Domain constants added
- [ ] Repository interface updated
- [ ] Repository implementation added
- [ ] Service method implemented
- [ ] Handler created
- [ ] Route added to server
- [ ] Progression key added (if applicable)

### Testing

- [ ] Unit tests written (80%+ coverage)
- [ ] Edge cases tested
- [ ] Concurrency tested (if applicable)
- [ ] Integration tests added
- [ ] Tested with `-race` flag

### Documentation

- [ ] Code comments added
- [ ] API documented
- [ ] Feature added to project docs

### Verification

- [ ] Code builds without errors
- [ ] All tests pass
- [ ] No lint errors
- [ ] Feature works end-to-end

---

## Example: Search Feature Implementation

See the Search feature implementation as a reference example:

- Migration: `migrations/0016_create_user_cooldowns.sql`
- Domain: `internal/domain/user.go` (constants)
- Repository: `internal/database/postgres/user.go` (cooldown methods)
- Service: `internal/user/service.go` (HandleSearch)
- Handler: `internal/handler/search.go`
- Tests: `internal/user/search_test.go`
- Route: `internal/server/server.go` (/user/search)

This implementation follows all patterns and best practices outlined in this guide.
