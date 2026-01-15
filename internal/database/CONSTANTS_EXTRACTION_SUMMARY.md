# Database Module Constants Extraction Summary

## Overview
This document summarizes the extraction of hard-coded constants from the `internal/database` module (excluding generated code).

## Files Created

### 1. `/internal/database/constants.go`
**Purpose:** Top-level database constants for connection pooling and core operations.

**Constants Extracted:**
- `DefaultMinConnections = 2` - Minimum pool connections
- Error messages for connection operations
- Log messages for database connection status

**Usage Example:**
```go
// Before:
config.MinConns = 2

// After:
config.MinConns = DefaultMinConnections
```

### 2. `/internal/database/postgres/constants.go`
**Purpose:** Comprehensive constants for PostgreSQL repository implementations.

**Categories of Constants:**

#### PostgreSQL Error Codes
- `PgErrorCodeUniqueViolation = "23505"` - Used in gamble.go for duplicate detection

#### Platform Names (8 occurrences)
- `PlatformNameTwitch = "twitch"`
- `PlatformNameYouTube = "youtube"`
- `PlatformNameDiscord = "discord"`
- Used in: user.go (lines 103-106, 136-142), user_linking.go (lines 40-46, 95-103)

#### Event Types
- `EventTypeProgressionNodeUnlocked = "progression.node_unlocked"`
- `EventTypeProgressionNodeRelocked = "progression.node_relocked"`
- `EventVersion1_0 = "1.0"`
- Used in: progression.go (lines 204, 240)

#### Engagement Metric Types
- `EngagementMetricMessage = "message"`
- `EngagementMetricCommand = "command"`
- `EngagementMetricItemCrafted = "item_crafted"`
- `EngagementMetricItemUsed = "item_used"`
- Used in: progression.go (lines 521-528, 556-561)

#### Default Engagement Weights
- Fallback values when database weights are unavailable
- Used in: progression.go (lines 556-561)

#### Inventory Constants
- `EmptyInventoryJSON = "{\"slots\": []}"` - Default inventory structure
- Used in: user.go (line 131)

#### Error Messages (200+ constants)
Organized by functional area:
- Transaction Operations (4 messages)
- User Operations (11 messages)
- Inventory Operations (10 messages)
- Item Operations (13 messages)
- Platform Operations (7 messages)
- Cooldown Operations (3 messages)
- Recipe Operations (21 messages)
- Economy Operations (2 messages)
- Gamble Operations (9 messages)
- Job Operations (11 messages)
- Stats Operations (9 messages)
- Progression Operations (40 messages)
- Voting Session Operations (8 messages)
- Unlock Progress Operations (6 messages)
- Leaderboard Operations (1 message)
- Junction Table Operations (2 messages)
- Linking Operations (3 messages)
- Conversion Operations (1 message)

#### Log Messages
- Job operation logs
- Event publishing logs
- Engagement weight conversion warnings

#### Database Operation Descriptions
- `OpDescGetInventory = "get inventory"`
- `OpDescGetInventoryForUpdate = "get inventory for update"`

## Files Modified

### `/internal/database/database.go`
**Lines Modified:** 21-44
**Changes:**
- Replaced hard-coded error messages with constants
- Replaced hard-coded `2` with `DefaultMinConnections`
- Replaced log message with constant

**Before:**
```go
return nil, fmt.Errorf("failed to parse connection string: %w", err)
config.MinConns = 2
slog.Default().Info("Successfully connected to the database")
```

**After:**
```go
return nil, fmt.Errorf("%s: %w", ErrMsgFailedToParseConnString, err)
config.MinConns = DefaultMinConnections
slog.Default().Info(LogMsgSuccessfullyConnectedToDatabase)
```

## Extraction Statistics

### Scope: internal/database/ (non-generated, non-test files)

| Category | Count |
|----------|-------|
| **Magic Numbers** | 1 |
| **Platform Name Strings** | 8 occurrences |
| **Event Type Strings** | 2 unique types |
| **Metric Type Strings** | 4 unique types |
| **Error Messages** | 200+ unique messages |
| **Log Messages** | 3 unique messages |
| **JSON Literals** | 1 (empty inventory) |
| **PostgreSQL Error Codes** | 1 |

