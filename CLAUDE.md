# BrandishBot Go - Claude Code Context

Go game engine API for live chatroom gaming. HTTP API (port 8080) + Discord bot (port 8082).

## Quick Commands

```bash
make run              # Start API server
make discord-run      # Start Discord bot
make test             # Run tests
make unit             # Run unit tests (fast, short mode)
make lint             # Run linters
make generate         # Regenerate SQLC/mocks
make mocks            # Regenerate mocks (uses mockery)
make migrate-up       # Apply migrations
make docker-up        # Start all services

# Devtool Helpers
make test-migrations  # Test migration idempotency
make check-deps       # Check dependencies
make check-db         # Check database status
```

---

## ⚡ Agent/Skill/MCP Usage Policy

**CRITICAL: Use specialized agents, skills, and MCPs proactively. Don't do work manually when tools are available.**

### 🤖 Task Agents (Automatic Delegation)

**Always delegate to agents for:**

| Trigger | Agent | When to Use |
|---------|-------|-------------|
| **Code review** | `code-reviewer` or `golang-pro` | When user asks to "review", before commits, after writing code |
| **Concurrency analysis** | `golang-pro` | Code with goroutines/channels/mutexes, mentions of "race", "deadlock", or working on workers/SSE/events |
| **Security audit** | `security-auditor` | User asks about security, reviewing auth/API endpoints, before production |
| **Performance optimization** | `performance-engineer` | User mentions "slow", "optimize", "bottleneck", database query optimization |
| **Database design** | `sql-pro` or `database-architect` | Designing schemas, complex queries, optimizing indexes |
| **Codebase exploration** | `Explore` | Finding patterns across files, "where is X", understanding code structure |
| **Constant extraction** | `hardcoded-constants-extractor` | After writing handlers/commands, user mentions "magic numbers" |
| **Test generation** | `test-automator` | Writing new features, test coverage requests, integration tests |
| **Refactoring** | `golang-pro` | Reducing duplication, improving code quality, modernizing patterns |

**Agent Chaining (Multi-Agent Workflows):**

Chain agents automatically for complex tasks:
- **"Review this code"** → `code-reviewer` → `security-auditor` → `golang-pro`
- **"Optimize database"** → `Explore` → `sql-pro` → `performance-engineer`
- **"Add feature X"** → `Plan` → `backend-architect` → `test-automator`
- **"Find and fix Y"** → `Explore` → specialized agent → implement fix

**Examples:**

❌ **DON'T:**
```
User: "Review the progression voting code"
Claude: *reads files manually and analyzes myself*
```

✅ **DO:**
```
User: "Review the progression voting code"
Claude: *uses golang-pro agent for Go-specific analysis*
Claude: *chains to code-reviewer for security review*
Claude: *reports findings and implements fixes*
```

---

### 🎯 Skills (Action-Trigger Based)

**Use Skills via the Skill tool for project-specific tasks:**

| User Action/Request | Skill to Use | Notes |
|---------------------|--------------|-------|
| "Check migrations", "Apply migrations" | `goose` skill | See `.agent/skills/goose/SKILL.md` |
| "Query database", "Check DB data" | `postgres` skill | See `.agent/skills/postgres/SKILL.md` |
| "Run tests", "Test this feature" | `testing` skill | See `.agent/skills/testing/SKILL.md` |
| "Deploy", "Rollback deployment" | `deployment` skill | See `.agent/skills/deployment/SKILL.md` |
| "Add Discord command", "Create slash command" | `discord` skill | See `.agent/skills/discord/SKILL.md` |
| "Modify progression tree", "Add node" | `progression` skill | See `.agent/skills/progression/SKILL.md` |
| "Concurrency patterns", "Go async best practices" | `systems-programming:go-concurrency-patterns` | Goroutines, channels, worker pools |

**Skill Detection:**
- When user mentions a specific operation (migrations, tests, deployment), use the corresponding skill
- Skills are documented in `.agent/skills/*/SKILL.md` files
- Refer to **AGENTS.md Action-Trigger Guide** for full skill mapping

---

### 🔌 MCPs (Data Access & Memory)

**Use MCPs for direct data operations:**

| MCP | When to Use | Example |
|-----|-------------|---------|
| `mcp__postgres__query` | Direct database queries for debugging/validation | `SELECT * FROM progression_nodes WHERE tier = 3` |
| `mcp__memory__*` | Store technical decisions, architecture patterns, lessons learned | Track refactoring decisions, document pain points |
| `mcp__memory__search_nodes` | Retrieve past decisions/patterns | "What did we decide about caching strategy?" |

