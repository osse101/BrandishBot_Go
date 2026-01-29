# Gamble Session ID Issues - Technical Analysis

**Date**: 2026-01-28
**Status**: Identified, Awaiting Fix
**Severity**: CRITICAL
**Affects**: Gamble system functionality across all clients

---

## Executive Summary

Multiple critical bugs in the gamble system related to ID handling prevent users from joining gambles and cause data corruption. The primary blocker is that Discord clients cannot retrieve the gamble ID after starting a gamble, making it impossible for users to join.

---

## Issue 1: Gamble ID Not Returned to Discord Client
**Severity**: CRITICAL - Users cannot join gambles they start

### Root Cause
The Discord client's `StartGamble()` method extracts the gamble ID from the API response but only returns the message string, discarding the ID.

### Current Flow
```
1. Handler (internal/handler/gamble.go:51)
   Returns: domain.Gamble{ID: uuid.UUID, InitiatorID: "...", ...}

2. Discord Client (internal/discord/client.go:276-310)
   Parses response into:
   var gambleResp struct {
       Message  string `json:"message"`
       GambleID string `json:"gamble_id"`
   }

   But returns: gambleResp.Message, nil  ❌

3. Discord Command (internal/discord/cmd_gamble.go:59)
   Receives: msg string (no gamble ID)
   Cannot pass ID to JoinGamble command
```

### Impact
- Users start a gamble via `/gamble` command
- Command shows "(no message)" or generic message
- No gamble ID is displayed or stored
- Users have NO way to join the gamble (cannot call `/join <id>`)
- Gamble system is essentially broken for Discord users

### Evidence
- **Line 309** of `internal/discord/client.go`: `return gambleResp.Message, nil` - only returns message
- **Line 301-304**: GambleID is parsed from JSON but never used
- **Line 59** of `internal/discord/cmd_gamble.go`: Command receives only `msg` string
- **docs/issues/todo.txt:7**: "GambleStart -- Response is '(no message)'. Should contain gamble ID."

### Proposed Fix
```go
// internal/discord/client.go:309
// BEFORE:
return gambleResp.Message, nil

// AFTER:
return gambleResp.GambleID, nil  // Return ID instead of message
```

Update Discord command to display the gamble ID and allow users to copy it for joining.

---

## Issue 2: Response Format Mismatch
**Severity**: HIGH - Related to Issue 1

### Root Cause
Handler returns full `domain.Gamble` object, but Discord client expects a wrapper struct with specific fields.

### Current State
**Handler** (`internal/handler/gamble.go:51`):
```go
respondJSON(w, http.StatusCreated, gamble)
// Returns: {"id":"uuid","initiator_id":"...","state":"joining",...}
```

**Discord Client Expectation** (`internal/discord/client.go:301-304`):
```go
var gambleResp struct {
    Message  string `json:"message"`
    GambleID string `json:"gamble_id"`
}
```

### Mismatch
- Handler doesn't return `message` or `gamble_id` fields
- Client tries to extract non-existent fields
- Both fields end up as empty strings

### Proposed Fix
Create response wrapper in handler:
```go
type StartGambleResponse struct {
    Message  string `json:"message"`
    GambleID string `json:"gamble_id"`
}

response := StartGambleResponse{
    Message:  fmt.Sprintf("Gamble started! Join within %s", time.Until(gamble.JoinDeadline).Round(time.Second)),
    GambleID: gamble.ID.String(),
}
respondJSON(w, http.StatusCreated, response)
```

---

## Issue 3: InitiatorID UUID Parsing Bug
**Severity**: HIGH - Data corruption, silent failures

### Root Cause
`InitiatorID` is stored as `"platform:platformID"` string (e.g., "discord:123456"), but the database schema defines it as `uuid` type, and the repository tries to parse it as a UUID.

### Current Bug
**File**: `internal/database/postgres/gamble.go:39-41`
```go
func (r *GambleRepository) CreateGamble(ctx context.Context, gamble *domain.Gamble) error {
    initiatorID, err := uuid.Parse(gamble.InitiatorID)  // ❌ Fails!
    if err != nil {
        return fmt.Errorf("invalid initiator id: %w", err)
    }
    // ...
}
```

### The Problem
- **Domain**: `domain.Gamble.InitiatorID` is type `string` (line 28 of `internal/domain/gamble.go`)
- **Service**: Creates gamble with `InitiatorID: user.ID` where `user.ID` is "discord:123456" format
- **Repository**: Tries to `uuid.Parse("discord:123456")` → **FAILS**
- **Schema**: `gambles.initiator_id` column is type `uuid` in database

### User ID Format in System
User IDs throughout the system are created as:
```go
// internal/user/service.go
user.ID = fmt.Sprintf("%s:%s", platform, platformID)
// Results in: "discord:123456", "twitch:789", etc.
```

This is NOT a UUID format.

### Impact
- Gamble creation likely fails silently or throws errors
- If it somehow succeeds, data corruption occurs (storing invalid UUID)
- Retrieving gambles converts back: `InitiatorID: g.InitiatorID.String()` (line 76) - may produce garbage

### Evidence
- Database migrations show `initiator_id uuid` type
- Service passes string format user IDs
- Repository attempts UUID parsing
- No UUID generation happens anywhere in the flow

### Proposed Fix
**Option A**: Change schema to `text` (RECOMMENDED)
```sql
ALTER TABLE gambles ALTER COLUMN initiator_id TYPE text;
```
Remove `uuid.Parse()` call from repository.

**Option B**: Create UUID mapping (Complex, not recommended)
- Maintain separate user_id → uuid mapping table
- Look up UUID before insert

---

## Issue 4: ParticipantUserID UUID Parsing Bug
**Severity**: HIGH - Same as Issue 3