**Total Constants Extracted:** 220+

## Implementation Notes

### Why These Were Extracted

1. **Platform Names (8 occurrences across 2 files)**
   - Value appears in switch statements and database queries
   - Critical for user platform linking consistency
   - Changes would require updating multiple locations

2. **Error Messages (200+ occurrences)**
   - Repeated across multiple repository files
   - Standardizes error reporting
   - Enables easier error message updates and localization

3. **Engagement Metrics**
   - Used in switch statements and default weight mapping
   - Central to progression system scoring
   - Type-safety for metric types

4. **Event Types**
   - Published to event bus for cache invalidation
   - Version tracking for event schema evolution

5. **Default Weights**
   - Fallback values for engagement scoring
   - Business logic parameters that may need tuning

6. **Empty Inventory JSON**
   - Ensures consistent initial state
   - Used during user registration

7. **PostgreSQL Error Code**
   - Detects duplicate key violations
   - Used for idempotency checks

### What Was NOT Extracted

- Generated SQLC code (`internal/database/generated/`)
- Test files (`*_test.go`)
- SQL query strings (managed by SQLC)
- JSONB field names (domain-level concern)
- Single-use literals with obvious context

## Next Steps for Full Refactoring

This scan focuses on **local cleanup** of the `internal/database/` module. To complete the refactoring:

1. **Replace Hard-Coded Values in Implementation Files**
   - Update all postgres/*.go files to use new constants
   - This is a mechanical find-replace operation
   - Estimated: 200+ replacements across 15 files

2. **Verify Build Success**
   - Run `make build` to ensure no syntax errors
   - Run `make test` to ensure behavior unchanged

3. **Project-Wide Constant Consolidation** (separate phase)
   - Check if platform names exist in `internal/domain/`
   - Check if error messages should move to error package
   - Determine if engagement constants should be in progression package

## File Locations Reference

```
internal/database/
├── constants.go                           # NEW - Top-level database constants
├── database.go                            # MODIFIED - Uses new constants
└── postgres/
    ├── constants.go                       # NEW - PostgreSQL repository constants
    ├── user.go                            # PENDING - 8 platform name replacements
    ├── user_linking.go                    # PENDING - 6 platform name replacements
    ├── progression.go                     # PENDING - 40+ error msg replacements
    ├── gamble.go                          # PENDING - Error code replacement
    ├── job.go                             # PENDING - Log message replacement
    ├── stats.go                           # PENDING - Error message replacements
    ├── crafting.go                        # PENDING - Error message replacements
    ├── economy.go                         # PENDING - Error message replacements
    ├── item.go                            # PENDING - Error message replacements
    ├── linking.go                         # PENDING - Error message replacements
    ├── eventlog.go                        # PENDING - Minimal changes
    ├── utils.go                           # PENDING - Error message replacements
    ├── progression_sessions.go            # PENDING - Error message replacements
    ├── progression_junction.go            # PENDING - Error message replacements
    ├── progression_prerequisites.go       # PENDING - Error message replacements
    └── progression_leaderboard.go         # PENDING - Error message replacements
```

## Verification Commands

```bash
# Check that constants file compiles
go build internal/database/constants.go

# Check that postgres constants file compiles
go build internal/database/postgres/constants.go

# Run database tests
go test ./internal/database/...

# Run full test suite
make test
```

## Impact Assessment

**Benefits:**
- Centralized constant management for 220+ values
- Easier maintenance of error messages
- Type-safe platform name references
- Consistent engagement metric types
- Clear documentation of magic numbers
- Foundation for future localization

**Risk Level:** Low
- Constants file is pure data declarations
- No behavioral changes in this phase
- Generated code unaffected
- Tests remain unchanged

**Effort for Full Implementation:**
- Constant extraction: ✅ Complete
- database.go updates: ✅ Complete
- postgres/*.go updates: ⏳ Pending (200+ replacements)
- Testing: ⏳ Pending
- Total estimated time: 2-3 hours for full replacement

---

**Last Updated:** 2026-01-15
**Scope:** internal/database module (non-generated code only)
**Limitation:** This scan focuses on local cleanup. Project-wide constant consolidation is a separate analysis phase.
