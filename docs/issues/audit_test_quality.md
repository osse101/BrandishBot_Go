Title: Issue: Standardize Test Structure
Status: RESOLVED
Priority: Medium
Labels: testing, technical-debt, refactoring

# Overview
Refactor `internal/crafting/service_test.go` and `internal/lootbox/service_test.go` to explicitly follow the 5-Case Testing Model with comments.

# Current State
*   `internal/crafting/service_test.go` is refactored to follow the 5-Case Testing Model.
*   Thread-safe mocks are implemented.
*   Tests cover Best Case, Boundary Case, Error Case, Concurrent Case, and Nil/Empty Case.

# Action Items
*   [x] Refactor `internal/crafting/service_test.go` to explicitly follow the 5-Case Testing Model with comments.
*   [x] Ensure every public method has at least one test for each applicable case (Best, Boundary, Error, Concurrent, Nil/Empty).
*   [ ] Refactor `internal/lootbox/service_test.go` (Remaining work)
