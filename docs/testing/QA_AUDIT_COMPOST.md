# QA & Testing Audit Report: `internal/compost`

## Overview

This report provides a QA and testing audit of the `internal/compost` package (`deposit_test.go`, `harvest_test.go`, and `engine_test.go`) in the BrandishBot_Go project. The audit focuses on Go testing best practices, concurrency handling, unit test coverage, integration testing, and end-to-end testing recommendations.

## 1. Unit Testing & Test Structure

### Current State

- **Good Coverage on Business Logic:** `engine_test.go` has excellent coverage (table-driven tests) for value calculations, ready times, and dominant type determination. It covers various input combinations well.
- **Service Tests exist:** `deposit_test.go` and `harvest_test.go` effectively use `stretchr/testify` (`assert`, `require`, `mock`) to verify service operations and database interactions via mocks.
- **Table-Driven Tests:** Used extensively in `engine_test.go`, but missing in `deposit_test.go` and `harvest_test.go`. Instead, these files use separate `t.Run` calls with repeated setup code.

### Areas for Improvement (The 5-Case Model)

According to the project's `TEST_GUIDANCE.md`, unit tests should cover the 5-Case Model (Best, Boundary, Edge, Invalid, Hostile).

- **Missing Boundary Tests:** The current tests do not explicitly verify boundary cases (e.g., exact maximum capacity vs capacity + 1, exactly 0 quantity vs -1 quantity, exactly at the sludge timer vs 1 second before/after).
- **Repetitive Setup:** `deposit_test.go` and `harvest_test.go` repeat mock setup logic (e.g., `mockRepo`, `mockUserRepo`, `mockProgressionSvc`, `mockJobSvc`) in almost every `t.Run` block. This violates DRY and makes tests harder to maintain.
- **Missing Edge/Hostile Cases:** What happens if `Items` slice is nil? What if an item has no `ContentType` (partially addressed in engine but not at service level)? What if a user attempts to deposit 10,000 items at once?

### Recommendations

1.  **Refactor to Table-Driven Tests:** Convert `deposit_test.go` and `harvest_test.go` to use table-driven tests. This will drastically reduce boilerplate and make adding new test cases (like boundary and edge cases) trivial.
2.  **Add Boundary Tests:** Explicitly add tests for:
    - Bin capacity limits (Boundary: exactly full, just over full).
    - Time boundaries (exactly at `ReadyAt`, exactly at `SludgeAt`).
3.  **Centralize Mock Setup:** Create a helper function (e.g., `setupCompostServiceMocks(t *testing.T)`) that returns the initialized mocks and service, allowing individual tests to just specify the expected mock behaviors (`.On(...)`).

## 2. Concurrency & Race Conditions

### Current State

- The `Service` struct contains a `sync.WaitGroup` (`wg`) and a `Shutdown` method.
- Background processing (like async event publishing) seems to be intended (though not explicitly tested in the currently reviewed files).

### Areas for Improvement

- **Missing Concurrency Tests:** There are no tests verifying thread safety. If two users try to harvest the exact same bin at the exact same millisecond, what happens? (Database transactions handle this, but the Go logic should be verified against race conditions).
- **Missing Goroutine Leak Tests:** Unlike `internal/economy/memory_test.go`, the compost tests do not use `leaktest.NewGoroutineChecker(t)`. If the `Harvest` or `Deposit` operations spawn background goroutines (e.g., publishing events), tests must ensure they don't leak.
- **Shutdown Not Tested:** The `Shutdown(ctx)` method is defined in `service.go` but never called or tested in `deposit_test.go` or `harvest_test.go` to ensure `wg.Wait()` behaves correctly.

### Recommendations

1.  **Add Race Tests:** Create a test that calls `Deposit` or `Harvest` concurrently from multiple goroutines (using `t.Parallel()` and wait groups) to ensure no panics or deadlocks occur. Use `go test -race`.
2.  **Implement Leak Testing:** Add `defer svc.Shutdown(context.Background())` and `leaktest` checks to all service tests to ensure no background event publishing goroutines are orphaned.

## 3. Integration Testing

### Current State

- The tests in `internal/compost` are strictly unit tests using interface mocks.
- There are no integration tests verifying the actual database queries or transaction behavior against a real PostgreSQL instance.

### Areas for Improvement

- **Mock Verification:** Mocks prove the _code_ calls the right interface methods, but not that the _database_ executes correctly. For example, `tx.GetBinForUpdate(ctx, userID)` uses a `SELECT ... FOR UPDATE` lock in PostgreSQL. Mocks cannot verify if this lock actually works or prevents race conditions at the DB level.

### Recommendations

1.  **Create Integration Tests:** Following the project's standard (using `testcontainers` and the TestMain pattern as seen in `internal/database/postgres/integration_test.go`), create an `internal/compost/integration_test.go`.
2.  **Test DB Locks:** Write an integration test that attempts concurrent deposits/harvests against the real test container to prove that `GetBinForUpdate` correctly sequences the operations and prevents duplicate harvests or over-capacity deposits.

## 4. End-to-End (E2E) Testing Recommendations

### Current State

- E2E testing is outside the scope of package-level tests, but crucial for features tied to external platforms (Discord/Twitch).

### Recommendations

- **Simulate Full Flow:** E2E tests should cover the entire lifecycle:
  1. User executes `!deposit apple 5` (Handler -> Service -> DB).
  2. Time advances (or mock time is used in the test environment).
  3. User executes `!harvest` (Handler -> Service -> DB -> Event Bus -> XP/Progression).
- **Verify Cross-Service Impacts:** Ensure that a successful harvest correctly triggers the event bus, which in turn awards Farming Job XP (handled by `job.Service`). The E2E test should verify the user's XP actually increased in the database.

## Summary of Action Items for Dev Team

1. Refactor `deposit_test.go` and `harvest_test.go` to use table-driven tests and reduce mock boilerplate.
2. Add explicit boundary tests for capacity, time, and quantities.
3. Integrate `leaktest` and explicitly call `Shutdown()` in tests to prevent goroutine leaks.
4. Add concurrent race-condition tests.
5. Create an `integration_test.go` using Testcontainers to verify `SELECT FOR UPDATE` transaction locking logic in Postgres.
