# Search System

The Search system allows players to scavenge for items and earn experience points. It serves as the primary "active" gameplay loop for gathering resources and is deeply integrated with the [Job System](./JOBS.md) (specifically the Explorer job).

## Core Mechanics

### Basics

- **Command**: `/search`
- **Cooldown**: **30 minutes** per user.
- **Cost**: None (free action).
- **Reward**: Chance to find a **Lootbox (Tier 0)** and gain **Explorer Job XP**.

### Success Rate

- **Base Success Rate**: **80%**
- **Critical Success Rate**: **5%** (Roll ≤ 0.05)
  - **Effect**: Finds **2x** items and applies a **+2 Quality Bonus**.
- **Near Miss Rate**: **5%** (Roll just above success threshold)
  - **Effect**: No item found, but special flavor text is shown.
- **Critical Failure Rate**: **5%** (Roll > 0.95)
  - **Effect**: No item found, humorous failure message.

### Daily Diminishing Returns

To prevent excessive farming while rewarding daily engagement, the system uses a diminishing returns mechanic.

- **Threshold**: **6 searches per day** (rolling 24-hour window).
- **Effect**: After the 6th search in a day:
  - **XP Multiplier**: Drops to **10%** (from 100%).
  - **Message**: Appends `(Exhausted)` to the result.
  - **Success Rate**: Remains **80%** (unchanged).

---

## Item Quality System

Every successful search rolls for item quality. The quality of the found lootbox is determined by a point system based on daily activity, streaks, and job level.

### 1. Base Quality (Daily Decay)

The more you search in a day, the lower the base quality becomes. This encourages "checking in" rather than grinding.

| Daily Search #  | Base Quality | Tier Name |
| :-------------- | :----------- | :-------- |
| **1st**         | **Uncommon** | "Green"   |
| **2nd - 5th**   | **Common**   | "White"   |
| **6th - 9th**   | **Poor**     | "Gray"    |
| **10th - 14th** | **Junk**     | "Trash"   |
| **15th+**       | **Cursed**   | "Cursed"  |

### 2. Quality Bonuses

Points are added to the base quality index to upgrade the result.

| Source               | Bonus                   | Notes                                                  |
| :------------------- | :---------------------- | :----------------------------------------------------- |
| **Critical Success** | **+2 Points**           | Occurs 5% of the time.                                 |
| **Streak Milestone** | **+1 Point**            | Applies if `Streak % 5 == 0` (e.g., Day 5, 10, 15...). |
| **Explorer Job**     | **+1 Point / 5 Levels** | Level 5 = +1, Level 10 = +2, etc.                      |

### 3. Final Quality Calculation

The points shift the quality tier upwards from the base.
_Example: 1st search (Uncommon base) + Critical (+2) + Level 5 Explorer (+1) = **Legendary** result._

---

## Rewards

### Items

- **Primary Reward**: `lootbox_tier0` (Junkbox).
- **Quantity**:
  - Standard: **1**
  - Critical Success: **2**

### Experience (XP)

- **Job**: Explorer
- **Amount**: Base amount defined in `internal/job/constants.go`.
- **Multiplier**:
  - Normal: 1.0x
  - Diminished: 0.1x

---

## Events

The system publishes the following events to the Event Bus:

### `search.performed`

Published whenever a search is attempted (success or fail).

**Payload:**

```json
{
  "user_id": "uuid-string",
  "success": true,
  "is_critical": false,
  "is_near_miss": false,
  "is_critical_fail": false,
  "xp_amount": 15,
  "item_name": "lootbox_tier0",
  "quantity": 1,
  "timestamp": 1234567890
}
```

---

## Configuration

Key constants defined in `internal/domain/constants.go`:

```go
const (
    SearchSuccessRate                = 0.8
    SearchCriticalRate               = 0.05
    SearchDailyDiminishmentThreshold = 6
    SearchCooldownDuration           = 30 * time.Minute
)
```
