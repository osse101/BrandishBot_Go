# Item Feature Design: Explosive Trap

## 1. Overview

The **Explosive Trap** is a tactical consumable that allows a player to booby-trap another user. Unlike direct weapons, the trap lies in wait and triggers only when the target user sends a message in chat.

- **Internal Name:** `explosive_trap`
- **Public Name:** `trap`
- **Display Name:** `A Targeted Explosive`
- **Tier:** 2
- **Tags:** `[consumable, tradeable, upgradeable, disassembleable, sellable, buyable]`

## Status
**Implemented** (2026-01-30). This feature is live in the codebase.

---

## 2. Utility & Mechanics

### Usage Flow

1. **Deployment**: User A runs `!use trap <UserB>`.
2. **Persistence**: A trap record is created in the database associating User A (setter) and User B (target).
3. **Activation**: The trap remains dormant until User B sends any message in chat.
4. **Trigger**:
   - User B is timed out for **60 seconds** (scaled by quality).
   - A public message is sent to the channel: _"BOOM! [UserB] stepped on [UserA]'s trap!"_
   - The trap record is marked as completed/removed.

### The "Self-Trap" Logic

To prevent spam and encourage strategic timing, only one trap can be active on a user at a time.

- If User C tries to trap User B while User B **already has an active trap**:
  - The existing trap on User B **immediately triggers** on User C (the person currently using the item).
  - User C is timed out.
  - User C's item is still consumed, and a _new_ trap is then placed on User B (with User C as the setter).
  - No cooldown is applied to User C for this "accidental" trigger.

---

## 3. Technical Implementation

### Persistence & Data Model

- **New Table**: `user_traps`
  - `id`: Primary Key
  - `setter_id`: User who placed the trap.
  - `target_id`: User who is trapped.
  - `quality_level`: Determines timeout duration.
  - `placed_at`: Timestamp for stale trap cleanup.
  - `triggered_at`: Nullable, set when fired.
- **Cleanup**: Considerations for a monthly maintenance task to remove very stale, untriggered traps.

### Message Hook

- Implementation targets `internal/user/service.go:HandleIncomingMessage`.
- Every incoming message checks the `user_traps` table for an active record where `target_id == messenger_id`.
- Cache for performance
- If found, the `trap.Trigger()` logic is executed.

### Concurrency & Atomicity

- **Atomic Transactions**: The check for an existing trap and the placement of a new one must be performed within a single database transaction to prevent race conditions (e.g., two users trapping the same target simultaneously).
- Use the Check then Lock and Recheck pattern for performance, when checking for existing traps.

---

## 4. Interactions

### Quality Interaction

Quality level modifies the severity of the explosion (timeout duration).

- **Base Duration**: 60 seconds (Common).
- **Scaling**: +/- 10 seconds per quality level.
- **Example**:
  - _Junk_: 40s
  - _Common_: 60s
  - _Legendary_: 100s

### Upgrade / Crafting Interaction

- **Is Upgradeable**: Yes (`5 traps + 5 Scrap` -> `1 Bomb`).
- **Is Disassembleable**: Yes (`1 Trap` -> `1 Scrap + 1 Mine`).
- **Perfect Salvage**: Standard multipliers apply.

---

## 5. Systems Integration

- **Progression**: Gated behind `feature_farming` (or a dedicated `demolitions` node).
- **Stats/Events**:
  - `EventTrapPlaced`
  - `EventTrapTriggered`
  - `StatMostTrappedUser`
- **Cooldown**:
  - **Standard**: 10-minute cooldown on the target user.
  - **Constraint**: A trap cannot be placed on a target who is currently on their "Post-Trap Cooldown" window (simplified logic).

---

## 6. Scope of Work

### Repositories & Files

- `internal/trap/`: New package for trap registry and trigger logic.
- `internal/user/item_handlers.go`: Implementation of `handleTrap`.
- `internal/user/service.go`: Hook into `HandleIncomingMessage`.
- `configs/items/items.json`: Item definition and handler config.
- `migrations/`: Migration for `user_traps` table.
- `internal/naming/constants.go`: Public/Internal name mappings.
