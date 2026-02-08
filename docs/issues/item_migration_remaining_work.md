# Item Migration - Remaining Work

## Completed (Jan 2026)

- Added 14 new items to `configs/items/items.json`
- Added tier3 lootbox (diamondbox) with loot table
- Added 9 crafting recipes and 9 disassemble recipes
- Added 14 progression nodes for new items
- Implemented weapon handler with configurable timeouts
- Implemented revive handler (reduces timeout)
- Implemented shield handler (placeholder)
- Implemented rarecandy handler (awards job XP)

---

## Remaining Work

### 1. Shield System - Persistent Storage

**Priority:** Medium
**Effort:** Medium

The shield handler currently has a placeholder implementation. Full implementation requires:

- [ ] Add `user_shields` table or add `shield_count` column to users table
- [ ] Store shield count persistently per user
- [ ] Integrate shield check into weapon handler (consume shield before applying timeout)
- [ ] Add migration for database schema

**Files to modify:**
- `internal/user/service.go` - `ApplyShield()` method
- `internal/user/item_handlers.go` - `handleWeapon()` to check for shields
- `migrations/` - new migration file

### 2. ~~Explosive/Trap System~~ [COMPLETED - 2026-01-30]

**Status**: Implemented.
- `user_traps` table exists.
- `handleTrap` is fully implemented in `internal/user/item_handlers.go`.
- `triggerTrap` is implemented in `internal/user/service.go` and hooked into message processing.

### 3. Handler Config Parsing

**Priority:** Low
**Effort:** Low

Currently, `handler_config` in items.json is not parsed. Timeout values are hard-coded in maps within `item_handlers.go`. To make this data-driven:

- [ ] Add `HandlerConfig map[string]interface{}` to `item.Def` struct
- [ ] Add `HandlerConfig` column to items table (JSONB)
- [ ] Update item loader to parse and sync handler_config
- [ ] Modify handlers to read config from item instead of static maps

### 4. Deferred Items (Stream Integration)

**Priority:** Future
**Effort:** High

These items require stream platform integration:

| Item | Description | Requirement |
|------|-------------|-------------|
| `sabotage_input` | Input delay on stream | OBS/Stream integration |
| `sabotage_swap` | Control swap on stream | OBS/Stream integration |
| `sabotage_input60` | Extended input delay | OBS/Stream integration |
| `stream_poll` | Trigger stream poll | Twitch API |
| `stream_fx` | Trigger stream effects | OBS WebSocket |

### 5. Pre-existing Test Failures

**Priority:** Medium
**Effort:** Low

Handler tests `TestHandleSellItem` and `TestHandleRemoveItem` have pre-existing failures unrelated to item migration. The tests expect old response format without `message` field.

**Files to fix:**
- `internal/handler/inventory_test.go`

---

## Testing Checklist

After implementing remaining items:

- [ ] Run `make generate` if SQLC changes needed
- [ ] Run `make mocks` if interfaces changed
- [ ] Run `make test` to verify no regressions
- [ ] Test progression node sync with `make run`
- [ ] Verify new items appear in `/api/v1/prices` endpoint
- [ ] Test upgrade/disassemble recipes work
- [ ] Test each new handler in isolation
