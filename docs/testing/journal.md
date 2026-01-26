# Testing Lessons Learned Journal

A collection of practical insights gained from expanding the BrandishBot_Go testing infrastructure. These lessons are meant to help future contributors avoid common pitfalls and adopt proven patterns.

---

## Domain Model Changes

### Lesson 1: Field Renames Break Tests Silently
**Problem:** When `Item.Name` was renamed to `Item.InternalName`, no compile error occurred in existing test files because Go allows arbitrary struct fields. Tests only failed at runtime.

**Solution:** 
- Run `go build ./...` before committing domain changes
- Search codebase for old field names: `grep -r "Name:" internal/*_test.go`
- Consider compile-time interface assertions for critical types

**Pattern:**
```go
// In domain struct changes, add deprecation annotations
type Item struct {
    InternalName string `json:"internal_name"` // Renamed from Name
    // Name string // REMOVED - use InternalName
}
```

---

## Mock Management

### Lesson 2: Don't Duplicate Mocks Across Test Files
**Problem:** Created `MockRepository` in `memory_test.go` when it already existed in `service_test.go`. Go doesn't allow duplicate type definitions in the same package.

**Solution:**
- Check for existing mocks: `grep -n "type Mock" internal/package/*_test.go`
- Share mocks via `*_test.go` files in the same package
- For cross-package mocks, create a `testutil` package

**Pattern:**
```go
// In memory_test.go - reuse existing mocks
repo := new(MockRepository) // Defined in service_test.go

// DON'T redeclare:
// type MockRepository struct { mock.Mock }  ❌
```

### Lesson 3: Mock Return Types Must Match Exactly
**Problem:** `lootbox.DroppedItem{}` vs `domain.LootboxItem{}` caused type mismatches when mocking service calls.

**Solution:**
- Check interface definitions for exact return types
- View existing tests for correct usage patterns

**Pattern:**
```go
// Check the interface first
lootboxSvc.On("OpenLootbox", ...).Return([]lootbox.DroppedItem{}, nil)  ✅
lootboxSvc.On("OpenLootbox", ...).Return([]domain.LootboxItem{}, nil)   ❌
```

---

## Goroutine Leak Detection

### Lesson 4: Use Tolerance for Async Operations
**Problem:** Services with background XP awards or event publishing spawn goroutines that complete after the test ends.

**Solution:**
- Add small sleep before goroutine count check
- Use tolerance parameter in leak checker
- Understand which services spawn background tasks

**Pattern:**
```go
checker := leaktest.NewGoroutineChecker(t)

_, _ = svc.SomeAsyncOperation(ctx)

time.Sleep(100 * time.Millisecond) // Allow background tasks
checker.Check(1) // Tolerance of 1 goroutine
```

### Lesson 5: Not All Services Have Async Operations
**Problem:** Assumed progression service had background goroutines like economy/gamble.

**Reality:** Progression service is fully synchronous:
- No XP awards spawned
- Voting is pure state management
- All operations complete inline

**Insight:** Memory leak tests still valuable for:
- Validating clean design
- Catching future regressions if async logic is added
- Documentation of service behavior

---

## Struct Field Navigation

### Lesson 6: Use grep to Find Field Access Patterns
**Problem:** `session.Options[0].NodeKey` didn't exist; needed `session.Options[0].NodeDetails.NodeKey`.

**Solution:** Search existing tests for the correct pattern:
```bash
grep -n "session.Options\[0\]." internal/progression/*_test.go
```

**Pattern:** Always check how a type is used elsewhere before guessing at field names.

---

## Go Syntax Gotchas

### Lesson 7: Go Uses `nil`, Not `null`
**Problem:** JavaScript habit of writing `!= null` instead of `!= nil`.

**Pattern:**
```go
if session.Options[0].NodeDetails != nil { ✅
if session.Options[0].NodeDetails != null { ❌ // Compile error
```

### Lesson 8: Struct Field Names in Tests
**Problem:** Domain models evolve. Fields like `Name`, `ParentLevel` get renamed to `DisplayName`, `ParentUnlockLevel`.

**Solution:**
```bash
# Find struct definition first
go doc domain.ProgressionNode

# Or view the source
grep -A 20 "type ProgressionNode struct" internal/domain/*.go
```

---

## Test Organization

### Lesson 9: Integration Tests Should Skip in Short Mode
**Pattern:**
```go
func TestIntegration_ActualConfigFiles(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    // ... rest of test
}
```

**Benefits:**
- `go test ./... -short` runs fast for CI
- `go test ./...` runs full suite including integration
- Clear distinction between unit and integration tests

### Lesson 10: Testdata Directory for JSON Fixtures
**Pattern:**
```
internal/naming/
├── resolver.go
├── resolver_test.go
├── integration_test.go
└── testdata/
    ├── valid_aliases.json
    ├── valid_themes.json
    ├── malformed.json
    ├── missing_default.json
    └── invalid_dates.json
```

**Benefits:**
- Fixtures versioned with tests
- Easy to add edge cases
- Self-documenting test scenarios

---

## Coverage Strategies

### Lesson 11: Edge Cases Drive Coverage
High coverage comes from testing edge cases, not happy paths:

| Test Type | Coverage Value |
|-----------|---------------|
| Happy path only | ~50% |
| + Error cases | ~70% |
| + Edge cases | ~85% |
| + Race conditions | ~90%+ |

**High-value edge cases:**
- Empty inputs
- Missing fields
- Malformed data
- Concurrent access
- Timeout scenarios

