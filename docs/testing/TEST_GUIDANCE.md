# Test Guidance Document

**Purpose:** Define ideal testing practices for BrandishBot_Go  
**Audience:** All contributors writing or reviewing tests

---

## Core Principle

> **A test's value is measured by the bugs it catches, not the lines it covers.**

---

## The 5-Case Testing Model

Every testable unit should prove correctness across these dimensions:

### 1. Best Case

The happy path. Valid inputs, expected behavior.

```go
func TestSellItem_Success(t *testing.T) {
    // User has item, sells it, receives money
}
```

### 2. Boundary Case ⭐

**Critical: Test every boundary three ways**

1. **On the boundary** - Exact limit value
2. **Just inside** - One unit within valid range
3. **Just outside** - One unit beyond valid range

```go
func TestSellItem_QuantityBoundaries(t *testing.T) {
    tests := []struct {
        name     string
        quantity int
        valid    bool
    }{
        {"negative (beyond lower)", -1, false},
        {"zero (on lower boundary)", 0, false},
        {"one (just inside)", 1, true},
        {"max (on upper boundary)", 10000, true},
        {"over max (beyond upper)", 10001, false},
    }
    // Test each boundary case
}
```

**Common Boundaries:**

- Numeric: min/max values, zero, negative
- Collections: empty, single item, max capacity
- Strings: empty, max length, unicode edge cases
- Time: past/future, timezone boundaries
- Permissions: just below/at/above threshold

**Handling "Unbounded" Values:**

Even "unbounded" values have practical boundaries:

1. **Type limits**: `int32`, `int64` have max values - test near these
2. **Business limits**: Most cases impose logical constraints (e.g., max inventory = 10000)
3. **Representative large values**: Test with sufficiently large numbers that exercise the logic

```go
// Example: Testing "any positive integer" quantity
const MaxReasonableQuantity = 1000000 // Practical upper bound

tests := []struct {
    name  string
    qty   int
    valid bool
}{
    {"zero", 0, false},              // Lower boundary
    {"one", 1, true},                // Just inside
    {"typical", 100, true},          // Normal case
    {"large", MaxReasonableQuantity, true}, // Practical upper
    {"overflow risk", math.MaxInt32, true}, // Type boundary
}
```

### 3. Edge Case

Unusual but legal scenarios within valid range.

```go
func TestSellItem_LastItem(t *testing.T) {
    // Selling last item removes slot from inventory
    // Verify slot cleanup logic
}
```

### 4. Invalid Case

Malformed or incorrect inputs.

```go
func TestSellItem_InvalidInputs(t *testing.T) {
    // Empty username, non-existent item
    // All should return appropriate errors
}
```

### 5. Hostile Case

Deliberately malicious attempts.

```go
func TestSellItem_SQLInjection(t *testing.T) {
    // Item name: "'; DROP TABLE items--"
    // Username with control characters
    // Verify proper sanitization
}
```

---

## Test Structure

### Ideal Test Function

```go
func Test<Function>_<Scenario>(t *testing.T) {
    // 1. ARRANGE: Setup test data
    input := createValidInput()
    expected := calculateExpectedOutput()

    // 2. ACT: Execute function under test
    actual, err := FunctionUnderTest(input)

    // 3. ASSERT: Verify results
    require.NoError(t, err)
    assert.Equal(t, expected, actual)
}
```

**Max 30 lines per test.** Extract complex setup to helpers.

### Table-Driven Tests

