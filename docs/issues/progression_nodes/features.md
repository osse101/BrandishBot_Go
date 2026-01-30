# Feature Implementation Tasks

3 feature nodes requiring unlock gate implementation.

8 out of 11 total features already have gates implemented. These are the remaining 3.

---

## Tier 3 Features

### Duels Minigame (`feature_duel`)

**Type**: feature | **Tier**: 3 | **Size**: large

**Prerequisites**: job_gambler

**Description**: Bet your lives in this game of chance

**Implementation Checklist**:

- [ ] Add duel feature service (if not exists)
- [ ] Add unlock check in duel handlers
- [ ] Update duel initiation logic to verify unlock
- [ ] Add tests for duel when locked/unlocked
- [ ] Verify with admin unlock: `curl -X POST .../admin/unlock -d '{"node_key": "feature_duel", "level": 1}'`
- [ ] Test locked behavior (duel feature unavailable)
- [ ] Test unlocked behavior (can challenge others to duels)

**Files to Modify**:

- `internal/handler/duel.go` (create if doesn't exist) - Add duel endpoints
- `internal/duel/service.go` (create if doesn't exist) - Add duel logic
- `internal/duel/service_test.go` (create if doesn't exist) - Add tests
- `internal/server/server.go` - Register duel routes

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

- [ ] Add compost feature service (if not exists)
- [ ] Add unlock check in compost handlers
- [ ] Update compost conversion logic to verify unlock
- [ ] Add tests for compost when locked/unlocked
- [ ] Verify with admin unlock: `curl -X POST .../admin/unlock -d '{"node_key": "feature_compost", "level": 1}'`
- [ ] Test locked behavior (compost feature unavailable)
- [ ] Test unlocked behavior (can convert junk to gems over time)

**Files to Modify**:

- `internal/handler/compost.go` (create if doesn't exist) - Add compost endpoints
- `internal/compost/service.go` (create if doesn't exist) - Add compost conversion logic
- `internal/compost/service_test.go` (create if doesn't exist) - Add tests
- `internal/server/server.go` - Register compost routes

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

- [ ] Add expedition feature service (if not exists)
- [ ] Add unlock check in expedition handlers
- [ ] Update expedition logic to verify unlock
- [ ] Add tests for expedition when locked/unlocked
- [ ] Verify with admin unlock: `curl -X POST .../admin/unlock -d '{"node_key": "feature_expedition", "level": 1}'`
- [ ] Test locked behavior (expedition feature unavailable)
- [ ] Test unlocked behavior (can start expeditions)

**Files to Modify**:

- `internal/handler/expedition.go` (create if doesn't exist) - Add expedition endpoints
- `internal/expedition/service.go` (create if doesn't exist) - Add expedition logic
- `internal/expedition/service_test.go` (create if doesn't exist) - Add tests
- `internal/server/server.go` - Register expedition routes

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
