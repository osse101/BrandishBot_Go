Title: Audit: Test Quality & Code Efficiency

# Overview
This issue summarizes the findings from a codebase audit focused on test quality, coverage (~65% average), and code efficiency.

# Test Quality

## Strengths
*   **Structured Testing**: `internal/user/search_test.go` sets a high standard with explicit "5-Case Testing Model" comments (Best, Boundary, Error, Concurrent, Nil/Empty).
*   **Statistical Testing**: `internal/user/search_test.go` includes statistical tests for RNG mechanics (e.g., `TestHandleSearch_NearMiss_Statistical`), ensuring probability distributions are correct.
*   **Integration Testing**: `internal/database/postgres/user_service_integration_test.go` uses `testcontainers` for robust integration testing against a real Postgres instance, covering concurrency and race conditions.
*   **Generated Mocks**: `internal/gamble` uses `mock.Mock` embedding effectively, which is cleaner than manual struct mocks.

## Weaknesses & Technical Debt
*   **Manual Mocks**: `internal/user` and `internal/crafting` rely heavily on manually maintained mock structs (e.g., `MockRepository` in `service_test.go`). These are brittle, require manual updates whenever interfaces change, and obscure test logic.
*   **Inconsistent 5-Case Model**: While `search_test.go` is explicit, other packages like `crafting` and `lootbox` implicitly cover some cases but lack the structured approach and comments, making it harder to verify if all edge cases are covered.
*   **Hardcoded Test Data**: Tests in `internal/user` often use hardcoded IDs (e.g., `1`, `3`), which makes them fragile to configuration changes.
*   **Lack of Benchmarks**: `internal/crafting` and `internal/gamble` lack performance benchmarks, which is critical for complex state transitions and inventory operations.

# Integration Tests
*   The existing integration tests in `internal/database/postgres` are high quality but are skipped if Docker is not available.
*   **Missing Integration**: There are no full integration tests for `gamble` flows (Start -> Join -> Execute -> Settlement), which involve complex state transitions and transactions.

# Code Efficiency

## Optimizations Verified
*   **Batch Operations**: `internal/lootbox` and `internal/crafting` correctly use batch lookups (`GetItemsByNames`, `GetItemsByIDs`) to avoid N+1 query performance issues.
*   **Inventory Updates**: `internal/crafting/service.go` uses `utils.AddItemsToInventory` for efficient batch inventory updates.
*   **Concurrency**: `internal/user/service.go` uses `sync.RWMutex` for item caching, reducing DB load.

## Areas for Improvement
*   **Benchmark Coverage**: Add benchmarks for `Crafting` (especially recipe lookup and recursive material checks if added later) and `Gamble` settlement logic.

# Action Items

## 1. Refactor Mocks
*   [ ] Replace manual `MockRepository` structs in `internal/user` and `internal/crafting` with generated mocks (using `vektra/mockery` or `stretchr/testify/mock`).

## 2. Standardize Test Structure
*   [ ] Refactor `internal/crafting/service_test.go` and `internal/lootbox/service_test.go` to explicitly follow the 5-Case Testing Model with comments.
*   [ ] Ensure every public method has at least one test for each applicable case (Best, Boundary, Error, Concurrent, Nil/Empty).

## 3. Expand Integration Tests
*   [ ] Add `gamble_integration_test.go` to `internal/database/postgres` to test the full gamble lifecycle with real database transactions.

## 4. Add Benchmarks
*   [ ] Add benchmarks for `CraftingService.UpgradeItem` and `DisassembleItem`.
*   [ ] Add benchmarks for `GambleService.ExecuteGamble` with varying numbers of participants.
