# Job Bonuses: Merchant

## Overview

The Merchant job affects **buying**, **selling**, **sales events**, and **quests**. XP is earned through transactions and quest completion.

## Status: Resolved

## Continuous Bonuses

| Bonus Type           | Base | Per Level | Max  | Effect                                                      |
| -------------------- | ---- | --------- | ---- | ----------------------------------------------------------- |
| `sell_price_bonus`   | 0.00 | 0.01      | 0.25 | Increases sell price. At level 10: +10%, level 25: +25% cap |
| `buy_discount`       | 0.00 | 0.01      | 0.20 | Reduces buy price. At level 10: -10%, level 20: -20% cap    |
| `quest_reward_bonus` | 0.00 | 0.02      | 0.50 | Increases quest reward value. At level 10: +20%             |

## Level Gates

| Level | Unlock                                          |
| ----- | ----------------------------------------------- |
| 3     | Access to sales events (limited-time discounts) |
| 5     | 2nd active quest slot                           |
| 15    | 3rd active quest slot                           |

## Integration Points

### Buy/Sell (not yet implemented as services)

Buy and sell mechanics would apply `base_value` multiplied by merchant bonus:

```
sellPrice = item.BaseValue * (1.0 + sell_price_bonus)
buyPrice  = item.BaseValue * (1.0 - buy_discount)
```

### Quests

`internal/quest/` service — `quest_reward_bonus` would multiply reward quantities/values on quest completion.

### Sales Events

Sales events would be a merchant-specific feature: periodic discounted items visible only to merchants at level 3+. This requires its own design — out of scope for the bonus system itself.

## Depends On

- `job_bonus_architecture.md` — formula engine and `job_bonus_config` table
