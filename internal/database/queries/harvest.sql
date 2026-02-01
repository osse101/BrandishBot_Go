-- name: GetHarvestState :one
SELECT * FROM harvest_state WHERE user_id = $1::uuid;

-- name: GetHarvestStateWithLock :one
SELECT * FROM harvest_state WHERE user_id = $1::uuid FOR UPDATE;

-- name: CreateHarvestState :one
INSERT INTO harvest_state (user_id, last_harvested_at)
VALUES ($1::uuid, NOW())
RETURNING *;

-- name: UpdateHarvestState :exec
UPDATE harvest_state
SET last_harvested_at = $2, updated_at = NOW()
WHERE user_id = $1::uuid;
