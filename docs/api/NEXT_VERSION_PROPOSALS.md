# Next API Version Proposals (v2)

This document outlines proposed changes for the next major API version release to improve consistency, remove redundancy, and clean up deprecated code.

## 1. Deprecated Code Removal

The following functions and methods are marked as deprecated and should be removed in the next version:

### Internal Handlers
- **File:** `internal/handler/crafting_request.go`
- **Function:** `trackCraftingEngagement`
- **Reason:** Redundant wrapper around `TrackEngagement`. Use `event_helpers.TrackEngagement` directly.

### Discord Bot
- **File:** `internal/discord/bot.go`
- **Method:** `SendDailyCommitReport`
- **Reason:** Replaced by `SendDailyPatchNotesReport`. The old method logs a warning but does nothing useful.

### Database
- **File:** `internal/database/postgres/progression.go`
- **Method:** `GetChildNodes`
- **Reason:** Deprecated in favor of junction table queries for prerequisites.

### Progression Service
- **File:** `internal/progression/voting_sessions.go`
- **Method:** `AdminForceEndVoting`
- **Reason:** Redundant wrapper around `EndVoting`. The naming is also confusing (see below).

## 2. API Consolidation & RESTful Improvements

### Inventory Management
- **Merge Endpoints:** `GET /user/inventory` and `GET /user/inventory-by-username`.
    - **Proposal:** Create a single `GET /user/inventory` endpoint that accepts either `platform_id` (preferred) or `username`.
    - **Current State:** Two separate endpoints exist with nearly identical functionality.
- **Standardize Item Operations:**
    - **Current:** `POST /user/item/add`, `POST /user/item/remove` (username only), `POST /user/item/give` (full details).
    - **Proposal:** Use standard RESTful resource paths:
        - `POST /user/items` (Add item)
        - `DELETE /user/items/{itemName}` (Remove item)
        - `POST /user/items/transfer` (Give item)

### User Search
- **Method Change:** `POST /user/search` -> `GET /user/search`
    - **Reason:** Search is typically a read-only operation and should use GET unless it triggers state changes (e.g., spending energy). If it strictly retrieves data, GET is more appropriate.

## 3. Admin Route Standardization

### Unified Namespace
- **Current State:** Admin routes are split between `/api/v1/admin/...` and domain-specific paths like `/api/v1/progression/admin/...`.
- **Proposal:** Consolidate all administrative actions under the `/api/v1/admin/` namespace to simplify access control and documentation.
    - Move `/api/v1/progression/admin/*` to `/api/v1/admin/progression/*`.

### Progression Voting Terminology
The current naming for voting session management is confusing:
- `POST .../end-voting` currently **freezes** the vote (pauses it).
- `POST .../force-end-voting` currently **ends** the vote (picks a winner).

**Proposal:**
- Rename `end-voting` to `freeze-voting` or `pause-voting`.
- Rename `force-end-voting` to `conclude-voting` or `finish-voting`.
- Rename `start-voting` to `resume-voting` (if resuming) or keep `start-voting` for new sessions.

## 4. General Improvements

### Naming Consistency
- **Issue:** Inconsistent handler naming (e.g., `HandleAddItemByUsername` vs `HandleGiveItem`).
- **Proposal:** Adopt a consistent naming convention like `Handle<Action><Resource>` (e.g., `HandleAddItem`, `HandleTransferItem`).

### ID vs Username
- **Issue:** Some endpoints strictly require IDs, others strictly require Usernames.
- **Proposal:** Where possible, allow endpoints to accept either (resolving Usernames to IDs internally) or enforce IDs strictly for v2 to avoid ambiguity, relying on a lookup service for username-to-ID resolution.
