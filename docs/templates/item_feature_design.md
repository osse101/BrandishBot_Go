# Item Feature Design Template: [Item Name]

## 1. Overview

Briefly describe the item's role in the gameplay loop and its thematic purpose.

- **Internal Name:** `item_internal_name`
- **Public Name:** `public_name`
- **Display Name:** `Default Display Name`
- **Tier:** [0-4]
- **Tags:** `[consumable, material, tradeable, upgradeable, disassembleable, sellable, buyable]`

---

## 2. Utility & Mechanics

Detailed technical specification of the item's primary behavior.

### Behavior Logic

- [ ] **Active Use**: Triggered via `!use [item] [args]`.
- [ ] **Passive Effect**: Constant buff/debuff while in inventory.
- [ ] **Event-Driven**: Triggered by external hooks (e.g., chat messages, reactions, level ups).

### Execution Flow

1. **Pre-conditions**: (e.g., Target must be active, User must not be timed out).
2. **Action**: (e.g., Apply status, grant item, trigger event).
3. **Feedback**: (e.g., Channel message, DM to user, visual "juice").

### API / Handler Input Args

What keys should be present in the `args map[string]interface{}`?

- `target_username`: (string) Targeted user.
- `quantity`: (int) Number of items being used.
- `platform`: (string) Origin platform (twitch/discord).

---

## 3. Technical Infrastructure

### Persistence & State

- **Database Changes**: Does this item require a new table or new columns in `users`/`inventory`?
- **TTL/Cleanup**: If the item creates a lasting state (like a trap or a buff), how and when is it cleaned up?

### Concurrency & Performance

- **Atomicity**: Does the logic require a database transaction (`tx`) to prevent dupes or race conditions?
- **Locking**: Are there shared resources that need mutex protection?
- **Complexity**: Is the effect expensive? Should it run in a goroutine?

### Event Hooks

Where does this item interface with the rest of the bot?

- `internal/user/item_handlers.go`: Standard `!use` logic.
- `internal/user/service.go`: Hooks into message/user lifecycle.
- `internal/handler/`: Direct command handler hooks.

---

## 4. Interactions

### Shine Interaction

How does quality (COMMON to LEGENDARY) scale the effect?

- **Potency**: (e.g., More XP, more damage).
- **Duration**: (e.g., Longer buffs, shorter timeouts).
- **Visuals**: Special emojis or message prefixes for high-tier shine.

### Upgrade / Crafting Interaction

- **Upgradeable**: Yes/No (Define materials and target item).
- **Disassembleable**: Yes/No (Define materials returned).
- **Masterwork/Perfect Salvage**: Any item-specific critical success effects?

---

## 5. Systems Integration

- **Progression**: Which node unlocks this item?
- **Job System**: Does it grant XP? Which job is the "specialist" for this item?
- **Stats & Tracking**: What metrics should be recorded (e.g., `EventItemUsed`, `StatTotalDamageDealt`)?
- **Cooldowns**: Standard global cooldown or item-specific lockout?

---

## 6. Scope of Work

### Implementation Checklist

- [ ] Add definition to `configs/items/items.json`.
- [ ] Create database migration in `migrations/`.
- [ ] Register handler in `internal/user/handler_registry.go`.
- [ ] Implement logic in `internal/user/item_handlers.go`.
- [ ] (Optional) Create new domain constants in `internal/domain/`.
- [ ] Add unit tests for success and failure cases.
- [ ] Ensure `naming/constants.go` is updated for display resolution.
