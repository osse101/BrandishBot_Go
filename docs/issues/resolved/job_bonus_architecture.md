# Job Bonus Architecture

## Overview

The Job Bonus system provides layered rewards for progressing in different occupations. Instead of hardcoded values, this system uses formula-based modifiers and level gates to ensure scalability and balance.

## Reward Categories

1.  **Primary Bonuses (Continuous)**: Multipliers that scale directly with level (e.g., +2% harvest yield per Farmer level).
2.  **Level Gates**: Functionalities or items unlocked at specific milestones (e.g., "Disassemble" unlocked at Blacksmith level 5).
3.  **Upgrade Nodes**: Existing progression tree enhancements that interact with jobs.

## Formula Engine

Primary bonuses are calculated using simple math expressions stored in the database.

### Database Schema

```sql
CREATE TABLE job_bonus_config (
    id SERIAL PRIMARY KEY,
    job_key VARCHAR(50) NOT NULL,
    bonus_type VARCHAR(50) NOT NULL,
    formula VARCHAR(255) NOT NULL, -- e.g. "1.0 + (level * 0.02)"
    max_value NUMERIC(10, 4),
    description TEXT
);
```

### Implementation Pattern

1.  Call `GetJobBonus(ctx, userID, jobKey, bonusType)` at the point where the tunable parameter is used.
2.  Apply the returned value as an additive multiplier (e.g., `1.0 + bonus`).
3.  Fall back to 0.0 on error (no bonus, no crash).

Example (already exists in harvest):

```go
if yieldBonus, err := s.jobSvc.GetJobBonus(ctx, userID, "job_farmer", "harvest_yield"); err == nil {
    yieldMultiplier += yieldBonus
}
```

## Level Gates

Level gates are handled via:

- **Service Checks**: `if currentLevel < requiredLevel { return ErrFeatureLocked }`
- **Static Data**: `required_job_level` column in `crafting_recipes` (already implemented).

## Job-Specific Issues

Detailed implementation plans for each job are documented in separate issues:

- `job_bonus_farmer.md`
- `job_bonus_explorer.md`
- `job_bonus_blacksmith.md`
- `job_bonus_merchant.md`
- `job_bonus_gambler.md`
- `job_bonus_scholar.md`
