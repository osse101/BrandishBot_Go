-- name: RecordEvent :one
INSERT INTO stats_events (user_id, event_type, event_data, created_at)
VALUES ($1, $2, $3, $4)
RETURNING event_id, created_at;

-- name: GetEventsByUser :many
SELECT event_id, user_id, event_type, event_data, created_at
FROM stats_events
WHERE user_id = $1 AND created_at >= $2 AND created_at <= $3
ORDER BY created_at DESC;

-- name: GetUserEventsByType :many
SELECT event_id, user_id, event_type, event_data, created_at
FROM stats_events
WHERE user_id = $1 AND event_type = $2
ORDER BY created_at DESC
LIMIT $3;

-- name: GetEventsByType :many
SELECT event_id, user_id, event_type, event_data, created_at
FROM stats_events
WHERE event_type = $1 AND created_at >= $2 AND created_at <= $3
ORDER BY created_at DESC;

-- name: GetTopUsers :many
SELECT se.user_id, u.username, COUNT(*) as event_count
FROM stats_events se
JOIN users u ON se.user_id = u.user_id
WHERE se.event_type = $1 AND se.created_at >= $2 AND se.created_at <= $3
GROUP BY se.user_id, u.username
ORDER BY event_count DESC
LIMIT $4;

-- name: GetEventCounts :many
SELECT event_type, COUNT(*) as count
FROM stats_events
WHERE created_at >= $1 AND created_at <= $2
GROUP BY event_type;

-- name: GetUserEventCounts :many
SELECT event_type, COUNT(*) as count
FROM stats_events
WHERE user_id = $1 AND created_at >= $2 AND created_at <= $3
GROUP BY event_type;

-- name: GetTotalEventCount :one
SELECT COUNT(*)
FROM stats_events
WHERE created_at >= $1 AND created_at <= $2;
