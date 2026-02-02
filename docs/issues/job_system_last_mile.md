# Issue: Job System Last Mile Steps

The core job system infrastructure is complete, including XP awarding, daily resets, and dynamic level caps. However, several steps remain to stabilize the feature and integrate job levels into the core gameplay loop.

## 1. Remove Obsolete Components

- [x] Remove the `jobs/bonus` command from the Discord client (as the API endpoint is missing and bonuses are not yet ready for display).
- [x] Remove references to `feature_jobs_xp` in the job service. Only individual job nodes (e.g., `job_blacksmith`) should gate XP gain and job visibility.

## 2. Bonus Integration (Gating & Scaling)

Currently, most jobs (except Scholar) do not provide tangible gameplay benefits based on their level.

- [ ] **Blacksmith**: Link `UpgradeItem` success rates or cost reductions to Blacksmith job level.
- [ ] **Explorer**: Link Search "Shine" chance or item rarity weight to Explorer job level.
- [ ] **Merchant**: Integrate Merchant level into buy/sell price calculations (currently solely based on global progression).
- [ ] **Gambler**: Add a small win probability bonus based on Gambler level.

## 3. Level Gating

Implement RPG-style requirements for advanced features:

- [ ] "Requires Blacksmith Level X to craft" for high-tier recipes.
- [ ] "Requires Explorer Level X" for certain search locations (when implemented) -> Usage: `!search item` for the server to select the best location for finding that item type. Defaults to highest level location available.

## 4. Job Identity & UI

- [ ] Integrate "Primary Job" title into the `/profile` command display.
- [ ] Add job level requirements to `/recipes` list display.

## 5. Farmer Job Implementation

- [x] Implement XP awarding in `harvest.Service`.
- [ ] Define benefits for Farmer job level in farming features (e.g. increased yield, faster growth times, etc.).

---

**Status**: In Progress
**Priority**: Medium
**Related**: `docs/issues/progression_nodes/jobs.md`
