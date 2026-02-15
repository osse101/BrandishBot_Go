# Issue: Job System Last Mile Steps

The core job system infrastructure is complete, including XP awarding, daily resets, and dynamic level caps. However, several steps remain to stabilize the feature and integrate job levels into the core gameplay loop.

## 1. Remove Obsolete Components

- [x] Remove the `jobs/bonus` command from the Discord client (as the API endpoint is missing and bonuses are not yet ready for display).
- [x] Remove references to `feature_jobs_xp` in the job service. Only individual job nodes (e.g., `job_blacksmith`) should gate XP gain and job visibility.

## 2. Bonus Integration (Gating & Scaling)

Currently, job benefits are integrated primarily through progression system modifiers rather than direct level checks.

- [x] **Blacksmith**: Link `UpgradeItem` success rates or cost reductions to Blacksmith job level. (Implemented via `crafting_success_rate` modifier)
- [ ] **Explorer**: Link Search "Quality" chance or item rarity weight to Explorer job level. (Not implemented - `calculateSearchQuality` logic does not use modifiers)
- [x] **Merchant**: Integrate Merchant level into buy/sell price calculations. (Implemented via `economy_bonus` modifier)
- [x] **Gambler**: Add a small win probability bonus based on Gambler level. (Implemented via `gamble_win_bonus` modifier)

## 3. Level Gating

Implement RPG-style requirements for advanced features:

- [ ] "Requires Blacksmith Level X to craft" for high-tier recipes. (Currently relies on recipe unlocks, not explicit job level checks in code)
- [ ] "Requires Explorer Level X" for certain search locations (when implemented) -> Usage: `!search item` for the server to select the best location for finding that item type. Defaults to highest level location available.

## 4. Job Identity & UI

- [ ] Integrate "Primary Job" title into the `/profile` command display.
- [ ] Add job level requirements to `/recipes` list display.

## 5. Farmer Job Implementation

- [x] Implement XP awarding in `harvest.Service`.
- [ ] Define benefits for Farmer job level in farming features (e.g. increased yield, faster growth times, etc.). Currently only XP is awarded; no benefits are applied.

---

**Status**: In Progress
**Priority**: Medium
**Related**: `docs/issues/progression_nodes/jobs.md`

## Status Update (2026-01-30)

- **Completed**: Bonus integration for Blacksmith, Merchant, and Gambler is done via progression modifiers. Farmer XP awarding is implemented.
- **Pending**: Explorer job integration (Search Quality), Farmer job benefits, explicit Level Gating, and UI updates.

## Status Update (2026-02-06)

- **Explorer Job**: `internal/user/search_helpers.go` confirmed to not use Explorer job level for quality calculations.
- **Farmer Job**: `internal/harvest/service.go` awards XP but does not use job level to modify yield or speed.
- **Workers**: Daily and weekly reset workers are implemented and functioning.
- **Status**: Still In Progress for feature integration.
