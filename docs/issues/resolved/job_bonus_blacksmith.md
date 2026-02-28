# Job Bonuses: Blacksmith

## Overview

The Blacksmith job affects **crafting**, **upgrading**, and **disassembling**. XP is earned through these actions.

## Status: Resolved

## Continuous Bonuses

| Bonus Type             | Base | Per Level | Max  | Effect                                                              |
| ---------------------- | ---- | --------- | ---- | ------------------------------------------------------------------- |
| `craft_cost_reduction` | 0.00 | 0.01      | 0.25 | Reduces crafting `base_cost`. At level 10: -10%, level 25: -25% cap |
| `disassemble_yield`    | 0.00 | 0.02      | 0.50 | Increases material return on disassemble. At level 10: +20%         |

## Level Gates

The `required_job_level` column on `crafting_recipes` already implements this pattern. Additional gates:

| Level | Unlock                                                           |
| ----- | ---------------------------------------------------------------- |
| 0     | Basic recipes                                                    |
| 5     | Disassemble action                                               |
| 10+   | Higher-tier recipes (defined per recipe in `required_job_level`) |

## Integration Points

### Crafting (level gate exists)

`crafting_recipes.required_job_level` already gates recipes by blacksmith level — this pattern is fully wired.

- `craft_cost_reduction` would apply to `base_cost` when calculating the crafting price
- Crafting service would need to call `GetJobBonus(ctx, userID, "job_blacksmith", "craft_cost_reduction")`

### Disassemble (not implemented yet)

Disassemble as a feature would return materials from items. `disassemble_yield` bonus would increase material return.

## Depends On

- `job_bonus_architecture.md` — formula engine and `job_bonus_config` table