**MCP Usage Pattern:**

```
✅ Use postgres MCP:
- Quick data validation ("How many users voted?")
- Debugging ("Show me the current voting session")
- Ad-hoc reports ("Count items by rarity")

✅ Use memory MCP:
- Document architecture decisions after making them
- Store discovered patterns for future reference
- Track technical debt items
```

---

### 🔄 Decision Tree: Which Tool to Use?

```
User asks for analysis/review/optimization?
  ├─ YES → Use Task Agent (golang-pro, code-reviewer, etc.)
  └─ NO
      ├─ User wants to run commands (migrations, tests, deploy)?
      │   └─ YES → Use Skill (goose, testing, deployment)
      └─ User wants data/memory lookup?
          └─ YES → Use MCP (postgres query, memory search)
```

---

### 📊 Usage Examples by Scenario

**Scenario: User wants to add a new feature**
```
1. Use Plan agent to design implementation
2. Use backend-architect agent for service design
3. Use Skills (goose) to create migration
4. Implement feature manually
5. Use test-automator agent for tests
6. Use hardcoded-constants-extractor for cleanup
7. Use code-reviewer before commit
```

**Scenario: User reports a bug**
```
1. Use Explore agent to find related code
2. Use golang-pro agent if concurrency-related
3. Use postgres MCP to inspect data state
4. Fix bug manually
5. Use testing skill to verify fix
6. Document in memory MCP for future reference
```

**Scenario: Performance issue**
```
1. Use Explore agent to find bottleneck locations
2. Use performance-engineer agent for analysis
3. Use sql-pro agent for query optimization
4. Use postgres MCP to test queries
5. Implement optimizations
6. Use testing skill to benchmark
```

---

### ⚠️ Anti-Patterns to Avoid

❌ Reading files manually when Explore agent should be used
❌ Analyzing Go concurrency yourself instead of using golang-pro
❌ Writing tests manually instead of using test-automator
❌ Manually checking migrations instead of using goose skill
❌ Skipping code-reviewer before commits
❌ Not documenting decisions in memory MCP

---

### 🎯 Success Criteria

You're using tools correctly when:
- ✅ Every code review request triggers an agent
- ✅ Every "find X" request uses Explore agent
- ✅ Database operations use postgres MCP for validation
- ✅ Skills are invoked for project commands (tests, migrations, deploy)
- ✅ Multi-agent workflows happen automatically for complex tasks
- ✅ Important decisions are stored in memory MCP

---

## Architecture Pattern

```
Handler (internal/handler/) → Service (internal/*/service.go) → Repository (internal/repository/) → Postgres (internal/database/postgres/)
```

All services follow this pattern. Discord commands mirror this via API client calls.

## Core Modules

| Module | Purpose |
|--------|---------|
| `internal/bootstrap/` | Application initialization, dependency injection |
| `internal/config/` | Configuration management, environment variables |
| `internal/cooldown/` | Cooldown service for rate limiting |
| `internal/features/` | Feature flags and toggles |
| `internal/harvest/` | Passive resource accumulation (farming) |
| `internal/logger/` | Structured logging with Zap |
| `internal/metrics/` | Prometheus metrics collection |
| `internal/middleware/` | HTTP middleware (CORS, logging, recovery) |
| `internal/scheduler/` | Background job scheduling |
| `internal/sse/` | Server-Sent Events for real-time updates |
| `internal/streamerbot/` | Streamer.bot WebSocket integration |
| `internal/testing/` | Test utilities and helpers |
| `internal/utils/` | Shared utility functions |

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

### Harvest System (Farming)
| Layer | Location |
|-------|----------|
| Service | `internal/harvest/service.go` |
| Repository | `internal/repository/harvest.go` |
| Postgres | `internal/database/postgres/harvest.go` |
| Handler | `internal/handler/harvest.go` |
| Discord | `internal/discord/cmd_harvest.go` |
| Tiers | `internal/harvest/reward_tiers.go` |

**Key methods:** `Harvest`
**Mechanics:** Passive accumulation over time (min 1h). Rewards spoil after 336h (2 weeks). Awards Farmer XP.

