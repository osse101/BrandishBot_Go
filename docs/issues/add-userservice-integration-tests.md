RESOLVED

# Add Integration Tests for UserService and Async Operations

## Priority: Medium
## Labels: `testing`, `enhancement`, `good-first-issue`

---

## Background

Recent concurrency bug fixes revealed the value of comprehensive integration testing. Two critical bugs were caught by our integration test suite:

### Recent Bugs Caught by Integration Tests ✅

1. **AddItem Lost Updates** ([fixed 2025-12-27](file:///home/osse1/projects/BrandishBot_Go/internal/database/postgres/user.go#L40-L65))
   - **Symptom**: 19 out of 20 concurrent inventory additions were lost
   - **Root Cause**: `SELECT FOR UPDATE` on non-existent inventory rows returned no lock
   - **Test**: `TestConcurrentAddItem_Integration` 
   - **Fix**: Ensure row exists before locking with `INSERT ON CONFLICT DO NOTHING`

2. **Cooldown Race Condition** ([fixed 2025-12-27](file:///home/osse1/projects/BrandishBot_Go/internal/cooldown/postgres.go#L79-L127))
   - **Symptom**: All 10 concurrent requests succeeded when only 1 should (cooldown not enforced)
   - **Root Cause**: `SELECT FOR UPDATE` doesn't work when cooldown row doesn't exist
   - **Test**: `TestCooldownService_ConcurrentRequests_Integration`
   - **Fix**: Use PostgreSQL advisory locks that work without existing rows

**Impact**: Both bugs would have caused data corruption in production. Integration tests caught them before deployment.

---

## Problem

While our integration test coverage is excellent (12/12 tests passing, 100%), there are gaps in testing complex user service operations and async behavior:

1. **UserService** has no direct integration tests despite complex inventory logic
2. **Async XP awards** run in goroutines but aren't tested for graceful shutdown
3. **Cross-user operations** (GiveItem) lack concurrency testing

---

## Recommended Tests

### 1. UserService Inventory Integration Test (HIGH PRIORITY) ⭐

**File**: `internal/database/postgres/user_service_integration_test.go`

**Why**: Would have caught the AddItem race condition earlier in the development cycle.

**Test Scenarios**:
```go
func TestUserService_InventoryOperations_Integration(t *testing.T) {
    t.Run("Concurrent GiveItem Between Users", func(t *testing.T) {
        // Test 10 concurrent transfers from userA to userB
        // Verify both inventories update correctly with no lost transfers
    })
    
    t.Run("UseItem Handler Execution", func(t *testing.T) {
        // Test item handlers execute correctly in real DB context
        // Verify inventory updates and handler side effects
    })
    
    t.Run("AddItem to Full Inventory", func(t *testing.T) {
        // Test boundary conditions and error handling
    })
}
```

**Estimated Work**: 2 hours

---

### 2. Async XP Award Verification (HIGH PRIORITY) ⭐

**File**: `internal/user/async_integration_test.go`

**Why**: Async operations can hide bugs that only appear during graceful shutdown.

**Test Scenarios**:
```go
func TestUserService_AsyncXPAward_Integration(t *testing.T) {
    t.Run("Shutdown Waits for XP Awards", func(t *testing.T) {
        // Trigger 100 async XP awards
        // Call service.Shutdown()
        // Verify all awards complete before shutdown returns
    })
    
    t.Run("XP Awards Don't Block Main Flow", func(t *testing.T) {
        // Verify search completion doesn't wait for XP award
    })
}
```

**Code Reference**: [`user/service.go:762`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service.go#L762)

**Estimated Work**: 1 hour

---

### 3. GiveItem Concurrency Test (MEDIUM PRIORITY)

**File**: `internal/database/postgres/give_item_integration_test.go`

**Why**: Similar to AddItem but involves two users' inventories.

**Test Scenarios**:
```go
func TestConcurrentGiveItem_Integration(t *testing.T) {
    // 20 concurrent transfers: userA -> userB
    // Verify final state matches expected totals
    // Ensure no duplicated or lost items
}
```

**Estimated Work**: 1 hour

---

### 4. JobService Integration Test (MEDIUM PRIORITY)

**File**: `internal/job/integration_test.go`

**Why**: Validates XP/level calculations with real persistence.

**Test Scenarios**:
```go
func TestJobService_XPCalculation_Integration(t *testing.T) {
    t.Run("Level Up Thresholds", func(t *testing.T) {
        // Award XP incrementally and verify level-ups
    })
    
    t.Run("Critical Success Multipliers", func(t *testing.T) {
        // Verify bonus XP calculations persist correctly
    })
}
```

**Estimated Work**: 1.5 hours

---

## Current Test Status

✅ **Integration Tests**: 12/12 passing (100%)  
✅ **Unit Tests**: 71 test files  
✅ **Concurrency Coverage**: AddItem, Cooldowns validated  
✅ **Code Coverage**: 60-95% across most packages

**Test Infrastructure**: Modern (testcontainers, proper isolation) ✅

---

## Acceptance Criteria

- [ ] `TestUserService_InventoryOperations_Integration` passes with concurrent GiveItem
- [ ] `TestUserService_AsyncXPAward_Integration` validates graceful shutdown
- [ ] `TestConcurrentGiveItem_Integration` validates inventory consistency
- [ ] `TestJobService_XPCalculation_Integration` validates XP persistence
- [ ] All existing tests continue to pass
- [ ] Test coverage remains above 60% for affected packages

---

## Notes

- These tests complement existing repository-level integration tests
- Focus on service-layer behavior with real database interactions
- Priority based on potential production impact
- **Total Estimated Work**: 5.5 hours

---

## References

- [Test Coverage Analysis](file:///home/osse1/.gemini/antigravity/brain/bf537a7d-c2e1-47d7-9ad4-f25090434ebd/test_coverage_analysis.md)
- [Recent Bug Fixes Walkthrough](file:///home/osse1/.gemini/antigravity/brain/bf537a7d-c2e1-47d7-9ad4-f25090434ebd/walkthrough.md)
- [Integration Test Suite](file:///home/osse1/projects/BrandishBot_Go/internal/database/postgres)
