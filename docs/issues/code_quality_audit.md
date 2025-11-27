# Code Quality Audit Report

This report lists the issues identified after removing all linting suppressions (`//nolint`) and reverting configuration shortcuts.

## Status Update
- **Fixed**: `SellItem` complexity (refactored).
- **Fixed**: `Rollback` error handling (implemented `SafeRollback` helper).
- **Remaining**: See below.

## 1. Test Coverage Deficit
- **Current Coverage**: 35.6%
- **Target Threshold**: 80%
- **Gap**: 44.4%

## 2. Cyclomatic Complexity (gocyclo)
The following functions exceed the complexity threshold (15):
- `internal/economy/service.go`: `BuyItem` (20)
- `internal/user/item_handlers.go`: `processLootbox` (19)
- `internal/user/service.go`: `HandleSearch` (19)
- `internal/crafting/service.go`: `UpgradeItem` (26), `DisassembleItem` (22)
- **Integration Tests**: `TestUserRepository_Integration`, `TestStatsRepository_Integration`, `TestProgressionRepository_Integration`.

## 3. Security Vulnerabilities (gosec)
- **Weak RNG**: `internal/utils/math.go` (G404). *Note: Suppressed with explanation as it is for game logic.*

## 4. Unchecked Errors (errcheck)
- **HTTP Response Encoding**: `json.NewEncoder(w).Encode(...)` errors are ignored in almost all handlers.
- **HTTP Writes**: `w.Write` errors are ignored in `inventory.go` and `stats.go`.

## 5. Code Duplication (dupl)
- **Progression Repository**: `GetNodeByKey` vs `GetNodeByID`.
- **Stats Repository**: `GetEventsByUser` vs `GetEventsByType`.
- **Progression Handlers**: `HandleAdminUnlock` vs `HandleAdminRelock`.

## 6. String Constants (goconst)
- `lootbox1` in `internal/crafting/service_test.go`.
- `You have found nothing` in `internal/user/search_test.go`.

## Recommendations
1.  **Continue Refactoring**: Apply the `SellItem` refactoring pattern (helper functions) to `BuyItem`, `UpgradeItem`, etc.
2.  **Fix Encode Errors**: Update handlers to check `Encode` errors and log them.
3.  **Increase Test Coverage**: Add tests for domain logic.
