# Engagement Velocity Tracking

**Status:** Proposed  
**Priority:** Low  
**Effort:** Medium (2-3 days)  
**Type:** Enhancement

## Problem

Currently, the progression system tracks total contribution points but provides no visibility into:
- How fast the community is progressing
- When features might unlock
- Whether engagement is increasing or decreasing over time

This makes it difficult for users to understand progression momentum and for admins to gauge system health.

## Proposed Solution

Add engagement velocity tracking to provide time-based metrics and unlock predictions.

### Core Features

#### 1. Velocity Calculation
Track contribution points over configurable time windows (7d, 14d, 30d) to calculate points/day velocity.

```go
func GetEngagementVelocity(ctx context.Context, days int) (*VelocityMetrics, error)
```

Returns:
- `PointsPerDay` - Average daily contribution rate
- `Trend` - "increasing", "stable", or "decreasing"
- `SampleSize` - Number of data points used

#### 2. Unlock Time Estimation
Predict when a node will unlock based on current velocity and required points.

```go
func EstimateUnlockTime(ctx context.Context, nodeKey string) (*UnlockEstimate, error)
```

Returns:
- `EstimatedDays` - Days until unlock at current velocity
- `ConfidenceLevel` - "high", "medium", "low" based on velocity stability
- `RequiredPoints` - Points still needed
- `CurrentVelocity` - Points/day average

#### 3. API Endpoints
Add endpoints for frontend/Discord consumption:

```
GET /progression/velocity?days=7
GET /progression/estimate/:nodeKey
```

### Implementation Details

**Database Schema**
```sql
-- Option A: Aggregate daily
CREATE TABLE progression_velocity (
    date DATE PRIMARY KEY,
    total_points INT NOT NULL,
    unique_contributors INT NOT NULL
);

-- Option B: Use existing engagement_metrics with time-based queries
-- (No schema change needed)
```

**Calculation Logic**
- Query `engagement_metrics` grouped by day
- Calculate rolling average
- Detect trend using linear regression or simple comparison
- Estimate unlock time: `days = remaining_points / avg_points_per_day`

**Caching Strategy**
- Cache velocity calculations for 1 hour
- Invalidate on significant engagement spikes (>2x normal)
- Recalculate estimates when voting sessions change

### API Response Examples

**Velocity:**
```json
{
  "points_per_day": 450.5,
  "trend": "increasing",
  "period_days": 7,
  "sample_size": 7,
  "total_points": 3154
}
```

**Estimate:**
```json
{
  "node_key": "feature_economy",
  "estimated_days": 3.2,
  "confidence": "medium",
  "required_points": 1500,
  "current_progress": 50,
  "current_velocity": 450.5,
  "estimated_unlock_date": "2026-01-08T00:00:00Z"
}
```

### Edge Cases

1. **Low Activity Periods** - If velocity < 10 points/day, return "low activity" warning
2. **Insufficient Data** - Require minimum 3 days of data, otherwise return "insufficient data"
3. **No Active Target** - If no voting session active, estimate for most likely next unlock
4. **Multiple Prerequisites** - Factor in prerequisite unlock times recursively

### Testing Strategy

- Unit tests for velocity calculation with time-series data
- Test estimate accuracy with mocked progression scenarios
- Integration test with real metrics over 7+ days
- Edge case validation (zero activity, spikes, etc.)

### Success Metrics

- Estimate accuracy within ±20% for nodes with 7+ days of stable data
- API response time < 100ms (with caching)
- User engagement with estimates (track Discord command usage)

## Alternatives Considered

1. **Simple Linear Projection** - Too simplistic, doesn't account for velocity changes
2. **ML-Based Prediction** - Overkill for current scale, revisit at 1000+ daily users
3. **Manual Admin Estimates** - Not scalable, inconsistent

## Dependencies

- Requires completed progression tree v2.0 (prerequisites, dynamic costs) ✅
- Needs stable engagement metric collection (already in place) ✅

## Follow-up Work

- Add velocity graphs to admin dashboard
- Discord command: `/progression estimate <feature>`
- Weekly velocity reports via Discord notifications
- Adjust contribution weights based on velocity analysis

## References

- Related: `internal/progression/service.go` - Core progression logic
- Related: `docs/architecture/progression-system.md` - System overview
- Data: `engagement_metrics` table - Raw contribution data
