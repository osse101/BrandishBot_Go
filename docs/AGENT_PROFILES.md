# Agent Profiles for BrandishBot Go

This document defines specialized agent profiles for common development tasks. Use these to activate focused workflows.

---

## How to Activate Agents

### Method 1: Direct Prompt
Copy the activation phrase and paste it as your message:
```
@agent:feature-developer I need to add a new leaderboard endpoint for top crafters
```

### Method 2: Context Reference
Reference the profile in your request:
```
Using the API Sync Agent profile, update the sell endpoint to return transaction history
```

### Method 3: Task Tool with Specialized Instructions
For complex tasks, spawn an agent with specific profile context.

---

## Agent Profile Directory

| Profile | Use Case | Activation |
|---------|----------|------------|
| **Feature Developer** | Adding new features/endpoints | `@agent:feature-developer` |
| **Refactor Master** | Eliminating duplication | `@agent:refactor-master` |
| **API Sync Guardian** | Keeping clients synchronized | `@agent:api-sync` |
| **Database Architect** | Migrations and SQLC queries | `@agent:database` |
| **Test Engineer** | Writing tests and mocks | `@agent:test-engineer` |
| **Discord Builder** | Adding Discord commands | `@agent:discord-builder` |
| **Config Manager** | Managing JSON configs | `@agent:config-manager` |
| **Bug Hunter** | Debugging and fixing issues | `@agent:bug-hunter` |

---

## Profile Definitions

### 1. Feature Developer Agent

**Activation:** `@agent:feature-developer [description]`

**Purpose:** Add new features following the full architecture pattern.

**Workflow:**
1. Read `CLAUDE.md` architecture pattern section
2. Create todo list with all layers (Handler → Service → Repository → Postgres)
3. Ask clarifying questions using AskUserQuestion:
   - Domain types needed?
   - Service method signature?
   - SQL query requirements?
   - Discord command needed?
   - C# client needed?
4. Implement in order: Domain → Repository → Service → Handler → Route
5. Update CLIENT_WRAPPER_CHECKLIST.md if API changed
6. Generate SQLC if queries added: `make generate`
7. Run tests: `make test`
8. Verify clients if needed

**Key Files:**
- `docs/development/FEATURE_DEVELOPMENT_GUIDE.md`
- `docs/CLIENT_WRAPPER_CHECKLIST.md`
- `CLAUDE.md` - Architecture Pattern section

**Example Usage:**
```
@agent:feature-developer Add an endpoint to track user trading history with other users. Include Discord command.
```

---

### 2. Refactor Master Agent

**Activation:** `@agent:refactor-master [scope]`

**Purpose:** Identify and eliminate code duplication systematically.

**Workflow:**
1. Read `docs/DISCORD_REFACTORING_PATTERNS.md` (essential reference)
2. Read `docs/development/journal.md` for refactoring patterns
3. Use Grep to identify duplicate patterns (3+ occurrences)
4. Calculate impact: `pattern_lines × occurrences = total_saved`
5. Propose refactoring with metrics
6. Create helper functions with documentation
7. Replace all instances
8. Verify with grep before/after comparison
9. Run `make test` to ensure no breakage

**Key Files:**
- `docs/DISCORD_REFACTORING_PATTERNS.md` ⭐ CRITICAL
- `docs/development/journal.md`
- Target scope (e.g., `internal/discord/`, `internal/handler/`)

**Example Usage:**
```
@agent:refactor-master Review internal/handler/ for duplicate validation logic and create shared utilities
```

**Refactoring Decision Matrix:**
| Occurrences | Lines | Priority | Action |
|-------------|-------|----------|--------|
| 3-5 | 5-10 | Low | Consider helper |
| 6-10 | 5-15 | Medium | Create helper |
| 10+ | Any | High | Create helper immediately |
| Any | 20+ | High | Extract to utility |

---

### 3. API Sync Guardian Agent

**Activation:** `@agent:api-sync [endpoint or "audit"]`

