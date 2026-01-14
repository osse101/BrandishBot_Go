# BrandishBot Go - Claude Code Context

Go game engine API for live chatroom gaming. HTTP API (port 8080) + Discord bot (port 8082).

## Quick Commands

```bash
make run              # Start API server
make discord-run      # Start Discord bot
make test             # Run tests
make lint             # Run linters
make generate         # Regenerate SQLC/mocks
make mocks            # Regenerate mocks
make migrate-up       # Apply migrations
make docker-up        # Start all services
```

## Architecture Pattern

```
Handler (internal/handler/) → Service (internal/*/service.go) → Repository (internal/repository/) → Postgres (internal/database/postgres/)
```

All services follow this pattern. Discord commands mirror this via API client calls.

---

## Feature Quick Reference

### User/Inventory System
| Layer | Location |
|-------|----------|
| Service | `internal/user/service.go:32-66` |
| Repository | `internal/repository/user.go:11-42` |
| Postgres | `internal/database/postgres/user.go` |
| Handler | `internal/handler/user.go:33-80` |
| Discord | `internal/discord/cmd_inventory.go`, `cmd_profile.go` |
| Domain | `internal/domain/user.go`, `inventory.go`, `item.go` |

**Key methods:** `RegisterUser`, `AddItem`, `RemoveItem`, `UseItem`, `GetInventory`

### Economy System
| Layer | Location |
|-------|----------|
| Service | `internal/economy/service.go:16-48` |
| Repository | `internal/repository/economy.go` |
| Postgres | `internal/database/postgres/economy.go` |
| Handler | `internal/handler/prices.go` |
| Discord | `internal/discord/cmd_economy.go` |
| Pricing | `internal/economy/buyprices.go` |

**Key methods:** `GetSellablePrices`, `GetBuyablePrices`, `SellItem`, `BuyItem`

### Crafting System
| Layer | Location |
|-------|----------|
| Service | `internal/crafting/service.go:18-100+` |
| Repository | `internal/repository/crafting.go` |
| Postgres | `internal/database/postgres/crafting.go` |
| Handler | `internal/handler/upgrade.go`, `disassemble.go` |
| Discord | `internal/discord/cmd_crafting.go` |
| Loader | `internal/crafting/recipe_loader.go` |

**Key methods:** `UpgradeItem`, `DisassembleItem`, `GetRecipe`, `GetUnlockedRecipes`
**Special mechanics:** Masterwork (10% chance, 2x), Perfect Salvage (10% chance, 1.5x)

### Progression System
| Layer | Location |
|-------|----------|
| Service | `internal/progression/service.go:16-120` |
| Repository | `internal/repository/progression.go:11-70` |
| Postgres | `internal/database/postgres/progression*.go` |
| Handler | `internal/handler/progression.go` |
| Discord | `internal/discord/cmd_progression.go` |
| Loader | `internal/progression/tree_loader.go` |
| Events | `internal/progression/handler.go`, `notifier.go` |
| Domain | `internal/domain/progression.go:1-189` |

**Key methods:** `VoteForUnlock`, `CheckAndUnlockNode`, `GetModifiedValue`, `RecordEngagement`
**Caching:** Unlock cache (event-invalidated) + modifier cache (30-min TTL)

### Gamble/Lootbox System
| Layer | Location |
|-------|----------|
| Service | `internal/gamble/service.go:23-68` |
| Lootbox | `internal/lootbox/service.go:55-100+` |
| Repository | `internal/repository/gamble.go` |
| Postgres | `internal/database/postgres/gamble.go` |
| Handler | `internal/handler/gamble.go` |
| Discord | `internal/discord/cmd_gamble.go` |
| Worker | `internal/worker/gamble_worker.go` |
| Domain | `internal/domain/gamble.go:1-67` |

**Key methods:** `StartGamble`, `JoinGamble`, `ExecuteGamble`, `OpenLootbox`
**Shine levels:** COMMON(1.0x), UNCOMMON(1.1x), RARE(1.25x), EPIC(1.5x), LEGENDARY(2.0x)
**Near-miss threshold:** 95%

### Job/XP System
| Layer | Location |
|-------|----------|
| Service | `internal/job/service.go:26-62` |
| Repository | `internal/repository/job.go` |
| Postgres | `internal/database/postgres/job.go` |
| Handler | `internal/handler/job.go` |
| Discord | `internal/discord/cmd_jobs.go` |

