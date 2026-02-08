# Weekly Quests & Weekly Sales Feature

## Overview

The Weekly Quests & Weekly Sales system provides players with rotating weekly challenges and dynamic item discounts to encourage regular engagement and varied gameplay.

**Key Components:**
- **Weekly Quests**: 3 rotating challenges that reset every Monday with diverse objectives (buy items, sell items, earn money, craft recipes, perform searches)
- **Weekly Sales**: Automatic item category discounts that rotate daily
- **Progression Gating**: Feature locked behind `feature_weekly_quests` progression node
- **Reward System**: Money and Merchant XP earned by claiming completed quests

---

## Weekly Quests System

### How Quests Work

#### Quest Selection
- **Every Monday at 00:00 UTC**: The system selects 3 random quests from the quest pool
- **Deterministic**: Same quests for all players that week (seeded by week number)
- **Rotates weekly**: New quests appear every Monday, old progress is cleared

#### Quest Types

##### 1. Buy Items (`buy_items`)
**Objective**: Purchase X items of a specific category

**Example**: "Buy 3 Weapons"
- Triggers when player uses `/buy` or equivalent economy endpoint
- Tracks quantity purchased
- Auto-completes when threshold reached

**Configuration**:
```json
{
  "quest_key": "buy_weapon_items",
  "quest_type": "buy_items",
  "description": "Buy {requirement} Weapons",
  "target_category": "Weapon",
  "base_requirement": 3,
  "base_reward_money": 800,
  "base_reward_xp": 150
}
```

##### 2. Sell Items (`sell_items`)
**Objective**: Sell X items (optionally of specific category)

**Example**: "Sell 5 Weapons"
- Triggers when player uses `/sell` or equivalent economy endpoint
- Tracks quantity sold
- Can be category-specific or accept any items

**Configuration**:
```json
{
  "quest_key": "sell_weapon_items",
  "quest_type": "sell_items",
  "description": "Sell {requirement} Weapons",
  "target_category": "Weapon",
  "base_requirement": 5,
  "base_reward_money": 1250,
  "base_reward_xp": 250
}
```

##### 3. Earn Money (`earn_money`)
**Objective**: Earn X money from selling items

**Example**: "Earn 5000 money from selling items"
- Triggers when player sells items
- Tracks total money earned
- Ignores item category

**Configuration**:
```json
{
  "quest_key": "earn_from_sales_medium",
  "quest_type": "earn_money",
  "description": "Earn {requirement} money from selling items",
  "base_requirement": 5000,
  "base_reward_money": 1500,
  "base_reward_xp": 300
}
```

##### 4. Craft Recipe (`craft_recipe`)
**Objective**: Perform a specific crafting recipe X times

**Example**: "Perform Upgrade Mine recipe 3 times"
- Triggers when player performs recipe (upgrade/disassemble)
- Tracks quantity of recipe completions
- Specific to recipe identifier

**Configuration**:
```json
{
  "quest_key": "upgrade_mine_x3",
  "quest_type": "craft_recipe",
  "description": "Perform crafting recipe: Upgrade Mine {requirement} times",
  "target_recipe_key": "upgrade_mine",
  "base_requirement": 3,
  "base_reward_money": 1000,
  "base_reward_xp": 200
}
```

##### 5. Perform Searches (`perform_searches`)
**Objective**: Perform X searches

**Example**: "Perform 10 searches this week"
- Triggers when player uses search functionality
- Tracks search count
- Simple counter-based progress

**Configuration**:
```json
{
  "quest_key": "search_items",
  "quest_type": "perform_searches",
  "description": "Perform {requirement} searches this week",
  "base_requirement": 10,
  "base_reward_money": 500,
  "base_reward_xp": 100
}
```

