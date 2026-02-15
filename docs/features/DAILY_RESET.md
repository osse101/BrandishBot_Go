# Daily Reset System

The Daily Reset system manages the recurring daily tasks required for the game economy and progression, ensuring consistent player limits and opportunities.

## Core Mechanics

### Schedule
- **Time**: The reset occurs daily at **00:00 UTC+7** (Indochina Time).
- **Grace Period**: If the system is down at the scheduled time, the reset will run immediately upon startup if the last recorded reset was the previous day.

### Reset Operations

#### 1. Job XP Cap Reset
- **Purpose**: Resets the daily XP gain limit for all user jobs.
- **Limit**: By default, players can earn a maximum amount of XP per job per day (`DefaultDailyCap`).
- **Reset**: Sets `xp_gained_today` to 0 for all `user_jobs` records.
- **Persistence**: Tracks the reset timestamp and number of affected records in the `daily_reset_state` table.

## Implementation Details

### Worker (`internal/worker/daily_reset_worker.go`)
- **Scheduling**: Calculates the duration until the next 00:00 UTC+7.
- **Execution**: Runs the reset logic in a separate goroutine.
- **Events**: Publishes `daily_reset_completed` event upon success.
- **Retry**: Includes jitter protection and retry logic via `ResilientPublisher`.

### Job Service (`internal/job/service.go`)
- **Logic**: `ResetDailyJobXP` executes the database update.
- **State**: `UpdateDailyResetTime` records the successful reset.
- **Cache**: Updates in-memory cache to reflect the new reset time.

## Admin API Endpoints

### Manual Reset
```http
POST /api/v1/admin/jobs/reset-daily-xp
```
Triggers an immediate daily reset. Useful for testing or correcting missed scheduled resets.

### Reset Status
```http
GET /api/v1/admin/jobs/reset-status
```
Returns the status of the last reset and the scheduled time for the next reset.

**Response**:
```json
{
  "last_reset_time": "2023-10-27T17:00:00Z",
  "next_reset_time": "2023-10-28T17:00:00Z",
  "records_affected": 150
}
```
