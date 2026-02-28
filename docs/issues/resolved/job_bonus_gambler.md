# Job Bonuses: Gambler

## Overview

The Gambler job affects **lootbox gambles** and **duels**. XP is earned through gamble participation and duel participation.

## Status: Resolved

## Continuous Bonuses

| Bonus Type              | Base | Per Level | Max  | Effect                                                                         |
| ----------------------- | ---- | --------- | ---- | ------------------------------------------------------------------------------ |
| `gamble_score_bonus`    | 0.00 | 0.01      | 0.25 | Increases gamble roll score. At level 10: +10%                                 |
| `crit_fail_protection`  | 0.00 | 0.005     | 0.10 | Shrinks the crit fail threshold (base 20%). At level 10: threshold becomes 15% |
| `near_miss_consolation` | 0.00 | 0.01      | 0.20 | Small payout on near-miss results. At level 10: +10% consolation               |

## Level Gates

| Level | Unlock                                   |
| ----- | ---------------------------------------- |
| 3     | Can initiate duels                       |
| 5     | Can bet higher-tier lootboxes in gambles |
| 10    | Double-or-nothing option on gamble wins  |

## Integration Points

### Gamble (not wired)

`internal/gamble/` tunable parameters:

- `CriticalFailThreshold = 0.20` — `crit_fail_protection` reduces this: `threshold = 0.20 - bonus`
- Score calculation in `execute.go` — `gamble_score_bonus` multiplies each participant's score
- `gamble_win_bonus` already exists as a progression feature key
- Near-miss logic in `determineCriticalFailures` / `determineNearMisses`

### Duels (partially implemented)

`internal/duel/service.go` — `Accept` is unimplemented. Level gates would apply when duel execution is built:

- Level 3 check before allowing `Challenge`
- See `docs/issues/duel_system.md` for full duel spec

## Depends On

- `job_bonus_architecture.md` — formula engine and `job_bonus_config` table
- `duel_system.md` — duel implementation (for duel-related gates)