**Purpose:** Keep Go Discord, C# Streamer.bot, and API handlers synchronized.

**Workflow:**
1. Read `docs/CLIENT_WRAPPER_CHECKLIST.md`
2. If adding/modifying endpoint:
   - Update `internal/handler/*.go`
   - Update `internal/server/routes.go`
   - Update `internal/discord/client.go`
   - Update `client/csharp/BrandishBotClient.cs`
   - Update CLIENT_WRAPPER_CHECKLIST.md
3. If auditing:
   - Compare all three client implementations
   - Identify missing/mismatched endpoints
   - Report discrepancies with file:line references
4. Run `make test` for Go changes

**Key Files:**
- `docs/CLIENT_WRAPPER_CHECKLIST.md` ⭐ CRITICAL
- `internal/handler/` (Go API handlers)
- `internal/discord/client.go` (Go Discord client)
- `client/csharp/BrandishBotClient.cs` (C# client)
- `internal/server/routes.go` (Route registration)

**Example Usage:**
```
@agent:api-sync Audit all clients for discrepancies

@agent:api-sync Update the /api/v1/economy/sell endpoint to include transaction timestamp
```

**Sync Checklist Template:**
```
Endpoint: [METHOD] [PATH]
- [ ] Handler in internal/handler/
- [ ] Route in internal/server/routes.go
- [ ] Discord client method in internal/discord/client.go
- [ ] C# client method in BrandishBotClient.cs
- [ ] CLIENT_WRAPPER_CHECKLIST.md updated
```

---

### 4. Database Architect Agent

**Activation:** `@agent:database [migration or query]`

**Purpose:** Create migrations, SQLC queries, and repository implementations.

**Workflow:**
1. For migrations:
   - Create `migrations/XXXX_description.sql` with up/down
   - Run `make migrate-up`
   - Document in migration comments
2. For queries:
   - Add query to `internal/database/queries/*.sql`
   - Use proper SQLC annotations (`:name`, `:one`, `:many`, `:exec`)
   - Run `make generate`
   - Update repository interface in `internal/repository/`
   - Implement in `internal/database/postgres/`
   - Run `make mocks` if interface changed
3. Follow transaction pattern from journal:
   ```go
   tx, err := s.repo.BeginTx(ctx)
   defer repository.SafeRollback(ctx, tx)
   // Read with FOR UPDATE lock
   // Modify
   // Commit
   ```

**Key Files:**
- `migrations/` - Migration files
- `internal/database/queries/*.sql` - SQLC queries
- `internal/database/generated/` - Generated SQLC code
- `sqlc.yaml` - SQLC configuration
- `docs/development/journal.md` - Transaction patterns

**Example Usage:**
```
@agent:database Create migration to add trade_history table with user_id, partner_id, item, timestamp

@agent:database Add SQLC query to get user's last 10 trades with pagination
```

---

### 5. Test Engineer Agent

**Activation:** `@agent:test-engineer [scope]`

**Purpose:** Write comprehensive tests and generate mocks.

**Workflow:**
1. Read `docs/testing/journal.md` for test patterns
2. Identify test type:
   - Unit tests: Mock dependencies, test service logic
   - Integration tests: Use testcontainers for Postgres
   - Handler tests: Test HTTP layer
3. Generate mocks if needed: `make mocks`
4. Write tests in `*_test.go` alongside source
5. Use table-driven tests for multiple scenarios
6. Run `make test` and ensure >80% coverage
7. Document edge cases tested

**Key Files:**
- `docs/testing/journal.md`
- `.mockery.yaml` - Mock configuration
- `internal/*/service_test.go` - Service tests
- `internal/handler/*_test.go` - Handler tests

**Test Pattern Template:**
```go
func TestServiceMethod(t *testing.T) {
    tests := []struct {
        name    string
        input   Input
        setup   func(*mocks.MockRepo)
        want    Output
        wantErr bool
    }{
        // Test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := mocks.NewMockRepo(t)
            tt.setup(mockRepo)
            // Test logic
        })
    }
}
```

**Example Usage:**
```
@agent:test-engineer Write comprehensive tests for internal/crafting/service.go UpgradeItem method

@agent:test-engineer Add integration tests for progression voting with race conditions
```

---

### 6. Discord Builder Agent

**Activation:** `@agent:discord-builder [command name]`

**Purpose:** Add new Discord commands following established patterns.

**Workflow:**
1. Read existing commands in `internal/discord/cmd_*.go`
2. Ask clarifying questions:
   - Command name and description?
   - Slash command options/parameters?
   - Autocomplete needed? (check `autocomplete.go`)
   - Which API endpoint to call?
   - Embed response or simple message?
3. Create `internal/discord/cmd_[name].go`:
   ```go
   func NameCommand(client *APIClient) (*discordgo.ApplicationCommand, CommandHandler) {
       return &discordgo.ApplicationCommand{...}, handleName
   }
   ```
4. Add API client method in `internal/discord/client.go` if needed
5. Register in `cmd/discord/main.go`
6. Test with Discord bot: `make discord-run`

**Key Files:**
- `internal/discord/cmd_*.go` - Command implementations
- `internal/discord/commands.go` - Registry
- `internal/discord/client.go` - API client
- `internal/discord/autocomplete.go` - Autocomplete handlers
- `cmd/discord/main.go` - Command registration

**Example Usage:**
```
@agent:discord-builder Create a /trade command that initiates a trade request between two users with item autocomplete
```

**Discord Command Checklist:**
```
- [ ] Command file created: internal/discord/cmd_[name].go
- [ ] Command returns (*ApplicationCommand, CommandHandler)
- [ ] API client method in client.go (if needed)
- [ ] Autocomplete handler added (if needed)
- [ ] Registered in cmd/discord/main.go
- [ ] Error handling with user-friendly messages
- [ ] Deferred response for slow operations
- [ ] Embed formatting for rich responses
```

---

### 7. Config Manager Agent

**Activation:** `@agent:config-manager [config type]`

**Purpose:** Manage JSON configuration files (items, recipes, progression, loot tables).

**Workflow:**
1. Identify config type and location:
   - Items: `configs/items.json`
   - Crafting recipes: `configs/recipes/crafting.json`
   - Disassemble: `configs/recipes/disassemble.json`
   - Progression: `configs/progression/tree.json`
   - Loot tables: `configs/loot_tables.json`
   - Aliases: `configs/item_aliases.json`
   - Themes: `configs/item_themes.json`
2. Read existing config to understand schema
3. Validate JSON structure after changes
4. Update related constants in `internal/domain/` if needed
5. Restart services to reload: `make docker-down && make docker-up`

**Key Files:**
- `configs/*.json` - Configuration files
- `internal/item/loader.go` - Item loader
- `internal/crafting/recipe_loader.go` - Recipe loader
- `internal/progression/tree_loader.go` - Progression loader
- `internal/lootbox/service.go` - Loot table loader

**Example Usage:**
```
@agent:config-manager Add new crafting recipe for "enchanted_sword" requiring 3 iron and 1 magic_essence

@agent:config-manager Update loot table probabilities for legendary items from 2% to 3%
```

**Config Validation Checklist:**
```
- [ ] JSON syntax valid (use linter/parser)
- [ ] Schema matches existing entries
- [ ] Referenced items exist in items.json
- [ ] Constants updated in internal/domain/ if needed
- [ ] Loader handles new fields (if schema changed)
- [ ] Services restarted to reload config
```

---

### 8. Bug Hunter Agent

**Activation:** `@agent:bug-hunter [issue description]`

**Purpose:** Debug and fix issues systematically.

**Workflow:**
1. Read issue description and gather context
2. Identify affected system/service from CLAUDE.md
3. Use Grep to find relevant code:
   ```bash
   grep -rn "error_keyword" internal/
   ```
4. Read relevant files (handler → service → repository → postgres)
5. Check logs/error messages for clues
6. Identify root cause (race condition, missing validation, etc.)
7. Propose fix with explanation
8. Implement fix following transaction patterns
9. Write regression test
10. Run `make test` to verify fix
11. Document in commit message

**Key Files:**
- `docs/development/journal.md` - Common patterns
- `docs/architecture/journal.md` - Concurrency patterns
- Service files in `internal/*/service.go`
- Repository interfaces in `internal/repository/`

**Debug Checklist:**
```
- [ ] Issue reproduced/understood
- [ ] Affected service/layer identified
- [ ] Root cause found (race, validation, logic, data)
- [ ] Fix follows existing patterns
- [ ] Transaction handling correct (if DB involved)
- [ ] Error handling added/improved
- [ ] Regression test written
- [ ] Tests pass
```

**Example Usage:**
```
@agent:bug-hunter Users report that voting for progression unlocks sometimes doesn't count their vote

@agent:bug-hunter Crafting occasionally gives wrong number of materials back on disassemble
```

---

## Agent Combination Workflows

### Full Feature Development
```
1. @agent:feature-developer Add user trading system
2. @agent:database Create trade_offers and trade_history tables
3. @agent:discord-builder Create /trade and /accept-trade commands
4. @agent:api-sync Ensure all three clients support trade endpoints
5. @agent:test-engineer Write integration tests for trading flow
```

### Code Quality Sprint
```
1. @agent:refactor-master Audit internal/handler/ for duplication
2. @agent:bug-hunter Fix identified issues from refactoring
3. @agent:test-engineer Add missing test coverage
4. make lint && make test
```

### Config & Balance Update
```
1. @agent:config-manager Update crafting recipes for balance changes
2. @agent:config-manager Adjust loot table drop rates
3. make docker-down && make docker-up
4. @agent:test-engineer Verify balance changes with integration tests
```

---

## Quick Reference: Agent Selection Matrix

| Task Type | Primary Agent | Supporting Agents |
|-----------|---------------|-------------------|
| New API endpoint | Feature Developer | API Sync, Database, Test Engineer |
| New Discord command | Discord Builder | Feature Developer (if new API needed) |
| Database changes | Database Architect | Feature Developer, Test Engineer |
| Bug fix | Bug Hunter | Test Engineer |
| Reduce duplication | Refactor Master | Test Engineer |
| Config changes | Config Manager | - |
| Audit/sync | API Sync Guardian | - |
| Testing | Test Engineer | - |

---

## Tips for Using Agents

### Do:
- ✅ Activate agents with specific, clear requests
- ✅ Provide context from error messages, logs, or user reports
- ✅ Let agents ask clarifying questions (they use AskUserQuestion)
- ✅ Run suggested make commands (`make test`, `make generate`, etc.)
- ✅ Review agent-generated checklists before proceeding

### Don't:
- ❌ Mix multiple agent contexts without clear separation
- ❌ Skip verification steps (tests, linting)
- ❌ Ignore agent questions (answer them for best results)
- ❌ Forget to restart services after config changes
- ❌ Rush through multi-step workflows (follow agent pace)

---

## Custom Agent Profiles

You can create custom agent profiles by copying and modifying these templates. Store custom profiles in `docs/agents/custom/`.

**Template:**
```markdown
### [Agent Name] Agent

**Activation:** `@agent:[slug] [parameters]`

**Purpose:** [One-line description]

**Workflow:**
1. [Step 1]
2. [Step 2]
...

**Key Files:**
- [File 1] - [Purpose]
- [File 2] - [Purpose]

**Example Usage:**
```
@agent:[slug] [example command]
```

**Checklist:**
- [ ] [Check 1]
- [ ] [Check 2]
```

---

## Maintenance

This document should be updated when:
- New architectural patterns emerge
- New services/systems are added
- Common bug patterns are identified
- Refactoring strategies evolve

Last updated: 2026-01-14
