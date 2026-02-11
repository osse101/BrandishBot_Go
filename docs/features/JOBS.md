# Job & XP System

The Job system provides RPG-style progression for players, allowing them to level up specific roles by performing related actions. Each job unlocks bonuses and perks as players advance.

## Job Types

| Job Key | Role | Primary Actions |
| :--- | :--- | :--- |
| **Blacksmith** | Crafting & Creation | Upgrading items, Disassembling items. |
| **Merchant** | Trading & Economy | Buying items, Selling items. |
| **Farmer** | Resource Gathering | Harvesting crops, Composting (In Dev). |
| **Gambler** | Risk & Reward | Opening lootboxes, Spinning slots, Gambling. |
| **Scholar** | Knowledge & Discovery | Performing searches, Analyzing items. |
| **Explorer** | Adventure & Travel | Leading expeditions, Discovering locations. |

## Core Mechanics

### Experience (XP) Gain
- **Actions**: XP is awarded automatically when players perform relevant actions (e.g., selling items awards Merchant XP).
- **Formula**: `XP = Base XP * Multipliers`
- **Epiphany**: A small chance (`EpiphanyChance`) to earn a massive XP bonus (`EpiphanyMultiplier`) on any action.
- **Daily Cap**: There is a limit to how much XP a player can earn per job per day (`DefaultDailyCap`).
  - **Exception**: Consumables like `Rare Candy` bypass the daily cap.

### Leveling Up
- **Formula**: `XP Required for Level N = BaseXP * (N ^ LevelExponent)`
- **Level Cap**: Default maximum level is 10 (`DefaultMaxLevel`).
  - **Progression**: The `upgrade_job_level_cap` node can increase this limit.
- **Bonuses**: Each level grants bonuses specific to that job (e.g., improved crafting success, better sell prices).

### Progression Integration
The job system is deeply integrated with the [Progression Tree](./PROGRESSION.md).
- **Unlocks**: Jobs must be unlocked via progression nodes (e.g., `job_blacksmith`).
- **Modifiers**: Progression nodes can boost XP gain, daily caps, and level limits.

## API Endpoints

### Get User Jobs
```http
GET /api/v1/jobs/user?user_id=uuid
```
Returns all unlocked jobs for a user, including current level, XP, and progress to next level.

**Response**:
```json
[
  {
    "job_key": "merchant",
    "display_name": "Merchant",
    "level": 5,
    "current_xp": 12500,
    "xp_to_next_level": 2500,
    "max_level": 10
  }
]
```

### Award XP (Admin/Testing)
```http
POST /api/v1/jobs/award-xp
```
**Body**:
```json
{
  "user_id": "uuid",
  "job_key": "blacksmith",
  "amount": 100,
  "source": "manual_award"
}
```

## Implementation Details

- **Service**: `internal/job/service.go`
- **Repository**: `internal/repository/job.go`
- **Database**: `jobs`, `user_jobs`, `job_xp_events`, `job_level_bonuses` tables.
- **Daily Reset**: Handled by [Daily Reset System](./DAILY_RESET.md).