**Key methods:** `GetAllJobs`, `GetUserJobs`, `AwardXP`, `GetJobLevel`, `GetJobBonus`

### Stats/Leaderboard System
| Layer | Location |
|-------|----------|
| Service | `internal/stats/service.go:14-32` |
| Repository | `internal/repository/stats.go` |
| Postgres | `internal/database/postgres/stats.go` |
| Handler | `internal/handler/stats.go` |
| Discord | `internal/discord/cmd_stats.go` |

**Key methods:** `RecordUserEvent`, `GetUserStats`, `GetUserCurrentStreak`, `GetLeaderboard`

### Event System
| Layer | Location |
|-------|----------|
| Core Bus | `internal/event/event.go:99-151` |
| Resilient Publisher | `internal/event/resilient_publisher.go` |
| Event Log | `internal/eventlog/service.go` |
| Metrics | `internal/metrics/collector.go` |

**Key types:** `ProgressionCycleCompleted`, `EventTypeEngagement`, `EventGambleStarted`
**Retry:** Exponential backoff (2s → 4s → 8s → 16s → 32s)

### Discord Bot
| Layer | Location |
|-------|----------|
| Bot Core | `internal/discord/bot.go:19-100+` |
| Registry | `internal/discord/commands.go:13-42` |
| API Client | `internal/discord/client.go` |
| Autocomplete | `internal/discord/autocomplete.go` |
| Main | `cmd/discord/main.go` |

**Command files:** `cmd_*.go` in `internal/discord/`

### Linking System
| Layer | Location |
|-------|----------|
| Service | `internal/linking/service.go` |
| Repository | `internal/repository/linking.go` |
| Discord | `internal/discord/cmd_link.go` |

---

## Database

**SQLC Config:** `sqlc.yaml`
**Generated Code:** `internal/database/generated/`
**Migrations:** `migrations/` (Goose)
**Connection:** `internal/database/database.go`

### Key Tables
- `users`, `user_platform_links` - User accounts
- `items`, `inventories` - Items and user inventories (JSONB)
- `crafting_recipes`, `disassemble_recipes` - Recipes
- `progression_nodes`, `progression_unlocks`, `progression_voting_sessions` - Progression tree
- `gambles`, `gamble_participants` - Gamble sessions
- `jobs`, `user_jobs` - Job/XP system
- `stats_events`, `events` - Event logging

---

## Configuration Files

| Config | Location | Loader |
|--------|----------|--------|
| Items | `configs/items.json` | `internal/item/loader.go` |
| Crafting Recipes | `configs/recipes/crafting.json` | `internal/crafting/recipe_loader.go` |
| Disassemble Recipes | `configs/recipes/disassemble.json` | `internal/crafting/recipe_loader.go` |
| Progression Tree | `configs/progression/tree.json` | `internal/progression/tree_loader.go` |
| Loot Tables | `configs/loot_tables.json` | `internal/lootbox/service.go` |
| Item Aliases | `configs/item_aliases.json` | `internal/naming/resolver.go` |
| Item Themes | `configs/item_themes.json` | `internal/naming/resolver.go` |

---

## Common Task Patterns

### Adding a New API Endpoint
1. Define request/response in `internal/domain/`
2. Add service method in `internal/*/service.go`
3. Add repository method in `internal/repository/*.go`
4. Implement repository in `internal/database/postgres/*.go`
5. Add SQLC query in `internal/database/queries/*.sql` and run `make generate`
6. Add handler in `internal/handler/*.go`
7. Register route in `internal/server/routes.go`

### Adding a New Discord Command
1. Create `internal/discord/cmd_*.go` with `*Command()` function
2. Return `(*discordgo.ApplicationCommand, CommandHandler)`
3. Add API client method in `internal/discord/client.go` if needed
4. Register in `cmd/discord/main.go`

### Adding a New Event
1. Define event type constant in `internal/domain/events.go`
2. Create typed payload struct in `internal/event/event.go`
3. Publish via `eventBus.Publish()` or `resilientPublisher.Publish()`
4. Subscribe in relevant service's handler

### Adding a New Database Migration
1. Create file in `migrations/` with format `XXXX_description.sql`
2. Run `make migrate-up`

### Modifying SQLC Queries
1. Edit `.sql` files in `internal/database/queries/`
2. Run `make generate`
3. Update repository implementations

