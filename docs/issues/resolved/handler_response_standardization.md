RESOLVED

# Standardization: JSON Response Handling

**Created:** 2026-06-18
**Status:** Resolved
**Priority:** Low
**Labels:** cleanup, refactor, technical-debt

## Summary

Standardize JSON response handling across all HTTP handlers by utilizing consistent helper functions defined in `internal/handler/responses.go` and removing repetitive boilerplate code.

## Background

The codebase currently contains duplicate logic for encoding JSON responses and handling errors. While helper functions like `respondJSON` and `respondError` exist, they are not consistently used across all handlers (e.g., `HandleSearch`, `HandleAdmin`, `HandleMessage`). This leads to code duplication and potential inconsistencies in error handling and logging.

## Related Files

- `internal/handler/responses.go` (Helper definitions)
- `internal/handler/search.go`
- `internal/handler/admin.go`
- `internal/handler/message.go`
- `internal/handler/user.go`
- `internal/handler/gamble.go`
- `internal/handler/linking.go`
- `internal/handler/job.go`

## Proposed Enhancements

1.  **Refactor Handlers**: Update all HTTP handlers to use `respondJSON` for successful responses and `respondError` (or a variation) for error responses.
2.  **Consolidate Error Handling**: Ensure that `respondJSON` handles JSON encoding errors consistently (logging them as it currently does).
3.  **Remove Boilerplate**: Delete repetitive `json.NewEncoder(w).Encode(...)` blocks.

## Implementation Plan

1.  **Audit**: Identify all instances of `json.NewEncoder(w).Encode` in `internal/handler`.
2.  **Refactor**: Replace direct encoding calls with `respondJSON` or `respondError`.
3.  **Verify**: Ensure no functionality is lost and that headers (Content-Type, Status Code) are set correctly by the helpers.
4.  **Test**: Run existing handler tests to confirm no regressions.

## Success Criteria

-   Reduction in lines of code in handler files.
-   Consistent usage of `respondJSON` and `respondError`.
-   No regression in API behavior.
