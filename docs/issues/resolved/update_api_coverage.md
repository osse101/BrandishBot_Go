# API Coverage Documentation Discrepancy

**Date:** 2026-02-23
**Status:** Closed
**Priority:** Low

## Summary

The file `docs/api/API_COVERAGE.md` is significantly outdated compared to the actual codebase. While the counts have been updated, the detailed lists of endpoints and commands need a thorough audit and update.

## Discrepancies Found (as of Feb 2026)

| Metric             | Documented (Old) | Actual (Feb 2026) |
| ------------------ | ---------------- | ----------------- |
| API Endpoints      | 59               | 97                |
| Discord Commands   | 41               | 60                |
| C# Client Methods  | 59               | 94                |
| C# Wrapper Methods | 59               | 65                |

## Action Items

1.  **Update API Endpoint Table**: Go through `internal/server/server.go` and list all 97 endpoints in the table.
2.  **Update Discord Command Table**: Go through `internal/discord/cmd_*.go` and list all 60 commands.
3.  **Update C# Client/Wrapper Tables**: Verify the methods in `client/csharp/` match the documented ones.
4.  **Verify Missing Features**: Check if the "Missing Items" section is still accurate.
5.  **Standardize Verification Commands**: Ensure the verification commands at the bottom of the document are accurate and robust.

## References

- `internal/server/server.go`
- `internal/discord/cmd_*.go`
- `client/csharp/BrandishBotClient.cs` (and partial files)
- `client/csharp/BrandishBotWrapper.cs`
