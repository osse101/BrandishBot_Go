# Job Bonuses: Scholar

## Overview

The Scholar job affects the **progression system** — voting, predictions, and engagement. XP is earned passively through chat engagement (subs, follows, messages).

## Status: Open

## Continuous Bonuses

| Bonus Type             | Base | Per Level | Max  | Effect                                                           |
| ---------------------- | ---- | --------- | ---- | ---------------------------------------------------------------- |
| `engagement_xp_bonus`  | 0.00 | 0.01      | 0.25 | Global XP multiplier for ALL jobs. At level 10: +10%             |
| `vote_weight`          | 0.00 | 0.1       | 2.0  | Extra weight on progression tree votes. At level 10: +1.0 weight |
| `unlock_cost_discount` | 0.00 | 0.01      | 0.20 | Reduces progression node unlock cost. At level 10: -10%          |

## Level Gates

| Level | Unlock                                         |
| ----- | ---------------------------------------------- |
| 3     | Can create predictions / propose vote topics   |
| 10    | Higher voting weight (via `vote_weight` bonus) |

## Integration Points

### Engagement XP (partially wired)

`internal/job/event_handler.go` `HandleEngagement` awards Scholar XP on engagement events. The `engagement_xp_bonus` would be queried by `AwardXP` in `xp.go` when calculating `actualAmount` for any job:

```go
scholarBonus, _ := s.getScholarBonus(ctx, userID)
actualAmount = int(float64(baseAmount) * xpMultiplier * (1.0 + scholarBonus))
```

This makes Scholar the "meta" job — passive XP earns global benefits.

### Voting

`internal/progression/` voting service — `vote_weight` would multiply vote impact when casting a progression tree vote.

### Node Unlocks

`unlock_cost_discount` would reduce the `unlock_cost` on progression nodes when a user initiates an unlock.

## Design Notes

Scholar XP comes from engagement (chat activity). Unlike other jobs where XP is earned by performing the feature directly, Scholar rewards consistent community participation. The global XP bonus is deliberately powerful to reward this passive investment.

## Depends On

- `job_bonus_architecture.md` — formula engine and `job_bonus_config` table