### Root Cause
Identical to Issue 3, but for `gamble_participants.user_id` column.

### Current Bug
**File**: `internal/database/postgres/gamble.go:106-110`
```go
func (r *GambleRepository) JoinGamble(ctx context.Context, participant *domain.Participant) error {
    userID, err := uuid.Parse(participant.UserID)  // ❌ Same problem
    if err != nil {
        return fmt.Errorf("invalid user id: %w", err)
    }
    // ...
}
```

### Schema
`gamble_participants.user_id` is defined as `uuid` in schema (line 92 of migrations)

### Impact
- Users cannot join gambles (JoinGamble fails)
- Data corruption if somehow inserted
- Same platform:id format issue

### Proposed Fix
Same as Issue 3 - change schema to `text`:
```sql
ALTER TABLE gamble_participants ALTER COLUMN user_id TYPE text;
```

---

## Issue 5: ID Type Inconsistencies (General)
**Severity**: MEDIUM - Design inconsistency

### Observations
The system uses UUIDs and string IDs inconsistently:

**UUID Usage** (Correct):
- Gamble ID: `gamble.ID uuid.UUID` - Generated by service, stored as uuid, works correctly
- Database: Gamble IDs stored as `uuid` type in schema

**String Usage** (Correct):
- User IDs: `"platform:platformID"` format throughout system
- Platform-specific identifiers

**Mismatch** (Incorrect):
- Gamble initiator/participant IDs stored as `uuid` but should be `text`
- Repository tries to parse strings as UUIDs

### Format Conversion Flow
```
Service Layer:    uuid.UUID (gamble ID) ✓    string (user IDs) ✓
   ↓
Repository Layer: uuid.Parse(string) ❌ Tries to convert user ID to UUID
   ↓
Database Layer:   uuid column ❌ Wrong type for platform:id strings
```

### Recommendation
Maintain clear separation:
- **Entity IDs** (gamble, item, etc.): Use `uuid.UUID`
- **User IDs** (cross-platform): Use `string` with "platform:id" format
- Never try to convert user IDs to UUIDs

---

## Issue 6: URL Parameter vs Body Parameter
**Severity**: LOW - Design inconsistency

### Current Implementation
Join endpoint uses query string for gamble ID:
```go
// Handler: internal/handler/gamble.go:61
gambleIDStr, ok := GetQueryParam(r, w, "id")

// Discord client: internal/discord/client.go:321
fmt.Sprintf("/api/v1/gamble/join?id=%s", gambleID)

// C# client: client/csharp/BrandishBotClient.cs:479
"/api/v1/gamble/join?id=" + gambleId
```

### Issue
Primary identifier conventionally goes in:
1. URL path: `/api/v1/gamble/{id}/join`
2. Request body: `{"gamble_id": "..."}`

Not query string: `/api/v1/gamble/join?id=...`

### Impact
- Confusing API design
- Contributes to ID handling confusion
- Not critical, but inconsistent with REST patterns

### Recommendation
Consider refactoring to:
```
POST /api/v1/gamble/{id}/join
```
Or keep query string but document clearly.

---

## Summary Table

| Issue | Severity | Blocker? | Files Affected |
|-------|----------|----------|----------------|
| **Gamble ID not returned** | CRITICAL | YES | `internal/discord/client.go:309`<br>`internal/discord/cmd_gamble.go:59` |
| **Response format mismatch** | HIGH | YES | `internal/handler/gamble.go:51`<br>`internal/discord/client.go:301-304` |
| **InitiatorID UUID parsing** | HIGH | YES | `internal/database/postgres/gamble.go:39`<br>Schema migrations |
| **ParticipantUserID UUID parsing** | HIGH | YES | `internal/database/postgres/gamble.go:106`<br>Schema migrations |
| **ID type inconsistencies** | MEDIUM | NO | System-wide design |
| **Query string ID parameter** | LOW | NO | Handler + both clients |

---

## Required Schema Migration

```sql
-- migrations/XXXX_fix_gamble_user_id_types.sql

BEGIN;

-- Change initiator_id from uuid to text
ALTER TABLE gambles ALTER COLUMN initiator_id TYPE text USING initiator_id::text;

-- Change participant user_id from uuid to text
ALTER TABLE gamble_participants ALTER COLUMN user_id TYPE text USING user_id::text;

-- Add comment explaining the format
COMMENT ON COLUMN gambles.initiator_id IS 'User ID in format platform:platformID (e.g., discord:123456)';
COMMENT ON COLUMN gamble_participants.user_id IS 'User ID in format platform:platformID (e.g., discord:123456)';

COMMIT;
```

**Data Migration Note**: If any gambles exist in the database with UUID-formatted initiator_ids, they will need manual cleanup or conversion.

---

## Testing Plan

After fixes:

1. **Start Gamble**
   - Discord: `/gamble` should return gamble ID
   - C#: StartGamble should return gamble ID
   - No UUID parsing errors in logs

2. **Join Gamble**
   - Discord: `/join <gamble_id>` should work
   - C#: JoinGamble(gambleID) should work
   - No UUID parsing errors in logs

3. **Database Validation**
   - Check `gambles` table: initiator_id should be "platform:id" format
   - Check `gamble_participants` table: user_id should be "platform:id" format
   - No NULL or malformed IDs

4. **Full Flow**
   - User A starts gamble → receives ID
   - User B joins with ID → succeeds
   - Gamble executes → both participants processed correctly

---

## References

- **Todo**: `docs/issues/todo.txt:7` - "GambleStart -- Response is '(no message)'"
- **Exploration**: Agent a6d8502 (Gamble system exploration, 2026-01-28)
- **Related**: Race condition in gamble execution (separate issue)
