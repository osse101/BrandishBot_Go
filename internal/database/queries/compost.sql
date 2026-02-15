-- Compost Bin Queries

-- name: GetCompostBin :one
SELECT * FROM compost_bins WHERE user_id = $1;

-- name: GetCompostBinForUpdate :one
SELECT * FROM compost_bins WHERE user_id = $1 FOR UPDATE;

-- name: CreateCompostBin :one
INSERT INTO compost_bins (user_id) VALUES ($1) RETURNING *;

-- name: UpdateCompostBin :exec
UPDATE compost_bins SET
    status = $2,
    items = $3,
    item_count = $4,
    started_at = $5,
    ready_at = $6,
    sludge_at = $7,
    input_value = $8,
    dominant_type = $9,
    updated_at = NOW()
WHERE user_id = $1;

-- name: ResetCompostBin :exec
UPDATE compost_bins SET
    status = 'idle',
    items = '[]'::jsonb,
    item_count = 0,
    started_at = NULL,
    ready_at = NULL,
    sludge_at = NULL,
    input_value = 0,
    dominant_type = '',
    updated_at = NOW()
WHERE user_id = $1;
