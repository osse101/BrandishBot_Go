# Feature Proposal — Job System v1

## 1. Title
**Short name:** Job System v1  
**Canonical ID:** feature/job-system-v1

## 2. One-line summary
A profession-based XP and leveling system where users accumulate experience in specific jobs through game activities, unlocking bonuses and gating content by job level.

## 3. Goal(s)
- Provide persistent user progression through profession specialization
- Create RPG-style identity through job levels (e.g., "Master Blacksmith")
- Gate advanced content behind job level requirements
- Reward consistent feature usage with scaling bonuses
- Integrate with existing progression system for job unlock voting

## 4. Scope
### In scope
- Per-user, per-job XP tracking
- XP → Level scaling system
- Job-specific bonuses (Blacksmith, Ranger, Merchant, etc.)
- Feature-to-job XP attribution
- Level cap tied to progression system unlocks
- Job identity (highest-level job)
- Recipe/item level requirements

### Out of scope
- Job class abilities (active skills)
- Job quests or missions
- Multi-job combinations or subclasses
- Prestige/rebirth mechanics (v2)
- Job-specific equipment bonuses

## 5. Motivation
Jobs add vertical progression that rewards consistent play. Unlike global progression unlocks, jobs are **personal** — each user develops their own specialization. This creates diverse player identities and encourages feature exploration.

---

## 6. References / Constraint Docs
- Architecture constraints: [ARCHITECTURE.md](../architecture/ARCHITECTURE.md)
- Security analysis: [SECURITY_ANALYSIS.md](../archived/SECURITY_ANALYSIS.md)
- Migrations guide: [MIGRATIONS.md](../database/MIGRATIONS.md)

---

## 7. Job Definitions

| Job | Associated Features | XP Sources | Bonus Type |
|-----|---------------------|------------|------------|
| **Blacksmith** | Upgrade, Craft, Disassemble | Recipe quality, items disassembled | Recipe level requirements, upgrade success rate, disassembly yields |
| **Explorer** | Search | Each search command | % chance for bonus money, amount scales with level |
| **Merchant** | Buy, Sell | Transactions | Better prices (buy lower, sell higher) |
| **Gambler** | Gamble | Lootbox value wagered | Increased prize when winning (small %) |
| **Farmer** | Farm (future) | Harvests completed | Faster grow times, better yields |
| **Scholar** | Community Progression | Actions contributing to global progression | XP bonus to all jobs, engagement score multiplier |

---

## 8. XP and Leveling System

### XP Scaling Formula
Leveling is intentionally **slow** to encourage long-term engagement.

```
XP required for level N = BASE_XP × (N ^ EXPONENT)

BASE_XP = 100
EXPONENT = 1.5
```

| Level | XP Required | Cumulative XP |
|-------|-------------|---------------|
| 1 | 100 | 100 |
| 2 | 283 | 383 |
| 5 | 1,118 | 3,043 |
| 10 | 3,162 | ~18,000 |
| 20 | 8,944 | ~100,000 |
| 50 | 35,355 | ~750,000 |

### Level Caps
Levels are capped by the **Job XP** node in the progression tree:

| Progression Unlock Level | Max Job Level |
|--------------------------|---------------|
| 1 (initial unlock) | 10 |
| 2 | 20 |
| 3 | 30 |
| ... | +10 per level |

### XP Boost Progression Node
A separate progression node **Job XP Boost** provides:

| Boost Node Level | Effect |
|------------------|--------|
| 1 | +25% XP gain, all new users start at job level 1 |
| 2 | +50% XP gain, all new users start at job level 2 |
| 3 | +75% XP gain, all new users start at job level 3 |
| 4 | +100% XP gain, all new users start at job level 5 |
| ... | Continues scaling |

**Note**: Daily XP caps (if enabled) also scale with this node.

---

## 9. Data Model (DDL Sketch)

### `jobs`
Static job definitions (seeded at migration).

```sql
CREATE TABLE jobs (
    id SERIAL PRIMARY KEY,
    job_key TEXT UNIQUE NOT NULL,          -- 'blacksmith', 'ranger', etc.
    display_name TEXT NOT NULL,            -- 'Blacksmith'
    description TEXT,
    associated_features TEXT[],            -- ['upgrade', 'craft']
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### `user_jobs`
Per-user job XP and level tracking.

```sql
CREATE TABLE user_jobs (
    user_id UUID NOT NULL REFERENCES users(user_id),
    job_id INT NOT NULL REFERENCES jobs(id),
    current_xp BIGINT NOT NULL DEFAULT 0,
    current_level INT NOT NULL DEFAULT 0,
    xp_gained_today BIGINT DEFAULT 0,      -- For daily caps (optional)
    last_xp_gain TIMESTAMPTZ,
    PRIMARY KEY (user_id, job_id)
);