### Progression System
| Layer | Location |
|-------|----------|
| Service | `internal/progression/service.go:16-120` |
| Repository | `internal/repository/progression.go:11-70` |
| Postgres | `internal/database/postgres/progression*.go` |
| Handler | `internal/handler/progression.go` |
| Discord | `internal/discord/cmd_progression.go` |
| Loader | `internal/progression/tree_loader.go` |
| Events | `internal/progression/handler.go` |
| Domain | `internal/domain/progression.go:1-189` |
| Cost Calculator | `internal/progression/cost_calculator.go` |
| Prerequisites | `internal/progression/prerequisite_parser.go` |
| Voting Sessions | `internal/progression/voting_sessions.go` |

**Key methods:** `VoteForUnlock`, `CheckAndUnlockNode`, `GetModifiedValue`, `RecordEngagement`
**Caching:** Unlock cache (event-invalidated) + modifier cache (30-min TTL)

#### Parallel Voting Architecture

The progression system supports concurrent voting on multiple nodes with unlock accumulation:

**Voting Session Statuses:**
- `voting` - Active voting, accumulating unlocks
- `frozen` - Voting frozen by admin, accumulation continues
- `completed` - Voting ended, cycle completed

**Workflow:**
1. Users vote on multiple nodes concurrently
2. System accumulates unlocked nodes during voting period
3. When cycle completes, all accumulated unlocks are published
4. New voting session starts for next tier

**Implementation:** `internal/progression/voting_sessions.go`

**Admin Controls:**
- Freeze voting: Prevents new votes, accumulation continues
- Force-end session: Manually trigger cycle completion
- Start new session: Begin next voting cycle

#### Dynamic Prerequisites

Progression nodes can have dynamic prerequisites that evaluate at runtime:

**Prerequisite Types:**
- `nodes_unlocked_below_tier` - Requires X nodes unlocked in tiers < N
- `total_nodes_unlocked` - Requires X total nodes unlocked

**Configuration Format:**
```json
{
  "node_id": "feature_1",
  "prerequisites": ["tier:2:5", "total:10"]
}
```

**Implementation:** `internal/progression/prerequisite_parser.go`, `internal/progression/progression.go`

#### Node Cost Calculation

Node unlock costs scale by tier and size:

**Formula:** `baseCost[size] × (1.30^tier)`

**Base Costs by Size:**
- Small: 200 engagement points
- Medium: 400 engagement points
- Large: 800 engagement points

**Tier Multipliers (1.30^tier):**
- Tier 1: 1.00x
- Tier 2: 1.30x
- Tier 3: 1.69x
- Tier 4: 2.20x
- Tier 5: 2.86x

**Implementation:** `internal/progression/cost_calculator.go`

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

**Key Event Types:**
- `EventTypeEngagement` - User engagement events
- `EventGambleStarted`, `EventGambleComplete` - Gamble lifecycle
- `EventTypeJobLevelUp` - Job level progression
- `EventTypeProgressionCycleCompleted` - Voting cycle completion
- `EventTypeProgressionTargetSet` - New unlock target set
- `EventTypeProgressionVotingStarted` - Voting session started
- `EventTypeProgressionAllUnlocked` - All nodes unlocked

**Event Constructors:**
- `NewProgressionCycleCompletedEvent()` - Published when voting cycle ends
- `NewProgressionTargetSetEvent()` - Published when new target set
- `NewEngagementEvent()` - Published on user engagement

**Retry:** Exponential backoff (2s → 4s → 8s → 16s → 32s)

**Publishing Locations:**
- `internal/progression/handler.go` - Progression events
- `internal/gamble/service.go` - Gamble events
- `internal/job/service.go` - Job level events

### Server-Sent Events (SSE)
| Layer | Location |
|-------|----------|
| Hub | `internal/sse/hub.go` |
| Client | `internal/sse/client.go` |
| Handler | `internal/handler/sse.go` |
| Integration | `internal/sse/event_integration.go` |

**Endpoint:** `GET /api/v1/events`

**Event Types Broadcasted:**
- `job.level_up` - User leveled up in a job
- `progression.cycle.completed` - Voting cycle completed
- `progression.target.set` - New unlock target set
- `progression.voting_started` - New voting session started
- `progression.all_unlocked` - All nodes unlocked
- `gamble.complete` - Gamble session completed

**Architecture:**
- Hub with 100-message buffer per client
- 30-second keepalive messages
- Automatic client cleanup on disconnect

**Client Integration:**
- Discord: `internal/discord/sse_client.go`
- Streamer.bot: `internal/streamerbot/client.go`

