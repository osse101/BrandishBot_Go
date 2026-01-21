# Issue: Job System Maintenance and Scalability

## Description
During an audit of the job system, several "real problems" were identified that affect the long-term functionality and scalability of the job XP system.

### 1. Missing Daily XP Reset
The `XPGainedToday` counter in the `user_jobs` table is never reset by the application. While `JobRepository` has a `ResetDailyJobXP` method, it is not currently called by any scheduled worker or internal logic.
- **Impact**: Users will hit their daily XP cap once and never be able to earn XP again until manual database intervention.
- **Root Cause**: Missing global scheduler job for XP reset.
- **Location**: `internal/job/service.go`, `internal/database/postgres/job.go`.

### 2. Hardcoded Max Level
The maximum job level is currently hardcoded to `10` via `DefaultMaxLevel` in `internal/job/constants.go` and is not currently influenced by the progression system.
- **Impact**: Progression in jobs is artificially capped regardless of community unlocks.
- **Root Cause**: `getMaxJobLevel` in `internal/job/service.go` is a TODO placeholder.
- **Location**: `internal/job/service.go`, `internal/job/constants.go`.

## Proposed Solution (DO NOT IMPLEMENT YET)
- Implement a `DailyXPResetJob` and schedule it in `cmd/app/main.go`.
- Implement per-user reset logic in `AwardXP` as a safety measure.
- Integrate `getMaxJobLevel` with the progression service to derive caps from node levels.
