# Feature: Job Service Progression Integration

**Created:** 2026-06-18
**Status:** RESOLVED
**Priority:** Medium
**Labels:** feature, jobs, progression

## Summary

Integrate the Job Service with the Progression System to dynamically scale XP multipliers, daily caps, and unlock levels based on community progression milestones.

## Background

The `internal/job/service.go` file contains several `TODO` comments indicating that job mechanics should be tied to the progression system. Currently, these values are hardcoded defaults (`DefaultXPMultiplier`, `DefaultDailyCap`, `DefaultMaxLevel`).

Integrating these systems will allow the game economy and player progression to evolve based on community achievements (Progression Nodes).

## Related Files

-   `internal/job/service.go`
-   `internal/progression/service.go` (and interface)

## Proposed Enhancements

1.  **XP Multiplier**: Query the `jobs_xp_boost` node level from `ProgressionService` to determine the current global XP multiplier.
2.  **Daily Cap**: Scale the daily XP cap based on the `jobs_xp_boost` node level.
3.  **Max Job Level**: Determine the maximum attainable job level based on the `jobs_xp` node unlock level.

## Implementation Plan

1.  **Define Progression Keys**: Identify or create the specific progression node keys (e.g., `jobs_xp_boost`, `jobs_xp`).
2.  **Implement `getXPMultiplier`**:
    -   Call `progressionSvc.GetNodeLevel("jobs_xp_boost")`.
    -   Map the level to a multiplier (e.g., Level 0 = 1.0x, Level 1 = 1.1x, etc.).
3.  **Implement `getDailyCap`**:
    -   Call `progressionSvc.GetNodeLevel("jobs_xp_boost")` (or a separate node).
    -   Scale the cap (e.g., Cap = Base + (Level * Bonus)).
4.  **Implement `getMaxJobLevel`**:
    -   Call `progressionSvc.GetNodeLevel("jobs_xp")`.
    -   Map level to max level cap (e.g., Level 1 = 10, Level 2 = 20).
5.  **Update `AwardXP`**: Ensure it uses these dynamic values.
6.  **Tests**: Update unit tests to mock `ProgressionService` responses and verify dynamic scaling.

## Success Criteria

-   `AwardXP` uses dynamic values from the progression system.
-   Hardcoded defaults are used only as fallbacks.
-   Unit tests verify correct scaling behavior.
