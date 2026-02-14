# Feature Implementation Tasks

3 feature nodes requiring unlock gate implementation.

8 out of 11 total features already have gates implemented. These are the remaining 3.

## Current Status (2026-02-02)

**Implementation Progress**:
- ✅ **Compost**: Fully implemented (Service, Repository, Engine, Handlers, Discord Commands).
- ✅ **Expeditions**: Fully implemented (Service, Repository, Engine, Handlers, Discord Commands, Worker).
- 🚧 **Duels**: Structurally implemented but missing business logic (`Accept` method), Repository implementation, and Discord commands.

**Next phase**: Complete Duels implementation and verify integration tests.

---

## Tier 3 Features

### Duels Minigame (`feature_duel`)

**Type**: feature | **Tier**: 3 | **Size**: large

**Prerequisites**: job_gambler

**Description**: Bet your lives in this game of chance

**Implementation Checklist**:

- [x] Add duel feature service (✅ `internal/duel/service.go`)
- [x] Add unlock check in duel handlers (✅ `CheckFeatureLocked` in all handlers)
- [x] Create domain types (✅ `internal/domain/duel.go`)
- [x] Create repository interface (✅ `internal/repository/duel.go`)
- [x] Create database migration (✅ `migrations/0010_add_duels.sql`)
- [x] Create HTTP handlers (✅ `internal/handler/duel.go`)
- [ ] Implement duel execution logic (coin flip/dice roll in `Accept()`)
- [ ] Implement repository (Postgres implementation)
- [ ] Register routes in `internal/server/server.go`
- [ ] Add tests for duel when locked/unlocked
- [ ] Verify with admin unlock: `curl -X POST .../admin/unlock -d '{"node_key": "feature_duel", "level": 1}'`
- [ ] Test locked behavior (duel feature unavailable)
- [ ] Test unlocked behavior (can challenge others to duels)
- [ ] Create Discord commands (`/duel challenge`, `/duel accept`, etc.)
- [ ] Update API client wrappers (Go Discord, C# Streamer.bot)

**Files Created**:

- ✅ `internal/handler/duel.go` - 5 endpoints (challenge, accept, decline, get pending, get duel)
- ✅ `internal/duel/service.go` - Service with placeholder logic
- ✅ `internal/domain/duel.go` - Domain types (DuelState, DuelStakes, DuelResult)
- ✅ `internal/repository/duel.go` - Repository interface with transaction support
- ✅ `migrations/0010_add_duels.sql` - Database schema
- ⏳ `internal/duel/service_test.go` - Not yet created
- ⏳ `internal/database/postgres/duel.go` - Not yet created
- ⏳ `internal/discord/cmd_duel.go` - Not yet created

**Acceptance Criteria**:

- ✓ Cannot access duel feature when locked
- ✓ Can initiate duels when unlocked
- ✓ Error message: "Duels feature not unlocked"
- ✓ Tests cover locked and unlocked states

**Implementation Details**:

Duels are a gambling minigame where users bet their "lives" (timeout risk) in a game of chance:

1. **Challenge Phase**: User A challenges User B to a duel
2. **Accept Phase**: User B accepts or declines
3. **Duel Phase**: Random outcome determines winner
4. **Result Phase**: Loser receives timeout, winner gets reward

**Suggested Service Structure**:

```go
// internal/duel/service.go
package duel

type Service struct {
    repo              repository.Duel
    progressionService progression.Service
    userService       user.Service
}

func (s *Service) ChallengeDuel(ctx context.Context, challengerID, targetID string, stakes int) error {
    // Check duel feature unlock
    unlocked, err := s.progressionService.IsNodeUnlocked(ctx, progression.FeatureDuel)
    if err != nil {
        return fmt.Errorf("failed to check duel unlock: %w", err)
    }
    if !unlocked {
        return ErrDuelNotUnlocked
    }

    // Create duel challenge
    // Return duel ID
}

func (s *Service) AcceptDuel(ctx context.Context, duelID string, accepterID string) (*DuelResult, error) {
    // Verify both users can duel
    // Execute random duel logic
    // Apply timeout to loser
    // Award winner
    // Return result
}
```

**API Endpoints**:

- `POST /api/v1/duel/challenge` - Initiate duel
- `POST /api/v1/duel/:id/accept` - Accept duel challenge
- `POST /api/v1/duel/:id/decline` - Decline duel challenge
- `GET /api/v1/duel/pending` - List pending duel challenges

**Estimated Effort**: Medium (4-6 hours) - New service module, PvP mechanics, timeout integration

---

### Compost Feature (`feature_compost`)

**Type**: feature | **Tier**: 3 | **Size**: large

**Prerequisites**: job_farmer

**Description**: Turn junk into gems through the passage of time

**Implementation Checklist**:

- [x] Add compost feature service (✅ `internal/compost/service.go`)
- [x] Add unlock check in compost handlers (✅ `CheckFeatureLocked` in all handlers)
- [x] Create domain types (✅ `internal/domain/compost.go`)
- [x] Create repository interface (✅ `internal/repository/compost.go`)
- [x] Create database migration (✅ `migrations/0011_add_compost.sql`)
- [x] Create HTTP handlers (✅ `internal/handler/compost.go`)
- [x] Implement conversion rate logic (✅ `internal/compost/engine.go`)
- [x] Implement harvest logic with inventory updates (✅ `internal/compost/service.go`)
- [x] Implement repository (✅ `internal/database/postgres/compost.go`)
- [x] Register routes in `internal/server/server.go`
- [ ] Add tests for compost when locked/unlocked
- [ ] Verify with admin unlock: `curl -X POST .../admin/unlock -d '{"node_key": "feature_compost", "level": 1}'`
- [ ] Test locked behavior (compost feature unavailable)
- [ ] Test unlocked behavior (can convert junk to gems over time)
- [x] Create Discord commands (✅ `internal/discord/cmd_compost.go`)
- [ ] Update API client wrappers (Go Discord, C# Streamer.bot)

**Files Created**:

- ✅ `internal/handler/compost.go` - 3 endpoints (deposit, status, harvest)
- ✅ `internal/compost/service.go` - Service with core logic
- ✅ `internal/compost/engine.go` - Engine for conversion logic
- ✅ `internal/domain/compost.go` - Domain types (CompostDeposit, CompostStatus, CompostMetadata)
- ✅ `internal/repository/compost.go` - Repository interface with transaction support
- ✅ `migrations/0011_add_compost.sql` - Database schema
- ⏳ `internal/compost/service_test.go` - Not yet created
- ✅ `internal/database/postgres/compost.go` - Postgres implementation
- ✅ `internal/discord/cmd_compost.go` - Discord commands

**Acceptance Criteria**:

- ✓ Cannot access compost feature when locked
- ✓ Can convert junk items to gems over time when unlocked
- ✓ Error message: "Compost feature not unlocked"
- ✓ Tests cover locked and unlocked states

---

## Implementation Details

### Compost Mechanics (Proposed)

Based on the description "Turn junk into gems through the passage of time":

1. **Deposit Phase**: User deposits unwanted items into compost bin
2. **Composting Phase**: Items convert over time (e.g., 24-48 hours)
3. **Harvest Phase**: User retrieves converted gems/currency

### Suggested Implementation

#### Service Structure

```go
// internal/compost/service.go
package compost

type Service struct {
    repo              repository.Compost
    progressionService progression.Service
    itemService       item.Service
}

func (s *Service) DepositItem(ctx context.Context, userID, itemKey string, quantity int) error {
    // Check compost feature unlock
    unlocked, err := s.progressionService.IsNodeUnlocked(ctx, progression.FeatureCompost)
    if err != nil {
        return fmt.Errorf("failed to check compost unlock: %w", err)
    }
    if !unlocked {
        return ErrCompostNotUnlocked
    }

    // Remove item from inventory
    // Add to compost bin with timestamp
    // Return success
}

func (s *Service) HarvestCompost(ctx context.Context, userID string) ([]domain.Item, error) {
    // Check compost feature unlock
    unlocked, err := s.progressionService.IsNodeUnlocked(ctx, progression.FeatureCompost)
    if err != nil {
        return nil, fmt.Errorf("failed to check compost unlock: %w", err)
    }
    if !unlocked {
        return nil, ErrCompostNotUnlocked
    }

    // Find composted items past threshold time
    // Convert to gems based on item value
    // Add gems to user inventory
    // Return converted items
}
```

#### API Endpoints

```go
// POST /api/v1/compost/deposit
type DepositRequest struct {
    ItemKey  string `json:"item_key"`
    Quantity int    `json:"quantity"`
}

// GET /api/v1/compost/status
// Returns current compost bin contents and completion times

// POST /api/v1/compost/harvest
// Harvests all completed compost conversions
```

#### Database Schema

```sql
-- Add to migrations
CREATE TABLE compost_deposits (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    item_key VARCHAR(255) NOT NULL,
    quantity INT NOT NULL,
    deposited_at TIMESTAMP NOT NULL DEFAULT NOW(),
    harvest_ready_at TIMESTAMP NOT NULL,
    harvested_at TIMESTAMP,
    gems_awarded INT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_compost_user_id ON compost_deposits(user_id);
CREATE INDEX idx_compost_harvest_ready ON compost_deposits(harvest_ready_at) WHERE harvested_at IS NULL;
```

#### Conversion Rates

Suggested conversion based on item rarity/value:

| Item Type | Compost Time | Gem Output   |
| --------- | ------------ | ------------ |
| Common    | 24 hours     | 10-20 gems   |
| Uncommon  | 36 hours     | 30-50 gems   |
| Rare      | 48 hours     | 75-100 gems  |
| Epic      | 72 hours     | 150-200 gems |

### Testing Pattern

```go
func TestCompost_Locked(t *testing.T) {
    // Setup mock progression service returning false for IsNodeUnlocked
    // Attempt to deposit item to compost
    // Assert error is ErrCompostNotUnlocked
}

func TestCompost_Unlocked_Deposit(t *testing.T) {
    // Setup mock progression service returning true
    // Deposit junk item to compost
    // Assert item removed from inventory
    // Assert compost deposit created with future harvest time
}

func TestCompost_Unlocked_Harvest(t *testing.T) {
    // Setup compost deposits that are ready to harvest
    // Call HarvestCompost
    // Assert gems added to inventory
    // Assert compost deposits marked as harvested
}

func TestCompost_Harvest_NotReady(t *testing.T) {
    // Setup compost deposits that aren't ready yet
    // Call HarvestCompost
    // Assert no gems awarded
    // Assert deposits remain unharvested
}
```

### Discord Bot Integration

```go
// /compost deposit <item> <quantity>
// /compost status
// /compost harvest
```

### Feature Dependencies

**Depends on**:

- `job_farmer` (prerequisite)
- Item system (for removing/adding items)
- Time-based job system (scheduler)

**Related upgrades** (potential future):

- Compost speed upgrade (reduce conversion time)
- Compost efficiency upgrade (increase gem output)

### Implementation Steps

1. **Database Migration** - Create compost_deposits table
2. **Repository Layer** - Add compost repository methods
3. **Service Layer** - Implement DepositItem, HarvestCompost, GetStatus
4. **Handler Layer** - Add HTTP endpoints
5. **Scheduler Integration** - Add background job to check ready compost
6. **Discord Commands** - Add /compost commands
7. **Tests** - Add comprehensive unit and integration tests
8. **Documentation** - Update API docs

### Estimated Effort

**Medium-High Complexity** (6-10 hours):

- New service module required
- Database schema needed
- Time-based mechanics
- Scheduler integration
- Multi-layer implementation

### Alternative: Simpler Implementation

If full time-based system is too complex initially:

**Instant Conversion**:

- User deposits junk items
- Immediately converts to gems at reduced rate
- No waiting period
- Simpler implementation (2-3 hours)

This can be enhanced later to add time-based conversion for better gem rates.

---

## Tier 4 Features

### Expeditions (`feature_expedition`)

**Type**: feature | **Tier**: 4 | **Size**: large

**Prerequisites**: job_explorer

**Description**: Unlock expedition/adventure system

**Implementation Checklist**:

- [x] Add expedition feature service (✅ `internal/expedition/service.go`)
- [x] Add unlock check in expedition handlers (✅ `CheckFeatureLocked` in all handlers)
- [x] Create domain types (✅ `internal/domain/expedition.go`)
- [x] Create repository interface (✅ `internal/repository/expedition.go`)
- [x] Create database migration (✅ `migrations/0012_add_expeditions.sql`)
- [x] Create HTTP handlers (✅ `internal/handler/expedition.go`)
- [x] Implement expedition execution logic (✅ `internal/expedition/engine.go`)
- [x] Implement repository (✅ `internal/database/postgres/expedition.go`)
- [x] Create expedition worker (✅ `internal/worker/expedition_worker.go`)
- [x] Integrate worker with scheduler
- [x] Register routes in `internal/server/server.go`
- [ ] Add tests for expedition when locked/unlocked
- [ ] Verify with admin unlock: `curl -X POST .../admin/unlock -d '{"node_key": "feature_expedition", "level": 1}'`
- [ ] Test locked behavior (expedition feature unavailable)
- [ ] Test unlocked behavior (can start expeditions)
- [x] Create Discord commands (✅ `internal/discord/cmd_expedition.go`)
- [ ] Update API client wrappers (Go Discord, C# Streamer.bot)

**Files Created**:

- ✅ `internal/handler/expedition.go` - 4 endpoints (start, join, get, get active)
- ✅ `internal/expedition/service.go` - Service with core logic
- ✅ `internal/expedition/engine.go` - Engine for encounter logic
- ✅ `internal/domain/expedition.go` - Domain types (ExpeditionState, ExpeditionMetadata, ExpeditionRewards)
- ✅ `internal/repository/expedition.go` - Repository interface with transaction support
- ✅ `migrations/0012_add_expeditions.sql` - Database schema (expeditions + participants)
- ⏳ `internal/expedition/service_test.go` - Not yet created
- ✅ `internal/database/postgres/expedition.go` - Postgres implementation
- ✅ `internal/worker/expedition_worker.go` - Background worker
- ✅ `internal/discord/cmd_expedition.go` - Discord commands

**Acceptance Criteria**:

- ✓ Cannot access expedition feature when locked
- ✓ Can start expeditions when unlocked
- ✓ Error message: "Expedition feature not unlocked"
- ✓ Tests cover locked and unlocked states

**Implementation Details**:

Expeditions are adventure-based activities where users send characters on timed missions to find items [feature document](../expedition_system.md):

1. **Start Phase**: User starts expedition with chosen difficulty/duration
2. **Active Phase**: Expedition runs for set time (e.g., 1-24 hours)
3. **Complete Phase**: User collects expedition rewards
4. **Cooldown Phase**: Wait before next expedition

**Suggested Service Structure**:

```go
// internal/expedition/service.go
package expedition

type Service struct {
    repo              repository.Expedition
    progressionService progression.Service
    itemService       item.Service
    jobService        job.Service
}

func (s *Service) StartExpedition(ctx context.Context, userID string, expeditionType string) error {
    // Check expedition feature unlock
    unlocked, err := s.progressionService.IsNodeUnlocked(ctx, progression.FeatureExpedition)
    if err != nil {
        return fmt.Errorf("failed to check expedition unlock: %w", err)
    }
    if !unlocked {
        return ErrExpeditionNotUnlocked
    }

    // Check user not on active expedition
    // Start expedition with completion time
    // Award explorer job XP
    // Return expedition ID
}

func (s *Service) CompleteExpedition(ctx context.Context, userID string, expeditionID int) ([]domain.Item, error) {
    // Check expedition is complete (past completion time)
    // Generate rewards based on expedition type and explorer level
    // Award items to user
    // Mark expedition complete
    // Return reward items
}
```

**API Endpoints**:

- `POST /api/v1/expedition/start` - Start new expedition
- `GET /api/v1/expedition/active` - Get user's active expedition
- `POST /api/v1/expedition/:id/complete` - Complete and collect rewards
- `GET /api/v1/expedition/available` - List available expedition types

**Expedition Types** (Suggested):
| Type | Duration | Difficulty | Rewards |
|------|----------|------------|---------|
| Quick Scout | 1 hour | Easy | Common items |
| Exploration | 4 hours | Medium | Uncommon items |
| Deep Dive | 12 hours | Hard | Rare items |
| Legendary Quest | 24 hours | Very Hard | Epic/Legendary |

**Database Schema**:

```sql
CREATE TABLE expeditions (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    expedition_type VARCHAR(50) NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    complete_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    rewards JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_expeditions_user_id ON expeditions(user_id);
CREATE INDEX idx_expeditions_complete_at ON expeditions(complete_at) WHERE completed_at IS NULL;
```

**Integration with Explorer Job**:

- Award explorer XP when starting expeditions
- Explorer level improves expedition rewards
- Higher levels unlock better expedition types

**Discord Bot Integration**:

```go
// /expedition start <type>
// /expedition status
// /expedition complete
```

**Estimated Effort**: High (8-12 hours) - New service module, time-based mechanics, reward generation system, scheduler integration

---

## Notes

- 3 unimplemented features remaining (8/11 done)
- Tier 3 feature suggests mid-game mechanic
- Farming prerequisite ties to resource management theme
- Description implies time-based conversion system
- Could integrate with scheduler for background processing
- May need new database tables for tracking compost state

## Priority

Recommended implementation order:

1. **feature_duel** (Tier 3) - PvP minigame, builds on gamble system
2. **feature_compost** (Tier 3) - Resource conversion, complements farming
3. **feature_expedition** (Tier 4) - Adventure system, endgame content

**Overall Priority**: High

- Completes feature gate implementation (100% coverage)
- Adds major gameplay systems (PvP duels, expeditions, resource conversion)
- Unlocks job progression (gambler → duel, explorer → expedition, farmer → compost)

## Related Systems

### Duels

- **Gamble** - Similar random outcome mechanics
- **User Timeout** - Applies timeout penalties
- **Job (Gambler)** - Prerequisites and XP integration

### Compost

- **Farming** - Primary activity for job_farmer
- **Economy** - Provides alternative currency acquisition
- **Inventory** - Manages item deposits and gem rewards
- **Scheduler** - Handles time-based conversion checks

### Expeditions

- **Exploration** - Core adventure mechanics
- **Job (Explorer)** - Prerequisites and XP rewards
- **Loot** - Reward generation system
- **Scheduler** - Time-based expedition management

---

## Implementation Progress Summary

### Phase 1: Structural Foundation ✅ COMPLETE (2026-02-01)

All three features have the following **structural foundation** implemented:

| Component | Duels | Compost | Expeditions |
|-----------|-------|---------|-------------|
| Database Migration | ✅ 0010 | ✅ 0011 | ✅ 0012 |
| Domain Types | ✅ duel.go | ✅ compost.go | ✅ expedition.go |
| Repository Interface | ✅ repository/duel.go | ✅ repository/compost.go | ✅ repository/expedition.go |
| Service Layer | ✅ duel/service.go | ✅ compost/service.go | ✅ expedition/service.go |
| HTTP Handlers | ✅ handler/duel.go | ✅ handler/compost.go | ✅ handler/expedition.go |
| Progression Checks | ✅ All handlers | ✅ All handlers | ✅ All handlers |
| Build Status | ✅ Compiles | ✅ Compiles | ✅ Compiles |

**API Endpoints Created:**
- **Duels**: 5 endpoints (challenge, accept, decline, get pending, get duel)
- **Compost**: 3 endpoints (deposit, status, harvest)
- **Expeditions**: 4 endpoints (start, join, get, get active)

**Total**: 12 new API endpoints ready for implementation

### Phase 2: Business Logic & Implementation 🚧 IN PROGRESS

The following components need to be implemented for each feature:

| Task | Duels | Compost | Expeditions |
|------|-------|---------|-------------|
| Postgres Repository | ⏳ | ✅ | ✅ |
| Business Logic | ⏳ Accept() | ✅ | ✅ |
| Route Registration | ⏳ | ✅ | ✅ |
| Worker Integration | N/A | N/A | ✅ |
| Discord Commands | ⏳ | ✅ | ✅ |
| Client Wrappers | ⏳ | ⏳ | ⏳ |
| Unit Tests | ⏳ | ⏳ | ⏳ |
| Integration Tests | ⏳ | ⏳ | ⏳ |

### Next Steps (Priority Order)

1. **Implement Postgres repositories** for all three features
2. **Register routes** in `internal/server/server.go`
3. **Implement business logic**:
   - Duels: Random outcome logic (coin flip/dice roll)
   - Compost: Conversion rate calculations (item rarity → gems)
   - Expeditions: Reward generation system + worker
4. **Create Discord commands** for all three features
5. **Update client wrappers** (Go Discord + C# Streamer.bot)
6. **Write tests** (unit + integration)
7. **Test with admin unlock** and verify locked/unlocked states

### Estimated Remaining Effort

- **Duels**: 6-8 hours (repository, business logic, Discord, tests)
- **Compost**: 8-10 hours (repository, conversion logic, Discord, tests)
- **Expeditions**: 12-15 hours (repository, execution logic, worker, Discord, tests)

**Total**: ~26-33 hours to complete all three features