Use for testing multiple scenarios of same function:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
        errMsg  string
    }{
        {"valid input", "test", false, ""},
        {"empty string", "", true, "cannot be empty"},
        {"too long", string(make([]byte, 101)), true, "too long"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

---

## Naming Conventions

### Test Files

- `<package>_test.go` - Unit tests
- `<package>_integration_test.go` - Integration tests (build tag required)

### Test Functions

Pattern: `Test<Function>_<Scenario>`

**Good:**

- `TestCalculateDiscount_ValidPercentage`
- `TestSellItem_InsufficientQuantity`
- `TestLoadConfig_MissingAPIKey`

**Bad:**

- `TestDiscount` - Too vague
- `TestCase1` - Meaningless
- `TestSellItemWithInvalidUserAndZeroQuantity` - Too specific, split into separate tests

### Subtest Names

Use `t.Run()` with descriptive strings:

```go
t.Run("returns error when user not found", func(t *testing.T) {
    // Test body
})
```

---

## Assertions

### Use the Right Tool

```go
// ✅ Preferred - testify/assert
assert.Equal(t, expected, actual)
assert.NoError(t, err)
assert.Contains(t, slice, item)

// ✅ Use require for fatal conditions
require.NoError(t, err) // Stops test if fails
assert.Equal(t, value, result) // Continues if fails

// ❌ Avoid - raw if statements
if result != expected {
    t.Errorf("got %v, want %v", result, expected)
}
```

### Common Patterns

```go
// Errors
assert.NoError(t, err)
assert.Error(t, err)
assert.ErrorIs(t, err, ErrNotFound)
assert.ErrorContains(t, err, "not found")

// Equality
assert.Equal(t, expected, actual)
assert.NotEqual(t, unexpected, actual)

// Collections
assert.Len(t, slice, 3)
assert.Contains(t, slice, item)
assert.Empty(t, slice)

// Numeric
assert.Greater(t, actual, threshold)
assert.InDelta(t, 1.0, result, 0.001) // Floats

// Booleans
assert.True(t, condition)
assert.False(t, condition)

// Nil checks
assert.Nil(t, ptr)
assert.NotNil(t, ptr)
```

---

## Mocking Strategy

### When to Mock

**Mock external dependencies:**

- Database
- HTTP clients
- File system
- Time/randomness
- External services

**Don't mock:**

- Simple value objects
- Pure functions
- Internal utilities

### Mockery (Recommended)

**For handler/controller tests**, use auto-generated mocks with mockery. Note that service mocks are in the global `mocks` package, while repository mocks are local to their package (e.g., `internal/user/mocks`).

```go
import "github.com/osse101/BrandishBot_Go/mocks"

func TestHandler_GetUser(t *testing.T) {
    // ✅ Use mockery for clean, type-safe tests
    // Service mocks are in the global package
    mockSvc := mocks.NewMockUserService(t)
    mockSvc.On("GetUser", "123").Return(user, nil)

    handler := NewHandler(mockSvc)
    result, err := handler.GetUser("123")

    assert.NoError(t, err)
    assert.Equal(t, user, result)
    mockSvc.AssertExpectations(t)
}
```

**Regenerate after interface changes:**

```bash
make mocks
```

**See [MOCKING.md](./MOCKING.md) for complete mockery guide.**

### Functional Mocks

**For service/integration tests**, use functional in-memory mocks:

```go
// ✅ Good for tests needing state management
type MockRepository struct {
    users map[string]*domain.User
}

func (m *MockRepository) GetUser(id string) (*domain.User, error) {
    if user, ok := m.users[id]; ok {
        return user, nil
    }
    return nil, ErrNotFound
}
```

### Mock Complexity Levels

**Level 1: No Mocks** (Ideal)

```go
func TestAdd(t *testing.T) {
    assert.Equal(t, 5, Add(2, 3))
}
```

**Level 2: Mockery Mocks** (Handler Tests)

```go
func TestHandler_GetUser(t *testing.T) {
    mockRepo := mocks.NewMockUserRepository(t)
    mockRepo.On("FindUser", "123").Return(user, nil)

    handler := NewHandler(mockRepo)
    result, err := handler.GetUser("123")

    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

**Level 3: Functional Mocks** (Service Tests)

```go
// Use when complex state management needed
mockRepo := NewMockRepository() // In-memory implementation
mockRepo.users["123"] = &domain.User{ID: "123"}
```

**Level 4: Multiple Mocks** (Minimize)

```go
// If test requires 3+ mocks, consider:
// - Is this an integration test?
// - Can we test smaller units?
// - Is design too coupled?
```

---

## Test Data Management

### Test Fixtures

Create helper functions for common test data:

```go
// helpers_test.go
func createTestUser() *domain.User {
    return &domain.User{
        ID:       "test-user-123",
        Username: "testuser",
    }
}

func createInventoryWithMoney(amount int) *domain.Inventory {
    return &domain.Inventory{
        Slots: []domain.InventorySlot{
            {ItemID: 1, Quantity: amount}, // Money
        },
    }
}
```

### Avoid Magic Numbers

```go
// ❌ Bad
assert.Equal(t, 42, result)

// ✅ Good
const expectedDiscount = 42
assert.Equal(t, expectedDiscount, result)

// ✅ Better - show calculation
basePrice := 100
discountPercent := 0.42
expected := int(basePrice * discountPercent)
assert.Equal(t, expected, result)
```

### Define Test Boundaries as Constants

```go
const (
    MinQuantity = 1
    MaxQuantity = 10000
    MinUsernameLength = 3
    MaxUsernameLength = 32
)

func TestQuantityValidation(t *testing.T) {
    tests := []struct {
        name  string
        qty   int
        valid bool
    }{
        {"below min", MinQuantity - 1, false},
        {"at min", MinQuantity, true},
        {"at max", MaxQuantity, true},
        {"above max", MaxQuantity + 1, false},
    }
    // Clear, self-documenting boundaries
}
```

---

## Error Testing

### Verify Error Behavior

```go
func TestOperation_ErrorHandling(t *testing.T) {
    tests := []struct {
        name      string
        input     Input
        wantErr   error
        errSubstr string
    }{
        {
            name:      "user not found",
            input:     Input{UserID: "nonexistent"},
            wantErr:   ErrUserNotFound,
            errSubstr: "user not found",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := Operation(tt.input)

            require.Error(t, err)
            assert.ErrorIs(t, err, tt.wantErr)
            assert.Contains(t, err.Error(), tt.errSubstr)
        })
    }
}
```

---

## HTTP Handler Testing

### Use httptest Package

```go
func TestHandleGetUser(t *testing.T) {
    // Setup
    mockService := &MockUserService{}
    mockService.On("GetUser", "123").Return(user, nil)
    handler := NewHandler(mockService)

    // Create request
    req := httptest.NewRequest("GET", "/users/123", nil)
    rec := httptest.NewRecorder()

    // Execute
    handler.HandleGetUser(rec, req)

    // Verify
    assert.Equal(t, http.StatusOK, rec.Code)

    var response UserResponse
    err := json.Unmarshal(rec.Body.Bytes(), &response)
    require.NoError(t, err)
    assert.Equal(t, user.ID, response.ID)
}
```

---

## Property-Based Testing

For mathematical or algorithmic functions, verify properties:

```go
func TestDiminishingReturns_Properties(t *testing.T) {
    t.Run("always between 0 and 1", func(t *testing.T) {
        for value := 0.0; value <= 10000; value += 10 {
            result := DiminishingReturns(value, 100)
            assert.GreaterOrEqual(t, result, 0.0)
            assert.LessOrEqual(t, result, 1.0)
        }
    })

    t.Run("monotonically increasing", func(t *testing.T) {
        prev := 0.0
        for value := 0.0; value <= 1000; value += 10 {
            current := DiminishingReturns(value, 100)
            assert.GreaterOrEqual(t, current, prev)
            prev = current
        }
    })
}
```

---

## Common Anti-Patterns

### 1. Testing Implementation, Not Behavior

```go
// ❌ Bad - tests internal implementation
func TestSellItem(t *testing.T) {
    // Verify private method called
    // Check internal state changes
}

// ✅ Good - tests public API behavior
func TestSellItem_Success(t *testing.T) {
    moneyBefore := getBalance(user)
    SellItem(user, item, 1)
    moneyAfter := getBalance(user)

    assert.Equal(t, moneyBefore + itemValue, moneyAfter)
}
```

### 2. Overly Specific Assertions

```go
// ❌ Bad - brittle, breaks on message changes
assert.Equal(t, "User john_doe not found in system", err.Error())

// ✅ Good - tests essential behavior
assert.ErrorIs(t, err, ErrUserNotFound)
assert.Contains(t, err.Error(), "not found")
```

### 3. Test Interdependence

```go
// ❌ Bad - tests depend on execution order
var sharedUser *User
func TestA() { sharedUser = CreateUser() }
func TestB() { UpdateUser(sharedUser) }

// ✅ Good - independent tests
func TestCreateUser(t *testing.T) {
    user := CreateUser()
    assert.NotNil(t, user)
}

func TestUpdateUser(t *testing.T) {
    user := createTestUser() // Helper
    UpdateUser(user)
    assert.Equal(t, expectedState, user.State)
}
```

### 4. No Assertions

```go
// ❌ Bad - test passes even if code is broken
func TestProcess(t *testing.T) {
    Process(input)
    // No assertions!
}

// ✅ Good
func TestProcess(t *testing.T) {
    result := Process(input)
    assert.NotNil(t, result)
    assert.NoError(t, result.Error)
}
```

---

## Performance Considerations

### Test Speed

- Unit tests: <10ms
- Integration tests: <1s
- Full suite: <30s

```go
// Use t.Parallel() for independent tests
func TestCalculation(t *testing.T) {
    t.Parallel() // Run concurrently with other parallel tests

    result := Calculate(input)
    assert.Equal(t, expected, result)
}
```

### Avoid Heavy Setup

```go
// ❌ Bad - creates real DB connection per test
func TestUserService(t *testing.T) {
    db := createRealDatabase()
    defer db.Close()
    // ...
}

// ✅ Good - use mocks for unit tests
func TestUserService(t *testing.T) {
    mockRepo := &MockRepository{}
    service := NewService(mockRepo)
    // ...
}
```

---

## Integration Tests

### Separate from Unit Tests

```go
//go:build integration
// +build integration

package user_test

// This only runs with: go test -tags=integration
func TestUserService_Integration(t *testing.T) {
    // Use real database, testcontainers, etc.
}
```

### Shared Container Infrastructure (Recommended)

**For packages with multiple integration tests**, use the TestMain pattern to share a single container:

```go
// test_helpers.go - Shared infrastructure
package mypackage

var (
    testDBConnString  string
    testPool          *pgxpool.Pool
    migrationsApplied bool
    migrationsMux     sync.Mutex
)

// ensureMigrations applies migrations once for all tests
func ensureMigrations(t *testing.T) {
    migrationsMux.Lock()
    defer migrationsMux.Unlock()

    if migrationsApplied {
        return
    }

    ctx := context.Background()
    if err := applyMigrations(ctx, t, testPool, "../../migrations"); err != nil {
        t.Fatalf("failed to apply migrations: %v", err)
    }

    migrationsApplied = true
}
```

```go
// integration_test.go - TestMain setup
func TestMain(m *testing.M) {
    flag.Parse()
    var terminate func()

    if !testing.Short() {
        ctx := context.Background()
        var connStr string
        connStr, terminate = setupContainer(ctx)
        testDBConnString = connStr

        if connStr != "" {
            testPool, _ = database.NewPool(connStr, 20, 30*time.Minute, time.Hour)
        }
    }

    code := m.Run()

    if testPool != nil {
        testPool.Close()
    }
    if terminate != nil {
        terminate()
    }

    os.Exit(code)
}

func setupContainer(ctx context.Context) (string, func()) {
    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("Recovered from panic: %v\n", r)
        }
    }()

    pgContainer, err := postgres.Run(ctx,
        "postgres:15-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(15*time.Second)),
    )
    if err != nil {
        return "", func() {}
    }

    connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")
    return connStr, func() {
        pgContainer.Terminate(ctx)
    }
}
```

```go
// your_integration_test.go - Test usage
func TestSomething_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    if testDBConnString == "" {
        t.Skip("Skipping integration test: database not available")
    }

    ctx := context.Background()
    ensureMigrations(t)  // Runs once for package

    // Use testPool for database operations
    repo := NewRepository(testPool)
    // ... test logic
}
```

**Benefits:**

- **85% faster** - One container vs multiple
- **Cleaner** - No duplicate setup code
- **Maintainable** - Change once, applies everywhere
- **Parallel-safe** - `sync.Mutex` protects migrations

**Critical: TestMain Location**  
TestMain MUST be in a file that contains test functions. Go will not invoke it from a `*_helpers.go` file.

**Test Data Isolation**  
With shared database, use unique test data:

```go
// Use timestamps or test names for uniqueness
userID := fmt.Sprintf("test-user-%d", time.Now().UnixNano())
username := fmt.Sprintf("user-%s", t.Name())
```

**Real Examples:**

- [`internal/database/postgres/`](file:///home/osse1/projects/BrandishBot_Go/internal/database/postgres/) - 9 files refactored, 85% faster
- [`internal/progression/`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/) - 4 files refactored, ~50% faster

---

## Code Coverage

### What Good Coverage Looks Like

- **Critical paths:** 90%+
- **Business logic:** 80%+
- **Utilities:** 95%+
- **Infrastructure:** 60%+

### Coverage is Not the Goal

```go
// ❌ This is 100% coverage but worthless:
func TestEverything(t *testing.T) {
    DoThing1()
    DoThing2()
    DoThing3()
    // No assertions!
}