CREATE INDEX idx_user_jobs_user ON user_jobs(user_id);
CREATE INDEX idx_user_jobs_level ON user_jobs(current_level DESC);
```

### `job_xp_events`
Audit log for XP gains (optional, for debugging/analytics).

```sql
CREATE TABLE job_xp_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(user_id),
    job_id INT NOT NULL REFERENCES jobs(id),
    xp_amount INT NOT NULL,
    source_type TEXT NOT NULL,             -- 'upgrade', 'search', 'sell'
    source_metadata JSONB,                 -- { "recipe_id": 5, "quality": "rare" }
    recorded_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_job_xp_events_user ON job_xp_events(user_id, recorded_at DESC);
```

### `job_level_bonuses`
Configurable bonuses per job per level tier.

```sql
CREATE TABLE job_level_bonuses (
    id SERIAL PRIMARY KEY,
    job_id INT NOT NULL REFERENCES jobs(id),
    min_level INT NOT NULL,
    bonus_type TEXT NOT NULL,              -- 'money_multiplier', 'success_rate', 'chance_bonus'
    bonus_value DECIMAL(10,4) NOT NULL,    -- 0.05 = 5%, 1.25 = 125%
    description TEXT,
    UNIQUE (job_id, min_level, bonus_type)
);
```

---

## 10. Progression Tree Integration

### New Progression Nodes

```
progression_system
└── jobs_xp (feature, multi-level)       -- Unlocks job XP system
    ├── jobs_xp_boost (upgrade, multi-level) -- XP multiplier + starting levels
    ├── job_blacksmith (feature)         -- Prerequisite: jobs_xp
    ├── job_explorer (feature)           -- Prerequisite: jobs_xp, search
    ├── job_merchant (feature)           -- Prerequisite: jobs_xp, economy
    ├── job_gambler (feature)            -- Prerequisite: jobs_xp, gamble
    ├── job_farmer (feature)             -- Prerequisite: jobs_xp, farm (future)
    └── job_scholar (feature)            -- Prerequisite: jobs_xp
```

### Multi-Level Nodes

#### `jobs_xp` (Level Cap)
| Level | Effect |
|-------|--------|
| 1 | Unlock job XP system, max job level 10 |
| 2 | Max job level 20 |
| 3 | Max job level 30 |
| ... | +10 max level per unlock |

#### `jobs_xp_boost` (XP Acceleration)
| Level | XP Multiplier | New User Starting Level |
|-------|---------------|-------------------------|
| 1 | 1.25x | 1 |
| 2 | 1.50x | 2 |
| 3 | 1.75x | 3 |
| 4 | 2.00x | 5 |

Each level requires a new vote and higher engagement score.

---

## 11. API & Events

### External API

```http
GET /jobs
Authorization: X-API-Key: <api_key>

Response 200:
{
  "jobs": [
    {
      "job_key": "blacksmith",
      "display_name": "Blacksmith",
      "description": "Masters of crafting and upgrades",
      "unlocked": true
    }
  ]
}

---

GET /jobs/{user_id}
Authorization: X-API-Key: <api_key>

Response 200:
{
  "user_id": "uuid",
  "primary_job": {
    "job_key": "blacksmith",
    "level": 15,
    "display_name": "Master Blacksmith"
  },
  "jobs": [
    {
      "job_key": "blacksmith",
      "level": 15,
      "current_xp": 25430,
      "xp_to_next_level": 4123,
      "max_level": 20
    },
    {
      "job_key": "ranger",
      "level": 8,
      "current_xp": 1890,
      "xp_to_next_level": 631,
      "max_level": 20
    }
  ]
}

---

POST /jobs/award-xp
Authorization: X-API-Key: <api_key>
Content-Type: application/json

{
  "user_id": "uuid",
  "job_key": "blacksmith",
  "xp_amount": 150,
  "source": "upgrade",
  "metadata": { "recipe_quality": "rare" }
}

