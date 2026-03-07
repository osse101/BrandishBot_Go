-- name: CreateDuel :exec
INSERT INTO duels (
    id, challenger_id, opponent_id, state, stakes, created_at, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: GetDuel :one
SELECT * FROM duels
WHERE id = $1;

-- name: GetDuelForUpdate :one
SELECT * FROM duels
WHERE id = $1
FOR UPDATE;

-- name: UpdateDuelState :exec
UPDATE duels
SET state = $2
WHERE id = $1;

-- name: GetPendingDuelsForUser :many
SELECT * FROM duels
WHERE opponent_id = $1 AND state = 'pending' AND expires_at > now();

-- name: AcceptDuel :exec
UPDATE duels
SET state = 'completed',
    started_at = now(),
    completed_at = now(),
    winner_id = $2,
    result_data = $3
WHERE id = $1;

-- name: DeclineDuel :exec
UPDATE duels
SET state = 'declined'
WHERE id = $1 AND state = 'pending';

-- name: ExpireDuels :exec
UPDATE duels
SET state = 'expired'
WHERE state = 'pending' AND expires_at <= now();