// ✅ This is 60% coverage but valuable:
func TestCriticalPath(t *testing.T) {
    result := CriticalOperation(input)
    assert.Equal(t, expected, result)
    // Tests what matters
}
```

---

## Documentation Through Tests

Tests should document:

1. Expected behavior
2. Edge cases
3. Error conditions
4. Performance characteristics

```go
func TestItemStack_MaxSize(t *testing.T) {
    stack := NewItemStack()

    // Document: Stacks limited to 99 items
    for i := 0; i < 99; i++ {
        err := stack.Add(item)
        assert.NoError(t, err)
    }

    // Document: 100th item returns error
    err := stack.Add(item)
    assert.ErrorIs(t, err, ErrStackFull)
}
```

---

## Review Checklist

Before submitting tests, verify:

- [ ] Tests all 5 cases where applicable
- [ ] Test names clearly describe scenario
- [ ] Boundaries defined as named constants
- [ ] No magic numbers or strings
- [ ] Appropriate assertions used
- [ ] Mocks used minimally
- [ ] Tests are independent
- [ ] Fast execution (<100ms unit tests)
- [ ] Tests would fail if code broke
- [ ] Clear error messages on failure

---

## Examples from Codebase

**Excellent Examples:**

- [`config_test.go`](file:///home/osse1/projects/BrandishBot_Go/internal/config/config_test.go) - Environment handling, edge cases
- [`math_test.go`](file:///home/osse1/projects/BrandishBot_Go/internal/utils/math_test.go) - Property-based testing
- [`inventory_test.go`](file:///home/osse1/projects/BrandishBot_Go/internal/utils/inventory_test.go) - Real scenarios

**Needs Improvement:**

- Handler tests (minimal coverage)
- Middleware integration flows
- Gamble/Progression memory leak tests

---

## Future Enhancements

- [ ] Property-based testing framework (gopter?)
- [ ] Mutation testing to verify test quality
- [ ] Performance benchmarking suite
- [ ] Real-world data replay tests
- [ ] Chaos testing for distributed components

---

## Questions?

When in doubt:

1. Would this test catch a real bug?
2. Would it fail if the code broke?
3. Does it document expected behavior?

If yes to all three → Good test ✅
