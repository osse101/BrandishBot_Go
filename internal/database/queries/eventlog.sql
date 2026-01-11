-- name: LogEvent :exec
INSERT INTO events (event_type, user_id, payload, metadata)
VALUES ($1, $2, $3, $4);

-- name: GetEvents :many
SELECT id, event_type, user_id, payload, metadata, created_at
FROM events
WHERE 
    (sqlc.narg('user_id')::text IS NULL OR user_id = sqlc.narg('user_id'))
    AND (sqlc.narg('event_type')::text IS NULL OR event_type = sqlc.narg('event_type'))
    AND (sqlc.narg('since')::timestamptz IS NULL OR created_at >= sqlc.narg('since'))
    AND (sqlc.narg('until')::timestamptz IS NULL OR created_at <= sqlc.narg('until'))
ORDER BY created_at DESC
LIMIT $1;

-- name: GetLogEventsByUser :many
SELECT id, event_type, user_id, payload, metadata, created_at
FROM events
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetLogEventsByType :many
SELECT id, event_type, user_id, payload, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: CleanupOldEvents :execrows
DELETE FROM events
WHERE created_at < NOW() - make_interval(days => @days::int);
