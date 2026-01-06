# Standardization: Logging Best Practices

**Created:** 2026-06-18
**Status:** Proposed
**Priority:** Low
**Labels:** cleanup, refactor, logging

## Summary

Replace standard library logging (`log.Println`, `fmt.Printf`) with the project's structured logger (`internal/logger`) to ensure consistent log formatting, levels, and integration with the centralized logging system.

## Background

The project uses a structured logger (`internal/logger`) for most operations. However, there are scattered instances of `log.Println` and `fmt.Printf` (and `fmt.Print`) used for logging information and errors. This bypasses the structured logging context (e.g., request IDs, correlation IDs) and makes log parsing and monitoring more difficult.

## Related Files

-   `internal/database/database.go`
-   `internal/database/postgres/integration_test.go`
-   `internal/database/postgres/job.go`
-   `internal/user/service.go`
-   `internal/server/server.go`
-   `internal/worker/pool.go` (and other worker files)

## Proposed Enhancements

1.  **Replace `log.Println`**: Use `logger.Info`, `logger.Error`, etc., passing the appropriate context if available.
2.  **Replace `fmt.Printf`**: Use structured logging methods. For CLI output (if any), ensure it's intentional; otherwise, switch to logging.
3.  **Context Propagation**: Ensure `context.Context` is available where logging is needed to allow `logger.FromContext(ctx)` to work effectively.

## Implementation Plan

1.  **Identify Usage**: grep for `log.Println`, `fmt.Printf`, `fmt.Println` in `internal/`.
2.  **Refactor**:
    -   In `internal/database/database.go`: Use structured logger for connection status.
    -   In `internal/server/server.go`: Use structured logger for server startup messages.
    -   In `internal/user/service.go`: Replace `fmt.Printf` with `log.Info` or `log.Debug`.
    -   In `internal/database/postgres/job.go`: Replace `fmt.Printf` with `log.Info`.
    -   In tests (`integration_test.go`): `fmt.Printf` might be acceptable for test output, but consider using `t.Logf`.
3.  **Verify**: Run the application and check logs to ensure messages appear in the correct format.

## Success Criteria

-   Zero usages of `log.Println` and `fmt.Printf` for application logging in `internal/`.
-   Consistent structured logging across the application.
