# Search System Refactor

**Status:** ✅ Resolved
**Date:** 2026-02-15
**Component:** `internal/user`

## Overview

The search system (`/search`) has been completely refactored to support diminishing returns, item quality, and job-based bonuses.

## Changes Implemented

### 1. File Splitting
- Logic moved from `internal/user/service.go` to `internal/user/search.go`.
- Helper logic (quality calculation, messaging) moved to `internal/user/search_helpers.go`.
- Constants centralized in `internal/user/constants.go`.

### 2. Diminishing Returns
- Implemented `SearchDailyDiminishmentThreshold` (6 searches/day).
- Implemented `SearchDiminishedSuccessRate` (10%) and `SearchDiminishedXPMultiplier` (10%).

### 3. Quality System
- **Base Quality**: Calculated from daily search count (Uncommon -> Common -> Poor -> Junk -> Cursed).
- **Bonuses**:
    - Critical Success (+2).
    - Streak % 5 (+1).
    - Explorer Job Level / 5 (+1 per 5 levels).
- **Refactored Reward Logic**: `grantSearchReward` now accepts quality level.

### 4. Stats Integration
- Integrated `statsService` to fetch daily search counts and current streak.
- Atomic cooldown enforcement using `cooldownService`.

### 5. Events
- Publishing `search.performed` event with detailed payload (success, critical, quality, xp).

## Verification
- Unit tests in `internal/user/search_test.go` cover 5-case model (Best, Boundary, Error, Concurrent, Nil/Empty).
- Statistical tests verify RNG distribution.