---

## Service Dependencies (from cmd/app/main.go)

```
user.Service      ← stats, job, lootbox, naming, cooldown
crafting.Service  ← job, stats, naming
economy.Service   ← job
gamble.Service    ← lootbox, stats, job, progression
progression.Service ← event bus
job.Service       ← progression, stats, event bus
```

---

## Entry Points

| Purpose | File |
|---------|------|
| API Server | `cmd/app/main.go` |
| Discord Bot | `cmd/discord/main.go` |
| DB Setup | `cmd/setup/main.go` |
| Debug Tools | `cmd/debug/main.go` |
| DB Reset | `cmd/reset/main.go` |

---

## Testing

- Unit tests: `*_test.go` alongside source files
- Mocks: Generated by Mockery (`.mockery.yaml`)
- Integration tests: Use testcontainers for Postgres
- Run: `make test`

---

## API Endpoints Summary

**Health:** `GET /healthz`, `GET /readyz`, `GET /version`, `GET /metrics`

**User:** `/api/v1/user/register`, `/api/v1/user/inventory`, `/api/v1/user/item/*`

**Economy:** `/api/v1/prices`, `/api/v1/prices/buy`, `/api/v1/economy/buy`, `/api/v1/economy/sell`

**Crafting:** `/api/v1/user/item/upgrade`, `/api/v1/user/item/disassemble`, `/api/v1/crafting/recipes`

**Progression:** `/api/v1/progression/*` (tree, available, vote, status, engagement, admin)

**Gamble:** `/api/v1/gamble/start`, `/api/v1/gamble/join`, `/api/v1/gamble/get`

**Jobs:** `/api/v1/jobs`, `/api/v1/jobs/user`, `/api/v1/jobs/award-xp`, `/api/v1/jobs/bonus`

**Stats:** `/api/v1/stats/event`, `/api/v1/stats/user`, `/api/v1/stats/system`, `/api/v1/stats/leaderboard`

**Docs:** `/swagger/`

---

## Workflow Automation

### Primary Dev Cycle
```bash
make docker-down && make build && make test && make docker-up
```

### After Editing Repository Interfaces
```bash
make mocks      # Regenerate mocks
make test       # Verify tests pass
```

### After Editing SQLC Queries
```bash
make generate   # Regenerate SQLC code
make mocks      # If query changes affect repository interface
```

### Pre-Commit Checklist
- [ ] Run `make test` or specific test file
- [ ] Run `make lint` (encouraged, not enforced)
- [ ] Check client synchronization if API changed

---

## Client Synchronization (CRITICAL)

**Three clients must stay synchronized when API changes:**

| Client | Location | Language |
|--------|----------|----------|
| Go Discord | `internal/discord/client.go` | Go |
| C# Streamer.bot | `client/csharp/BrandishBotClient.cs` | C# |
| API Handlers | `internal/handler/*.go` | Go |

### Sync Checklist for New/Modified Endpoints
- [ ] Handler implemented in `internal/handler/`
- [ ] Route registered in `internal/server/routes.go`
- [ ] Discord client method in `internal/discord/client.go`
- [ ] C# client method in `client/csharp/BrandishBotClient.cs`
- [ ] `docs/CLIENT_WRAPPER_CHECKLIST.md` updated

### Reference Document
See `docs/CLIENT_WRAPPER_CHECKLIST.md` for full endpoint coverage matrix.

---

## Pattern Learning Prompts

When performing these tasks, Claude should ask to fill in the complete pattern:

### Adding a New API Endpoint
**Ask:** "I'll add this endpoint. Please confirm the full pattern:
1. Domain types needed?
2. Service interface method signature?
3. Repository interface method?
4. SQL query needed?
5. Handler request/response format?
6. Route path and method?
7. Discord client method needed?
8. C# client method needed?"

### Adding a New Discord Command
**Ask:** "I'll add this command. Please confirm:
1. Command name and description?
2. Options/parameters?
3. Autocomplete needed for which fields?
4. Which API endpoint does it call?
5. Response formatting?"

### Refactoring for Duplication
**Look for:** Repeated code blocks, similar handler patterns, duplicated validation logic
**Propose:** Extract to shared utility, create interface, or use generics

---

## Known Pain Points

