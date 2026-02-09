# Audit Report: `internal/handler/admin_daily_reset_test.go`

## Overview
This document audits the test file `internal/handler/admin_daily_reset_test.go` for quality, test coverage, and best practices.

## Current State
- **Coverage:** The tests cover the two main handlers: `HandleManualReset` and `HandleGetResetStatus`.
- **Structure:** The tests are currently written as separate functions (`TestHandleManualReset_Success`, `TestHandleManualReset_ServiceError`, etc.).
- **Assertions:** Uses `stretchr/testify/assert` for HTTP status and response body string checks.
- **Mocking:** Uses `mockery` generated mocks (`mocks.NewMockJobService`).

## Issues / Findings
1.  **Code Duplication:** There is significant repetition between the success and error test cases for each handler. This violates DRY principles and makes maintenance harder.
2.  **Lack of Table-Driven Tests:** Go best practices strongly favor table-driven tests for testing multiple scenarios (success, error, edge cases) within a single test function.
3.  **Concurrency:** Tests are not running in parallel (`t.Parallel()`). Since these are unit tests with isolated mocks, they should be parallelizable for faster execution.
4.  **Brittle Assertions:** String-based checks like `assert.Contains(t, w.Body.String(), ...)` are fragile. Changes to JSON formatting (e.g., whitespace) could break tests without changing logic. It is better to decode the JSON response into a struct or map and assert specific fields.
5.  **Weak Mock Expectations:** `mock.Anything` is used for context parameters. While acceptable, stricter checking (e.g., verifying the context type or values) is preferable.
6.  **Missing Factory Test:** `NewAdminDailyResetHandler` is not explicitly tested.

## Recommendations
1.  **Refactor to Table-Driven Tests:** combine related test cases into a single function using a slice of test structs.
2.  **Enable Parallel Execution:** Add `t.Parallel()` to the top-level test functions and within the sub-tests (`t.Run`).
3.  **Use JSON Decoding:** Unmarshal response bodies into structs/maps to verify exact values (e.g., `records_affected`, error messages).
4.  **Add Factory Test:** Add a simple test for `NewAdminDailyResetHandler`.

## Action Plan
- Rewrite `TestHandleManualReset` and `TestHandleGetResetStatus` using table-driven tests.
- Add `TestNewAdminDailyResetHandler`.
- Verify all tests pass.