### Cooldown System
| Layer | Location |
|-------|----------|
| Service | `internal/cooldown/service.go` |
| Repository | `internal/repository/cooldown.go` |
| Postgres | `internal/database/postgres/cooldown.go` |

**Features:**
- Check-then-lock pattern for race-free cooldown checks
- Configurable cooldown durations per action
- User-specific and global cooldowns
- Transaction-based enforcement

**Key methods:** `CheckAndSetCooldown`, `GetCooldown`, `ClearCooldown`

### Job Scheduler
| Layer | Location |
|-------|----------|
| Scheduler | `internal/scheduler/scheduler.go` |
| Jobs | `internal/scheduler/jobs/` |

**Scheduled Jobs:**
- Progression cycle management (tier progression)
- Gamble cleanup (expired sessions)
- Stats aggregation (leaderboards)

**Implementation:** Cron-based scheduling with graceful shutdown

### Streamer.bot Integration
| Layer | Location |
|-------|----------|
| Client | `internal/streamerbot/client.go` |
| WebSocket | `internal/streamerbot/websocket.go` |
| Events | `internal/streamerbot/event_handler.go` |

**Features:**
- WebSocket connection to Streamer.bot
- Real-time event forwarding
- Automatic reconnection
- Event filtering and transformation

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
- `users`, `user_platform_links` - User accounts and platform linkage
- `items`, `user_inventory` - Items and user inventories (JSONB)
- `crafting_recipes`, `disassemble_recipes` - Crafting recipes
- `progression_nodes`, `progression_unlocks`, `progression_voting_sessions` - Progression tree
- `progression_voting_options`, `progression_voting_weights` - Voting system
- `engagement_metrics` - User engagement tracking
- `gambles`, `gamble_participants`, `gamble_opened_items` - Gamble sessions
- `jobs`, `user_jobs` - Job/XP system
- `stats_events`, `events` - Event logging
- `cooldowns` - Cooldown tracking
- `loot_tables`, `loot_table_items` - Loot drop configuration

---

## Configuration Files

| Config | Location | Loader |
|--------|----------|--------|
| Items | `configs/items/items.json` | `internal/item/loader.go` |
| Crafting Recipes | `configs/recipes/crafting.json` | `internal/crafting/recipe_loader.go` |
| Disassemble Recipes | `configs/recipes/disassemble.json` | `internal/crafting/recipe_loader.go` |
| Progression Tree | `configs/progression_tree.json` | `internal/progression/tree_loader.go` |
| Loot Tables | `configs/loot_tables.json` | `internal/lootbox/service.go` |
| Item Aliases | `configs/items/aliases.json` | `internal/naming/resolver.go` |
| Item Themes | `configs/items/themes.json` | `internal/naming/resolver.go` |

---

## Common Task Patterns

### Adding a New API Endpoint
1. Define request/response in `internal/domain/`
2. Add service method in `internal/*/service.go`
3. Add repository method in `internal/repository/*.go`
4. Implement repository in `internal/database/postgres/*.go`
5. Add SQLC query in `internal/database/queries/*.sql` and run `make generate`
6. Add handler in `internal/handler/*.go`
7. Register route in `internal/server/server.go`

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

## Service Dependencies & Initialization Order

From `cmd/app/main.go`, services initialize in this order:

1. **Config & Database** - Load env vars, connect to Postgres
2. **Event System** - Initialize event bus and resilient publisher
3. **Repositories** - Create all repository instances
4. **Core Services** - User, economy, item, naming services
5. **Feature Services** - Job, stats, progression, cooldown services
6. **Game Services** - Crafting, lootbox, gamble, harvest services
7. **Background Systems** - Scheduler, gamble worker
8. **Real-Time Systems** - SSE hub, Streamer.bot client
9. **HTTP Server** - API routes and middleware
10. **Event Handlers** - Subscribe to event bus