Response 200:
{
  "job_key": "blacksmith",
  "xp_gained": 150,
  "new_xp": 25580,
  "new_level": 15,
  "leveled_up": false
}
```

### Internal Events
- `JobXPGained { user_id, job_key, xp_amount, new_level, leveled_up }`
- `JobLevelUp { user_id, job_key, old_level, new_level, bonuses_unlocked[] }`

---

## 12. XP Attribution Rules

> **Note**: All XP values are base amounts before the `jobs_xp_boost` multiplier.

### Blacksmith XP
Awarded on successful **Upgrade**, **Craft**, or **Disassemble**:

| Recipe Quality | XP Awarded |
|----------------|------------|
| Common | 10 |
| Uncommon | 25 |
| Rare | 50 |
| Epic | 100 |
| Legendary | 200 |

**Formula**: `base_xp × quantity × boost_multiplier`

### Explorer XP
Awarded on **Search** command:

| Action | XP Awarded |
|--------|------------|
| Search (any result) | 5 |
| Bonus loot found | +10 |

### Merchant XP
Awarded on **Buy** or **Sell**:

| Transaction | XP Awarded |
|-------------|------------|
| Per item traded | 2 |
| Bulk trade (10+) | +5 bonus |

### Gambler XP
Awarded on **Gamble Join/Start**:

| Action | XP Awarded |
|--------|------------|
| Per lootbox wagered | 20 |
| Winning a gamble | +50 bonus |

### Farmer XP
Awarded on **Farm** actions:

| Action | XP Awarded |
|--------|------------|
| Plant seeds | 5 |
| Successful harvest | 25 |
| Rare crop harvested | +50 bonus |

### Scholar XP
Awarded on **Community Progression** contributions:

| Action | XP Awarded |
|--------|------------|
| Vote cast | 10 |
| Engagement milestone reached | 50 |
| Progression node unlocked (participant) | 100 |

---

## 13. Bonus System

### Explorer Bonus Example
```go
func (s *SearchService) ApplyExplorerBonus(ctx context.Context, userID string, baseReward int) (int, bool) {
    level, err := s.jobService.GetJobLevel(ctx, userID, "explorer")
    if err != nil || level == 0 {
        return baseReward, false
    }
    
    // 25% base chance, +1% per level (max 50% at level 25)
    bonusChance := min(0.25 + float64(level)*0.01, 0.50)
    
    if rand.Float64() < bonusChance {
        // Bonus amount scales with level: 10% + 2% per level
        bonusMultiplier := 1.10 + float64(level)*0.02
        bonusAmount := int(float64(baseReward) * (bonusMultiplier - 1))
        return baseReward + bonusAmount, true
    }
    
    return baseReward, false
}
```

### Gambler Bonus Example
```go
func (s *GambleService) ApplyGamblerBonus(ctx context.Context, userID string, basePrize int) int {
    level, err := s.jobService.GetJobLevel(ctx, userID, "gambler")
    if err != nil || level == 0 {
        return basePrize
    }
    
    // Small prize increase: 1% per level (max 25% at level 25)
    bonusPercent := min(float64(level)*0.01, 0.25)
    bonusAmount := int(float64(basePrize) * bonusPercent)
    return basePrize + bonusAmount
}
```

### Blacksmith Bonus Example
Recipe level requirement check:
```go
func (s *CraftingService) CanUseRecipe(ctx context.Context, userID string, recipe *Recipe) (bool, error) {
    if recipe.RequiredBlacksmithLevel == 0 {
        return true, nil
    }
    
    level, err := s.jobService.GetJobLevel(ctx, userID, "blacksmith")
    if err != nil {
        return false, err
    }
    
    return level >= recipe.RequiredBlacksmithLevel, nil
}
```

---

## 14. Job Identity System

A user's **primary job** is their highest-level job. This determines their display title.

### Title Format
```
{Rank} {JobName}
```

| Level Range | Rank |
|-------------|------|
| 1-4 | Apprentice |
| 5-9 | Journeyman |
| 10-14 | Expert |
| 15-19 | Master |
| 20-29 | Grandmaster |
| 30+ | Legendary |

**Example**: Level 15 Blacksmith → "Master Blacksmith"

### Tie-breaking
If multiple jobs share the highest level, use:
1. Most recently leveled
2. Alphabetical order

---

## 15. Business Rules

1. XP can only be gained for **unlocked jobs** (progression check).
2. XP gain requires the **jobs_xp** node to be unlocked.
3. Level cannot exceed cap set by `jobs_xp` unlock level.
4. XP at max level is still tracked (for future cap increases).
5. Job bonuses only apply if the associated feature is unlocked.
6. Daily XP caps (optional): 1000 XP per job per day to prevent exploitation.
7. All XP gains are atomic transactions.

---

## 16. Migrations

### Migration File
**File**: `migrations/YYYYMMDDHHMMSS_create_job_tables.sql`

```sql
-- +goose Up

