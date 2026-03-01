# Job Bonuses: Farmer

## Overview

The Farmer job affects **harvest** and **compost** features. XP is earned through harvesting and composting.

## Status: Open

## Continuous Bonuses

| Bonus Type         | Base | Per Level | Max  | Effect                                                                    |
| ------------------ | ---- | --------- | ---- | ------------------------------------------------------------------------- |
| `harvest_yield`    | 0.00 | 0.02      | 0.50 | Multiplies harvest item quantities. At level 10: +20%, level 25: +50% cap |
| `compost_yield`    | 0.00 | 0.01      | 0.25 | Multiplies compost output value. At level 10: +10%, level 25: +25% cap    |
| `compost_speed`    | 0.00 | 0.02      | 0.40 | Reduces composting duration. At level 10: 20% faster, level 20: 40% cap   |
| `spoil_extension`  | 0.0  | 2.0       | 48.0 | Adds hours before harvest spoils. At level 10: +20hrs                     |
| `sludge_extension` | 0.0  | 1.0       | 24.0 | Adds hours before compost becomes sludge. At level 10: +10hrs             |

## Level Gates

None proposed. Harvest and compost are passive features — gating them feels punishing. The progression tree already gates compost access via feature unlock.

## Integration Points

### Harvest (partially wired)

`internal/harvest/rewards.go` `calculateBonuses` already calls:

```go
s.jobSvc.GetJobBonus(ctx, userID, "job_farmer", bonusTypeHarvestYield)
s.jobSvc.GetJobBonus(ctx, userID, "job_farmer", bonusTypeGrowthSpeed)
```

- `harvest_yield` is applied as `yieldMultiplier` on reward quantities — **working once config exists**
- `growth_speed` is queried but the returned `growthMultiplier` is **never applied** to anything — needs wiring or removal
- `spoil_extension` would need code change in `calculateHarvestRewards` to add bonus hours to `spoiledThreshold`

### Compost (not wired)

`internal/compost/engine.go` has natural injection points:

- `CalculateOutput(inputValue, dominantType, isSludge, allItems, multiplier)` — `compost_yield` bonus adds to the `multiplier` param
- `CalculateReadyAt(startedAt, totalItemCount)` — `compost_speed` reduces the computed duration
- `CalculateSludgeAt(readyAt)` — `sludge_extension` adds to `SludgeTimeout`

The compost service needs `jobSvc` as a dependency (currently absent) to query bonuses.

## Depends On

- `job_bonus_architecture.md` — formula engine and `job_bonus_config` table