### Quest Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│ QUEST LIFECYCLE (Weekly)                                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│ Monday 00:00 UTC                                                 │
│ ├─ Old quests marked INACTIVE                                    │
│ ├─ Old progress deleted                                          │
│ └─ 3 new quests generated (seeded by week #)                    │
│                                                                   │
│ Mon-Sun 00:00                                                    │
│ ├─ Player actions trigger progress tracking:                    │
│ │  ├─ Buy item → OnItemBought()                                 │
│ │  ├─ Sell item → OnItemSold()                                  │
│ │  ├─ Craft recipe → OnRecipeCrafted()                          │
│ │  └─ Search → OnSearch()                                       │
│ ├─ Progress updates in database                                 │
│ └─ Quest auto-completes when threshold reached                  │
│                                                                   │
│ Anytime During Week                                              │
│ ├─ Player views progress via `/api/v1/quests/progress`          │
│ ├─ Player claims completed quest via `/api/v1/quests/claim`     │
│ │  ├─ Money awarded immediately                                 │
│ │  └─ Merchant XP awarded asynchronously                        │
│ └─ Discord bot notifies on completion                           │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Quest States

1. **ACTIVE** - Quest is current, players can make progress
2. **COMPLETED** - Player reached the threshold, awaiting claim
3. **CLAIMED** - Player claimed reward, progress archived
4. **INACTIVE** - Previous week's quest, progress deleted on reset

### Rewards

When a player claims a completed quest:

| Reward Type | Details |
|---|---|
| **Money** | Awarded immediately to inventory |
| **Merchant XP** | Awarded to Merchant job (async) |
| **Notifications** | Discord embed or SSE event (optional) |

**Base Rewards**:
- Common quests: 500-750 money, 100-150 XP
- Medium quests: 1000-1500 money, 200-300 XP
- Challenging quests: 2000-3000 money, 400-600 XP

---

## Weekly Sales System

### How Sales Work

#### Sale Rotation
- **Every 7 Days**: Sales rotate to next category in schedule
- **Day-of-Week Based**: Sales apply automatically based on current week offset
- **No Configuration Needed**: Applies automatically on buy price calculation

#### Sales Schedule

Default schedule (configurable in `configs/economy/weekly_sales.json`):

| Week Offset | Category | Discount | Active Days |
|---|---|---|---|
| 0 | Weapon | 25% | Mon-Sun |
| 1 | Armor | 20% | Mon-Sun |
| 2 | Consumable | 30% | Mon-Sun |
| 3 | Accessory | 15% | Mon-Sun |

Then repeats on Week 4 (back to Weapon, etc.)

#### Discount Calculation

```
Final Price = Base Price × (1 - Discount%)

Example:
- Weapon Base Price: 1000 money
- Week 0 Discount: 25%
- Final Price: 1000 × (1 - 0.25) = 750 money
```

### Implementation Details

**Applied At**: `Economy.GetBuyablePrices()` call
**Scope**: Buy prices only (not sell prices)
**Scope**: Category matches item's first type/category
**Fallback**: If category is null (all items), discount applies to everything

---

## API Endpoints

### Get Active Quests
```http
GET /api/v1/quests/active
```

**Response**:
```json
[
  {
    "quest_id": 1,
    "quest_key": "buy_weapon_items",
    "quest_type": "buy_items",
    "description": "Buy 3 Weapons",
    "target_category": "Weapon",
    "base_requirement": 3,
    "base_reward_money": 800,
    "base_reward_xp": 150,
    "active": true,
    "week_number": 5,
    "year": 2026,
    "created_at": "2026-02-03T00:00:00Z",
    "updated_at": "2026-02-03T00:00:00Z"
  }
]
```

### Get User Quest Progress
```http
GET /api/v1/quests/progress?user_id={uuid}
```

**Response**:
```json
[
  {
    "user_id": "abc-123",
    "quest_id": 1,
    "progress_current": 2,
    "progress_required": 3,
    "reward_money": 800,
    "reward_xp": 150,
    "started_at": "2026-02-03T00:00:00Z",
    "completed_at": "2026-02-05T14:30:00Z",
    "claimed_at": null,
    "quest_key": "buy_weapon_items",
    "quest_type": "buy_items",
    "description": "Buy 3 Weapons",
    "target_category": "Weapon"
  }
]
```

### Claim Quest Reward
```http
POST /api/v1/quests/claim
Content-Type: application/json

{
  "user_id": "abc-123",
  "quest_id": 1
}
```

**Response**:
```json
{
  "money_earned": 800,
  "xp_earned": 150,
  "message": "Quest reward claimed successfully"
}
```

---

## Configuration

### Quest Pool (`configs/quests/weekly_quest_pool.json`)

```json
{
  "version": "1.0",
  "quest_pool": [
    {
      "quest_key": "unique_identifier",
      "quest_type": "buy_items|sell_items|earn_money|craft_recipe|perform_searches",
      "description": "User-facing description with {requirement} placeholder",
      "target_category": "Weapon|Armor|Consumable|Accessory|null",
      "target_recipe_key": "recipe_key_here",
      "base_requirement": 5,
      "base_reward_money": 500,
      "base_reward_xp": 100
    }
  ]
}
```

**Required Fields**:
- `quest_key`: Unique identifier (no spaces, snake_case)
- `quest_type`: One of the 5 types listed above
- `description`: Shown to players, use `{requirement}` for placeholder
- `base_requirement`: Threshold to complete
- `base_reward_money`: Money awarded on claim
- `base_reward_xp`: Merchant XP awarded on claim

**Optional Fields**:
- `target_category`: For buy/sell items quests
- `target_recipe_key`: For craft_recipe quests

### Weekly Sales (`configs/economy/weekly_sales.json`)

```json
{
  "version": "1.0",
  "sales_schedule": [
    {
      "week_offset": 0,
      "target_category": "Weapon|Armor|Consumable|null",
      "discount_percent": 25,
      "description": "User-facing description of sale"
    }
  ]
}
```

**Fields**:
- `week_offset`: Position in rotation (0-indexed, repeats after schedule length)
- `target_category`: Item category to discount (null = all items)
- `discount_percent`: Percentage discount (0-100)
- `description`: Shown to players

---

## Discord Commands

### /quests
View current week's active quests and personal progress

```
/quests

📜 Weekly Quests
Complete rotating challenges each week for money and Merchant XP!

Buy 3 Weapons
🔄 In Progress: █████░░░░ 2/3
Reward: 800 money + 150 Merchant XP

Sell 5 Weapons
✅ Ready to Claim!: ██████████ 5/5
Reward: 1250 money + 250 Merchant XP

Earn 5000 money from selling items
🔄 In Progress: ███░░░░░░░ 1500/5000
Reward: 1500 money + 300 Merchant XP
```

### /claimquest
Claim reward from a completed quest

```
/claimquest quest_id:1

🎉 Quest reward claimed!
💰 Earned: 800 money
⭐ Merchant XP: 150
```

---

## Integration Points

### Economy Service
**Files**: `internal/economy/service.go`

When a player buys or sells items, the economy service automatically tracks progress:

```go
// Buy tracking
if s.questService != nil {
    itemCategory := getItemCategory(item)
    s.questService.OnItemBought(ctx, user.ID, itemCategory, quantity)
}

// Sell tracking
if s.questService != nil {
    s.questService.OnItemSold(ctx, user.ID, itemCategory, quantity, moneyEarned)
}
```

### Crafting Service
**Files**: `internal/crafting/service.go`

When a player performs crafting (upgrade/disassemble):

```go
// After successful craft
s.questService.OnRecipeCrafted(ctx, userID, recipe.Key, quantity)
```

### Search Handler
**Files**: `internal/handler/user.go`

When a player performs a search:

```go
// After successful search
questService.OnSearch(ctx, user.ID)
```

### Weekly Reset Worker
**Files**: `internal/worker/weekly_reset_worker.go`

Runs every Monday at 00:00 UTC:
1. Deactivates all active quests
2. Deletes progress for inactive quests
3. Generates 3 new random quests (deterministic seed)
4. Publishes reset event

---

## Feature Flag & Progression

### Unlock Requirements

Quest feature is locked behind progression node:
- **Node Key**: `feature_weekly_quests`
- **Node Type**: Feature
- **Tier**: 2 (Early unlock)
- **Prerequisites**: `feature_economy`

### Access Control

If feature is locked:
- API endpoints return 403 Forbidden
- Error message explains required nodes to unlock
- Discord commands not available

---

## Event System Integration

### Published Events

| Event Type | When | Payload |
|---|---|---|
| `quest.weekly_reset` | Monday 00:00 UTC | `reset_time`, `week_number`, `year`, `quests_generated` |
| `quest.progress_updated` | During progress tracking | `user_id`, `quest_id`, `progress` |
| `quest.completed` | Auto-complete threshold | `user_id`, `quest_id`, `quest_key` |
| `quest.claimed` | Reward claimed | `user_id`, `quest_id`, `reward_money`, `reward_xp` |

### Event Subscribers

- **SSE Hub**: Broadcasts quest events to connected clients
- **Discord Bot**: Notifies player when quest completes
- **Streamer.bot**: Relays events to streaming overlay

---

## Examples & Scenarios

### Scenario 1: Player Completes Buy Quest

```
Monday 00:00 UTC
├─ Quest generated: "Buy 3 Weapons" (Quest ID: 5)
└─ Player progress created: 0/3

Tuesday 10:00
├─ Player buys 1 Sword
├─ OnItemBought("sword", "Weapon", 1) called
├─ Progress updated: 1/3
└─ No notification (not complete)

Wednesday 15:30
├─ Player buys 2 Spears
├─ OnItemBought("spear", "Weapon", 2) called
├─ Progress updated: 3/3
├─ Quest auto-completes
├─ quest.completed event published
└─ Discord bot notifies player

Wednesday 16:00
├─ Player runs /claimquest quest_id:5
├─ Reward claimed:
│  ├─ 800 money added to inventory
│  └─ 150 Merchant XP awarded (async)
├─ quest.claimed event published
└─ Player sees confirmation message
```

### Scenario 2: Weekly Sales Cycle

```
Week 0 (Jan 20-26): Weapons 25% off
├─ Base price: 1000
└─ Discounted: 750

Week 1 (Jan 27-Feb 2): Armor 20% off
├─ Base price: 1000
└─ Discounted: 800

Week 2 (Feb 3-9): Consumables 30% off
├─ Base price: 500
└─ Discounted: 350

Week 3 (Feb 10-16): Accessories 15% off
├─ Base price: 800
└─ Discounted: 680

Week 4 (Feb 17-23): Back to Weapons 25% off
├─ Base price: 1000
└─ Discounted: 750
```

### Scenario 3: Deterministic Quest Selection

```
Week 5, 2026 (Feb 3-9)
├─ Seed: 5 * 100 + 5 = 505
├─ Random with seed 505:
│  ├─ Shuffle quest pool
│  ├─ Pick first 3:
│  │  1. "Buy 3 Weapons"
│  │  2. "Earn 5000 money"
│  │  3. "Perform Upgrade Mine 3 times"
│  └─ Same for all users this week
└─ Week 6, seed changes → different quests

Result: All players see same quests, but rotation is unpredictable
```

---

## Performance Considerations

### Database
- Quest definitions cached in memory at startup
- Progress queries optimized with indexes
- Weekly reset batches delete operations
- Deterministic seeding prevents N+1 queries

### Async Operations
- XP awards run in background goroutine
- Event publishing non-blocking
- SSE broadcasting asynchronous
- Won't delay transaction completion

### Scalability
- Minimal database writes (progress increments only)
- No polling - event-driven architecture
- Single weekly reset operation (non-blocking)
- Fits with existing cooldown/rate-limiting system

---

## Troubleshooting

### Quests Not Showing Up
1. Check feature is unlocked: `/api/v1/progression/tree` → look for `feature_weekly_quests`
2. Verify migration applied: `psql -c "SELECT * FROM quests;"`
3. Check server logs for weekly reset errors
4. Verify current week has quests: `SELECT * FROM quests WHERE active=true;`

### Progress Not Tracking
1. Verify questService is initialized in `cmd/app/main.go`
2. Check economy service has questService dependency
3. Monitor server logs for "Failed to track quest progress" warnings
4. Verify item has valid category/types: `/api/v1/prices`

### Sales Not Applying
1. Check weekly_sales.json is valid JSON
2. Verify item categories match config
3. Use `/api/v1/prices/buy` to see calculated prices
4. Check current week offset: `(weekNum) % salesSchedule.length`

### Weekly Reset Not Firing
1. Check worker logs for "Next weekly reset scheduled"
2. Verify time zone is UTC (server should use UTC)
3. Check database for reset state: `SELECT * FROM weekly_quest_reset_state;`
4. Manually test: temporarily change timer duration for testing

---

## Future Enhancements

Potential expansions to the quest system:

- **Quest Tiers**: Easy/Medium/Hard with scaled rewards
- **Quest Streaks**: Bonuses for consecutive weeks of completion
- **Achievement Tracking**: Milestones for total quests completed
- **Seasonal Quests**: Special quests during events
- **Daily Quests**: Complement weekly system with daily challenges
- **Quest Leaderboards**: Track fastest completion times
- **Quest History**: Archive past quests and rewards
- **Dynamic Scaling**: Adjust requirements based on user progression level

---

## See Also

- [Economy Feature Documentation](./ECONOMY.md)
- [Job/XP System](./JOBS.md)
- [Progression Tree](./PROGRESSION.md)
- [Event System Architecture](../architecture/EVENT_SYSTEM.md)
- [API Endpoints Reference](../API.md)
