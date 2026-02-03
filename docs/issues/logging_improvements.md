# Logging Improvements

Based on an analysis of recent logs, the following improvements are recommended to enhance system observability and reduce noise.

## 1. Unified Request Context

**Issue:** Logs often lack user context (e.g., "Feature is locked" or "Request completed"), making it difficult to debug 403 or 500 errors without manually correlating requests.

**Recommendation:** Update middleware to inject user identity into the logger context following authentication.

- **Action:** Implement `logger.WithUser(ctx, userID, username)`.
- **Integration:** Apply this in the Auth middleware.
- **Result:** All subsequent logs (errors, warnings, completion) will automatically include `user_id` and `username`.

## 2. Startup Log Consolidation

**Issue:** Log files are flooded with hundreds of "Updated/Inserted progression node" entries during application startup.

**Recommendation:** Demote detailed per-item logs to `DEBUG` level.

- **Action:** Change `Updated progression node` and similar logs to `DEBUG`.
- **Result:** Keep only the summary log at `INFO` level (e.g., `msg="Progression tree synced" inserted=47 updated=2`).

## 3. Chat Message Event Coalescing

**Issue:** Processing a single chat message generates 6-8 distinct log entries (`HandleIncomingMessage`, `Auto-registering`, `Message processed`, etc.), creating massive noise for high-traffic loops.

**Recommendation:** Consolidate success logs into a single high-value summary.

- **Action:** Log a single `INFO` entry at the end of successful processing: `msg="Message processed"`.
- **Fields:** Include `user_id`, `duration_ms`, `auto_registered` (bool), and `contribution_added` (int).
- **Benefit:** Reduces log volume by approximately 80% during normal operation.

## 4. Structured Error Responses

**Issue:** Errors are often split between a generic "Request completed status=400" log and a separate warning log appearing earlier (or not at all).

**Recommendation:** Capture and log the specific error reason in the final request completion log.

- **Example:** `msg="Request completed" status=400 error="field validation failed: known_platform"`

## 5. Deadletter Visibility

**Issue:** The system logs where the deadletter file is located, but does not alert the main log when an event is actually dropped.

**Recommendation:** Emit a `WARN` or `ERROR` log to the main output whenever an event is dead-lettered.

- **Content:** Include `event_type` and `failure_reason`.

## 6. External Service Latency

**Issue:** Total request duration is logged, but the time spent waiting for dependencies (Streamerbot, Database, Twitch API) is hidden.

**Recommendation:** Add field tracing for critical dependencies.

- **Example:** `msg="External call" service="streamerbot" duration_ms=15`

## 7. Feature Flag Context

**Issue:** Feature lock warnings are generic (`msg="Feature is locked"`).

**Recommendation:** Enrich feature access logs with user context.

- **Example:** `msg="Feature access denied" feature="search" user_id="12345" reason="locked"`