### Hard-Coded Strings
Watch for and extract to constants:
- Item names → `internal/domain/item.go` constants
- Event types → `internal/domain/events.go`
- Job keys → `internal/domain/job.go`
- Platform names → `internal/domain/user.go` (PlatformTwitch, etc.)
- Error messages → Consider error package

### Common Locations for Hard-Coded Values
- `internal/handler/*.go` - response messages
- `internal/discord/cmd_*.go` - command responses, embeds
- `configs/*.json` - item names, job keys (these are source of truth)

---

## Design Decision Documentation

### Journal Files (READ THESE FOR CONTEXT)
| Journal | Purpose |
|---------|---------|
| `docs/architecture/journal.md` | Architecture decisions, concurrency patterns, scaling |
| `docs/development/journal.md` | Dev patterns, transaction handling, cooldowns |
| `docs/testing/journal.md` | Test strategies, mocking patterns |
| `docs/benchmarking/journal.md` | Performance insights |
| `docs/tools/journal.md` | Tooling decisions |

### Key Architecture Documents
| Document | Purpose |
|----------|---------|
| `docs/CLIENT_WRAPPER_CHECKLIST.md` | API endpoint coverage for all clients |
| `docs/architecture/EVENT_SYSTEM.md` | Event bus architecture |
| `docs/architecture/cooldown-service.md` | Check-then-lock pattern |
| `docs/development/FEATURE_DEVELOPMENT_GUIDE.md` | Feature development workflow |
| `docs/ARCHITECTURE.md` | Overall system architecture |

---

## Code Quality Reminders

### Transaction Pattern (from journal)
```go
tx, err := s.repo.BeginTx(ctx)
if err != nil { return err }
defer repository.SafeRollback(ctx, tx)

// Read with FOR UPDATE lock
data, err := tx.GetWithLock(ctx, id)
// Modify...
// Write back
tx.Commit(ctx)
```

### Error Handling
- Wrap errors with `fmt.Errorf("context: %w", err)`
- Log internal details, return generic message to client
- Never expose DB errors to API responses

### Background Tasks
- Always `wg.Add(1)` BEFORE spawning goroutine
- Implement `Shutdown(ctx)` for graceful shutdown
- Use `ResilientPublisher` for event publishing

---

## Refactoring & Code Quality

### Duplication Identification & Refactoring

When performing code refactoring to reduce duplication:

1. **Identify high-impact patterns** - Use grep to find patterns appearing 3+ times
2. **Calculate savings** - `pattern_lines × occurrences = total_lines_saved`
3. **Create helpers** - Extract to utility functions in appropriate module
4. **Document patterns** - Add helper comments with usage examples
5. **Measure impact** - Grep before/after to verify replacements

**Example Refactoring Metrics** (Discord module, Jan 2026):
- Pattern: Embed creation + sending (30 occurrences, 8 lines each)
- Solution: `sendEmbed()` + `createEmbed()` helpers
- Impact: 240 lines removed, 62% duplication reduction
- Effort: 2 hours for design, implementation, testing

**For comprehensive refactoring guidance**, see:
→ **`docs/DISCORD_REFACTORING_PATTERNS.md`** (Essential Reference)

This document covers:
- Pattern identification checklist
- 4-level refactoring difficulty guide (trivial to complex)
- Implementation workflow (analysis → design → implementation → validation)
- Common patterns in Discord commands (deferred response, embed send, user extraction)
- Decision matrix for when to refactor
- Metrics & measurement
- Lessons learned & best practices

**Key Refactoring Principles:**
- Replace N-line duplicate patterns with 1-line helper calls
- Use boolean/enum parameters for variations instead of multiple helpers
- Create helpers with clear names and comprehensive documentation
- Test build after each implementation phase
- Verify with grep before/after to measure impact

### Related Journal References
- `docs/development/journal.md` - Development patterns
- `docs/testing/journal.md` - Test strategies
- `docs/architecture/journal.md` - Architectural decisions

---

## Quick Grep Commands

```bash
# Find all hard-coded item names
grep -rn '"money"\|"lootbox"\|"junkbox"' internal/

# Find API endpoints
grep -rn 'r\.\(Get\|Post\|Put\|Delete\)' internal/server/

# Find Discord commands
grep -rn 'ApplicationCommand{' internal/discord/

# Find event types
grep -rn 'Type =' internal/event/

# Find repository interfaces
grep -rn 'type.*Repository interface' internal/
```