### Lesson 12: Use `-cover` Early and Often
```bash
# Quick coverage check
go test ./internal/naming -cover

# Detailed HTML report
go test ./internal/naming -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Race Detection

### Lesson 13: Always Run Race Detector
**Pattern:**
```bash
go test -race ./internal/naming
```

**When to use:**
- Adding concurrent test patterns
- Testing services with mutex/RWMutex
- Before merging any PR

**Note:** Race detector is ~10x slower, so run selectively:
```bash
# Fast unit tests
go test ./... -short

# Full with race detection (CI only)
go test -race ./...
```

---

## Documentation Value

### Lesson 14: Test Names Are Documentation
**Good:**
```go
func TestStartGamble_NoGoroutineLeak(t *testing.T)
func TestLoadAliases_MalformedJSON(t *testing.T)
func TestGetDisplayName_FallbackBehavior(t *testing.T)
```

**Bad:**
```go
func TestService1(t *testing.T)
func TestHandler(t *testing.T)
func Test_Issue_123(t *testing.T)
```

### Lesson 15: Comment Non-Obvious Test Setup
```go
func TestExecuteGamble_NoGoroutineLeak(t *testing.T) {
    // ... mock setup ...
    
    // NOTE: Gamble must be in "Joining" state to execute
    // This test intentionally uses "Created" state to verify 
    // error path doesn't leak goroutines
    gamble := &domain.Gamble{State: domain.GambleStateCreated, ...}
```

---

## Quick Reference Commands

```bash
# Find mocks in a package
grep -n "type Mock" internal/package/*_test.go

# Find struct field usage
grep -rn "\.FieldName" internal/

# Check coverage
go test ./internal/package -cover

# Run with race detector
go test -race ./internal/package

# Skip integration tests
go test ./... -short

# View struct definition
go doc package.StructName
```

---

## Summary Checklist

Before adding tests to a new service:

- [ ] Check for existing mocks in `*_test.go` files
- [ ] Review interface definitions for exact return types
- [ ] Search existing tests for field access patterns
- [ ] Determine if service has async operations
- [ ] Create testdata directory if needed
- [ ] Add both happy path and edge case tests
- [ ] Run with race detector
- [ ] Skip integration tests in short mode

---

*Last updated: January 2026*

## Test Design Principles

### Lesson 16: Defensive Nil Checks Are Counterproductive in Tests
**Date:** January 26, 2026  
**Context:** Fixed nil pointer dereference in `internal/progression/integration_test.go`

**The Problem:**
Test accessed `newSession.Options` without checking if `newSession` was nil first, causing a panic.

**The Wrong Fix:**
Added defensive nil check:
```go
if newSession != nil {
    // Check session options
} else {
    // Verify target is set directly
}
```

This made the test pass but was **fundamentally wrong**.

**Why Defensive Nil Checks Are Bad:**

1. **They mask bugs** - If the implementation fails to create a session when it should, the defensive nil-check branch will pass silently, hiding a real bug. Tests should fail loudly to expose problems.

2. **They make expectations unclear** - A test with `if session exists do X, else do Y` doesn't clearly communicate what the expected behavior is. Tests should be unambiguous specifications.

3. **They enable non-determinism** - When a test doesn't know what state to expect, it indicates the test setup is wrong or the test is testing multiple scenarios at once.

**The Right Approach:**
Tests should **assert expected behavior** based on **known state**:

```go
// After unlocking lootbox0, 4 options become available:
// - money (root child, still available)
// - upgrade, disassemble, search (lootbox0 children, now unlocked)
// Since 4 options remain (≥2), a voting session SHOULD be created.

newSession, _ := repo.GetActiveSession(ctx)
assert.NotNil(t, newSession, "A new voting session should be created (4 options available)")
assert.NotEqual(t, session.ID, newSession.ID, "Should be a different session")
```

**Key Principle:** If you know the test tree structure and what was unlocked, you should know exactly whether a session will be created or not.

### Lesson 17: Make Integration Tests Deterministic
**Related to:** Lesson 16

**Problem:** Tests voted for `session.Options[0]`:

```go
nodeKey := session.Options[0].NodeDetails.NodeKey  // Could be money OR lootbox0
service.VoteForUnlock(ctx, "discord", "user1", nodeKey)
```

This was non-deterministic because:
- If money wins: Only lootbox0 remains → no session created
- If lootbox0 wins: 4 options remain → session created

**Solution:** Make tests deterministic by explicitly choosing which node to vote for:

```go
// Find lootbox0 option
var lootboxKey string
for _, opt := range session.Options {
    if opt.NodeDetails.NodeKey == "item_lootbox0" {
        lootboxKey = opt.NodeDetails.NodeKey
        break
    }
}
if lootboxKey == "" {
    t.Fatal("lootbox0 not found in session options")
}

service.VoteForUnlock(ctx, "discord", "user1", lootboxKey)
```

Now the test always votes for lootbox0, making the outcome predictable and assertions clear.

**General Testing Principles:**

✅ **Do:**
- Assert expected behavior based on known state
- Make tests deterministic
- Fail loudly when behavior is incorrect
- Document test expectations with comments explaining the "why"

❌ **Don't:**
- Add defensive nil checks "just in case"
- Write tests that handle multiple possible outcomes
- Use `Options[0]` when the order matters for the test outcome
- Future-proof tests against behavior changes (tests should break when behavior changes!)

**Key Quotes:**
> "Tests should verify expected behavior, not handle uncertainty."

> "Nil checks mask real bugs. If the code unexpectedly doesn't create a session when it should, the nil-check branch will pass silently, hiding the bug. The test should FAIL loudly to expose the problem."