**Dependency Relationships:**
```
user.Service      ← stats, job, lootbox, naming, cooldown
crafting.Service  ← job, stats, naming
economy.Service   ← job
gamble.Service    ← lootbox, stats, job, progression
harvest.Service   ← user, job, progression
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
| Progression Key Generator | `cmd/gen-progression-keys/main.go` |

---

## Testing

- Unit tests: `*_test.go` alongside source files
- Mocks: Generated by Mockery (`.mockery.yaml`)
- Integration tests: Use testcontainers for Postgres
- Run: `make test`

---

## API Endpoints Summary

### Health & Monitoring
- `GET /healthz` - Health check
- `GET /readyz` - Readiness check
- `GET /version` - Version info
- `GET /metrics` - Prometheus metrics

### User Management
- `POST /api/v1/user/register` - Register new user
- `GET /api/v1/user/inventory` - Get user inventory
- `GET /api/v1/user/inventory/:username` - Get inventory by username
- `PUT /api/v1/user/timeout` - Set user timeout
- `GET /api/v1/user/search` - Search users
- `POST /api/v1/user/item/add` - Add item to inventory
- `POST /api/v1/user/item/remove` - Remove item from inventory
- `POST /api/v1/user/item/use` - Use consumable item

### Economy
- `GET /api/v1/prices` - Get sellable item prices
- `GET /api/v1/prices/buy` - Get buyable item prices
- `POST /api/v1/economy/buy` - Buy item
- `POST /api/v1/economy/sell` - Sell item

### Crafting
- `POST /api/v1/user/item/upgrade` - Upgrade item
- `POST /api/v1/user/item/disassemble` - Disassemble item
- `GET /api/v1/crafting/recipes` - Get unlocked recipes

### Harvest
- `POST /api/v1/harvest` - Harvest items

### Progression
- `GET /api/v1/progression/tree` - Get progression tree
- `GET /api/v1/progression/available` - Get available nodes to unlock
- `POST /api/v1/progression/vote` - Vote for node unlock
- `GET /api/v1/progression/status` - Get current voting status
- `POST /api/v1/progression/engagement` - Record engagement
- `POST /api/v1/progression/engagement/:username` - Record engagement by username
- `GET /api/v1/progression/leaderboard` - Get engagement leaderboard
- `GET /api/v1/progression/session` - Get current voting session
- `POST /api/v1/progression/admin/freeze` - Freeze voting (admin)
- `POST /api/v1/progression/admin/force-end` - Force end session (admin)
- `POST /api/v1/progression/admin/start` - Start new session (admin)
- `PUT /api/v1/progression/admin/weights` - Update voting weights (admin)

### Gamble
- `POST /api/v1/gamble/start` - Start gamble session
- `POST /api/v1/gamble/join` - Join gamble session
- `GET /api/v1/gamble/get` - Get gamble session details

### Jobs
- `GET /api/v1/jobs` - Get all jobs
- `GET /api/v1/jobs/user` - Get user jobs
- `POST /api/v1/jobs/award-xp` - Award XP to user
- `GET /api/v1/jobs/bonus` - Get job bonus multiplier
- `POST /api/v1/admin/jobs/xp` - Award XP (admin)

### Stats
- `POST /api/v1/stats/event` - Record user event
- `GET /api/v1/stats/user` - Get user stats
- `GET /api/v1/stats/system` - Get system stats
- `GET /api/v1/stats/leaderboard` - Get leaderboard

### Message Handling
- `POST /api/v1/message/handle` - Handle chat message
- `POST /api/v1/message/test` - Test message handling

### Admin
- `POST /api/v1/admin/reload-aliases` - Reload item aliases
- `GET /api/v1/admin/cache/stats` - Get cache statistics

### Real-Time Events
- `GET /api/v1/events` - Server-Sent Events stream

### Documentation
- `/swagger/` - Swagger UI

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

### Development Artifacts Location
During development, create temporary artifacts in designated locations:
- **Plan documents:** `.claude/plans/*.md` (gitignored)
- **Temporary notes:** `.claude/notes/*.md` (gitignored)
- **Debug outputs:** `.claude/debug/*.log` or `.claude/debug/*.json` (gitignored)
- **Test data:** Use `/tmp/` or project-specific `testdata/` directories

**Never commit:** Plan documents, debug logs, temporary notes, or ephemeral test data. These locations are gitignored for this reason.

### Feature Completion Cleanup
When completing a feature or task, clean up all temporary artifacts:
- [ ] Delete plan documents (e.g., `.claude/plans/*.md`)
- [ ] Remove any temporary debugging files
- [ ] Delete test data files not needed for CI
- [ ] Clean up commented-out code blocks
- [ ] Remove TODO comments that were addressed

**Rationale:** Keep the repository clean and focused on production code. Plan documents and artifacts are useful during development but should not be committed unless they provide long-term documentation value.

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
- [ ] Route registered in `internal/server/server.go`
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