-- Job definitions
CREATE TABLE jobs (
    id SERIAL PRIMARY KEY,
    job_key TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    associated_features TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed initial jobs
INSERT INTO jobs (job_key, display_name, description, associated_features) VALUES
    ('blacksmith', 'Blacksmith', 'Masters of crafting, upgrades, and disassembly', ARRAY['upgrade', 'craft', 'disassemble']),
    ('explorer', 'Explorer', 'Scouts who find extra rewards', ARRAY['search']),
    ('merchant', 'Merchant', 'Traders who get better deals', ARRAY['buy', 'sell']),
    ('gambler', 'Gambler', 'High rollers who win bigger prizes', ARRAY['gamble']),
    ('farmer', 'Farmer', 'Patient cultivators of valuable crops', ARRAY['farm']),
    ('scholar', 'Scholar', 'Contributors to community progress', ARRAY['progression']);

-- User job progress
CREATE TABLE user_jobs (
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    job_id INT NOT NULL REFERENCES jobs(id),
    current_xp BIGINT NOT NULL DEFAULT 0,
    current_level INT NOT NULL DEFAULT 0,
    xp_gained_today BIGINT DEFAULT 0,
    last_xp_gain TIMESTAMPTZ,
    PRIMARY KEY (user_id, job_id)
);

CREATE INDEX idx_user_jobs_user ON user_jobs(user_id);
CREATE INDEX idx_user_jobs_level ON user_jobs(current_level DESC);

-- XP event log
CREATE TABLE job_xp_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(user_id),
    job_id INT NOT NULL REFERENCES jobs(id),
    xp_amount INT NOT NULL,
    source_type TEXT NOT NULL,
    source_metadata JSONB,
    recorded_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_job_xp_events_user ON job_xp_events(user_id, recorded_at DESC);

-- Bonus configuration
CREATE TABLE job_level_bonuses (
    id SERIAL PRIMARY KEY,
    job_id INT NOT NULL REFERENCES jobs(id),
    min_level INT NOT NULL,
    bonus_type TEXT NOT NULL,
    bonus_value DECIMAL(10,4) NOT NULL,
    description TEXT,
    UNIQUE (job_id, min_level, bonus_type)
);

-- Seed explorer bonuses
INSERT INTO job_level_bonuses (job_id, min_level, bonus_type, bonus_value, description) VALUES
    ((SELECT id FROM jobs WHERE job_key = 'explorer'), 1, 'bonus_money_chance', 0.25, '25% base chance'),
    ((SELECT id FROM jobs WHERE job_key = 'explorer'), 10, 'bonus_money_chance', 0.35, '35% at level 10'),
    ((SELECT id FROM jobs WHERE job_key = 'explorer'), 20, 'bonus_money_chance', 0.45, '45% at level 20');

-- Seed gambler bonuses
INSERT INTO job_level_bonuses (job_id, min_level, bonus_type, bonus_value, description) VALUES
    ((SELECT id FROM jobs WHERE job_key = 'gambler'), 1, 'prize_increase', 0.01, '1% prize increase per level'),
    ((SELECT id FROM jobs WHERE job_key = 'gambler'), 10, 'prize_increase', 0.10, '10% at level 10'),
    ((SELECT id FROM jobs WHERE job_key = 'gambler'), 25, 'prize_increase', 0.25, '25% max at level 25');

-- +goose Down
DROP TABLE IF EXISTS job_level_bonuses;
DROP TABLE IF EXISTS job_xp_events;
DROP TABLE IF EXISTS user_jobs;
DROP TABLE IF EXISTS jobs;
```

---

## 17. Testing Requirements

### Unit Tests
- XP calculation: `CalculateLevel()`, `GetXPForLevel()`, `GetXPProgress()`
- Level cap enforcement
- Bonus calculation: `GetJobBonus()`, `ApplyBonus()`
- Primary job selection logic

### Integration Tests
- XP attribution from feature usage (upgrade → blacksmith XP)
- Level-up event emission
- Bonus application in search/upgrade
- Progression lock enforcement

### Concurrency Tests
- Simultaneous XP gains for same user/job
- Race condition on level-up

---

## 18. Implementation Strategy

### Task Breakdown
| Task | Size | Description |
|------|------|-------------|
| 1. DB schema | S | Migration with tables and seed data |
| 2. Domain models | S | `Job`, `UserJob`, `JobXPEvent` structs |
| 3. Repository | M | CRUD for user_jobs, job lookup, XP events |
| 4. Service | L | XP award, level calc, bonus lookup, primary job |
| 5. API handlers | S | GET /jobs, GET /jobs/{user_id}, POST /jobs/award-xp |
| 6. Progression integration | M | Add job unlock nodes, check caps |
| 7. Feature integration | L | Wire XP attribution into upgrade, search, etc. |
| 8. Tests | M | Unit, integration, concurrency |

**Estimated Total**: 2-3 weeks (1 developer)

---

## 19. Sequence Diagram

```
User                Bot                JobService           Database
  |                  |                     |                    |
  | /upgrade ...  -->|                     |                    |
  |                  | Upgrade succeeds -->|                    |
  |                  |                     | GetUserJob ------->|
  |                  |                     |<--- current state --|
  |                  |                     | CalcXP(recipe) --->|
  |                  |                     | AwardXP ---------->|
  |                  |                     |<--- new level -----|
  |                  | JobXPGained event ->|                    |
  |<-- "Upgraded! +50 Blacksmith XP"       |                    |
  |                  |                     |                    |
  | /search -------->|                     |                    |
  |                  | Search executes --->|                    |
  |                  |                     | GetExplorerBonus --->|
  |                  |                     |<--- 35% chance ----|
  |                  |                     | (roll succeeds)    |
  |                  |<--- bonus $50 ------|                    |
  |<-- "Found $100 + $50 Explorer bonus!"    |                    |
```

---

## 20. Acceptance Criteria

- **AC1**: XP is correctly awarded when using associated features.
- **AC2**: Level correctly calculated from cumulative XP.
- **AC3**: Level cap enforced based on `jobs_xp` progression unlock level.
- **AC4**: Explorer bonus correctly applied with level-scaling chance/amount.
- **AC5**: Blacksmith level requirement blocks recipes if not met.
- **AC6**: Gambler prize bonus correctly applied on wins.
- **AC7**: XP boost multiplier from progression node applied to all XP gains.
- **AC8**: New users receive starting levels based on boost node level.
- **AC9**: Primary job correctly identified as highest-level job.
- **AC10**: `JobLevelUp` event emitted exactly once on level-up.
- **AC11**: All job operations are locked behind appropriate progression nodes.
- **AC12**: Daily XP caps scale with boost node level.

---

## 21. Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| XP inflation | Medium | High | Daily caps scaling with boost node, market pricing |
| Level cap confusion | Low | Medium | Clear UI messaging about unlock requirements |
| Bonus exploitation | High | Medium | Rate limiting + reactionary shop pricing |
| Performance on leaderboards | Medium | Low | Index on level, cache top rankings |
| Complex integration | Medium | Medium | Progression system gates rollout naturally |

---

## 22. Decision Points

### DP1: XP Daily Cap ✅ DECIDED
**Decision**: Daily cap per job, scaling with `jobs_xp_boost` node level.

| Boost Level | Daily Cap per Job |
|-------------|-------------------|
| 0 (base) | 500 XP |
| 1 | 625 XP |
| 2 | 750 XP |
| 3 | 875 XP |
| 4 | 1000 XP |

**Rationale**: Market pricing will react to item demand, pricing out abuse. Daily caps provide a secondary safeguard.

---

### DP2: XP Event Logging
- **Option A**: Log all XP events (full audit trail)
  - ✅ Debugging, analytics, leaderboards
  - ❌ Storage growth
- **Option B**: Only track current XP (no history)
  - ✅ Minimal storage
  - ❌ No audit capability

**Recommendation**: **Option A** with 30-day retention.

---

### DP3: Initial Job Rollout ✅ DECIDED
**Decision**: Launch all 6 jobs with progression system gating the rollout.

Each job is locked behind its own progression node. The community votes to unlock jobs as they see fit, providing natural staggered rollout without artificial limitations.

**Job Dependencies**:
- Blacksmith: Requires `upgrade` feature unlocked
- Explorer: Requires `search` feature unlocked
- Merchant: Requires `economy` feature unlocked
- Gambler: Requires `gamble` feature unlocked
- Farmer: Requires `farm` feature unlocked (future)
- Scholar: No feature dependency (always available after `jobs_xp`)

---

## 23. Future Enhancements (v2)

- Job prestige system (reset to level 1 for permanent bonuses)
- Job-specific achievements
- Job combination bonuses (e.g., Blacksmith + Scholar = Artificer)
- Job leaderboards
- Daily/weekly job quests for bonus XP
