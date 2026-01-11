RESOLVED

# Code Quality Audit Report

This report lists the issues identified after removing all linting suppressions (`//nolint`) and reverting configuration shortcuts.

## Status Update
- **Fixed**: `SellItem` complexity (refactored).
- **Fixed**: `Rollback` error handling (implemented `SafeRollback` helper).
- **Fixed**: Unchecked Encode/Write errors (all HTTP handlers now properly handle errors).
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

## 4. Unchecked Errors (errcheck) - **FIXED**
**Resolution Date**: 2026-01-05

All unchecked errors have been resolved:
- **HTTP Response Encoding**: Removed redundant `WriteHeader` calls in `inventory.go` and `stats.go` (were calling it before `respondJSON` which already handles headers)
- **Handler Package**: All handlers now use `respondJSON()` helper which logs encoding errors
- **Discord Package**: Added error checking for `Encode()` calls in `server.go`, `health.go`, and `test_helper.go`
- **Version Handler**: Converted to use `respondJSON()` helper, removing unused `json` import

All changes verified with:
- ✅ `go build ./...` - no errors
- ✅ `go test ./internal/handler/...` - all tests pass
- ✅ `go test ./internal/discord/...` - all tests pass

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
