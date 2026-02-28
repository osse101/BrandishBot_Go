# Job Bonuses: Explorer

## Overview

The Explorer job affects **search** and **expeditions**. XP is earned through searching and expedition participation.

## Status: Resolved

## Continuous Bonuses

| Bonus Type                 | Base | Per Level | Max  | Effect                                                                        |
| -------------------------- | ---- | --------- | ---- | ----------------------------------------------------------------------------- |
| `search_crit_chance`       | 0.00 | 0.005     | 0.10 | Added to `SearchCriticalRate` (base 5%). At level 10: +5%, level 20: +10% cap |
| `search_quality_boost`     | 0.00 | 0.01      | 0.25 | Biases quality roll upward in `calculateSearchQuality`. At level 10: +10%     |
| `expedition_outcome_shift` | 0.00 | 0.01      | 0.20 | Shifts expedition outcome weights toward positive results. At level 10: +10%  |

## Level Gates

| Level | Unlock                                         |
| ----- | ---------------------------------------------- |
| 10    | Bonus encounter type in expedition config      |
| 15    | Guaranteed quality boost on first daily search |

## Integration Points

### Search (not wired)

`internal/user/search.go` tunable parameters:

- `SearchCriticalRate = 0.05` — `search_crit_chance` would be added to this in `executeSearch`
- `calculateSearchQuality` — `search_quality_boost` biases the quality roll result
- User service would need `jobSvc` dependency or `GetJobBonus` call added

### Expeditions (partially wired)

`internal/expedition/service.go` already has `JobService` interface for skill checks via `GetUserJobs`.

- `rollOutcome` in `encounters.go` applies `weightMods` to shift outcomes — `expedition_outcome_shift` bonus would feed into this
- `rollEncounter` could add bonus encounter types at higher explorer levels

## Depends On

- `job_bonus_architecture.md` — formula engine and `job_bonus_config` table
