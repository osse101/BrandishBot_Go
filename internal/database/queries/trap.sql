-- name: CreateTrap :one
INSERT INTO user_traps (id, setter_id, target_id, quality_level, timeout_seconds, placed_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, setter_id, target_id, quality_level, timeout_seconds, placed_at, triggered_at;

-- name: GetActiveTrap :one
SELECT id, setter_id, target_id, quality_level, timeout_seconds, placed_at, triggered_at
FROM user_traps
WHERE target_id = $1 AND triggered_at IS NULL
LIMIT 1;

-- name: GetActiveTrapForUpdate :one
SELECT id, setter_id, target_id, quality_level, timeout_seconds, placed_at, triggered_at
FROM user_traps
WHERE target_id = $1 AND triggered_at IS NULL
LIMIT 1
FOR UPDATE;

-- name: TriggerTrap :exec
UPDATE user_traps
SET triggered_at = NOW()
WHERE id = $1;

-- name: GetTrapsByUser :many
SELECT id, setter_id, target_id, quality_level, timeout_seconds, placed_at, triggered_at
FROM user_traps
WHERE setter_id = $1
ORDER BY placed_at DESC
LIMIT $2;

-- name: GetTriggeredTrapsForTarget :many
SELECT id, setter_id, target_id, quality_level, timeout_seconds, placed_at, triggered_at
FROM user_traps
WHERE target_id = $1 AND triggered_at IS NOT NULL
ORDER BY triggered_at DESC
LIMIT $2;

-- name: CleanupStaleTraps :exec
DELETE FROM user_traps
WHERE triggered_at IS NULL AND placed_at < NOW() - INTERVAL '1 day' * $1;
