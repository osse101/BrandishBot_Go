# Test Audit: internal/progression

This document outlines the findings of an audit performed on the `internal/progression` package tests.

## 1. Organization

### Duplicate Mocks
There is a significant duplication of mocking effort:
- **Hand-rolled Mocks:** `internal/progression/service_test.go` defines a comprehensive `MockRepository` struct with in-memory logic. `internal/progression/mock_user_test.go` defines `MockUser`.
- **Generated Mocks:** `internal/progression/mocks/mock_repository.go` contains a `mockery`-generated mock for the same `Repository` interface.

**Impact:** The unit tests primarily use the hand-rolled mock. This mock implements complex business logic (e.g., filtering in `GetNodeByFeatureKey`), which means tests are verifying against a "fake implementation" rather than just interface expectations. This can lead to false positives if the fake implementation diverges from the real repository logic.

### File Naming
- **Ambiguity:** `test_helper.go` (defines `TestHelper` struct) and `test_helpers.go` (defines `TestMain` and setup logic) have confusingly similar names.
- **Service vs Logic:** `voting_sessions_test.go` tests internal helper functions (`findWinningOption`, `selectRandomNodes`), while `voting_session_comprehensive_test.go` tests the service layer methods. The naming does not clearly distinguish between unit tests of internals and service tests.

### Obsolete Code
- `TestVoteForUnlock` in `service_test.go` is explicitly skipped with the message "Obsolete test - voting now uses session-based system".

## 2. Robustness

### Integration Tests
- The package uses `testcontainers` in `service_integration_test.go`, which provides excellent robustness by testing against a real PostgreSQL instance.
- **Known Bugs:** The integration test `testZeroCostAutoUnlock` explicitly checks for a known bug (orphaned voting sessions). This is good for documentation but indicates unresolved issues.

### Manual Migrations
- The `test_helpers.go` file manually parses and executes SQL migration files. While functional, this is more brittle than using the standard `goose` library or binary used in production/deployment.

### Concurrency
- The hand-rolled `MockRepository` uses `sync.RWMutex` but explicitly notes it is "NOT thread-safe by design" for multi-instance contexts. While acceptable for unit tests, this mismatch with production architecture (which relies on DB locks) suggests reliance on integration tests is crucial.

## 3. Quality

### Coverage
- The test suite appears comprehensive, covering standard flows, edge cases, and some race conditions (`race_condition_test.go`).
- The existence of specific files like `fk_constraint_test.go` shows attention to database-level constraints.

### Recommendations
1. **Consolidate Mocks:** Transition unit tests to use the generated `mocks/mock_repository.go` where possible to avoid maintaining the complex logic in the hand-rolled mock.
2. **Rename Helpers:** Rename `test_helpers.go` to `setup_test.go` and `test_helper.go` to `test_utils.go` for clarity.
3. **Clarify Voting Tests:** Rename `voting_sessions_test.go` to `voting_logic_test.go` to distinguish it from service-level tests.
4. **Remove Dead Code:** Delete the obsolete `TestVoteForUnlock`.
