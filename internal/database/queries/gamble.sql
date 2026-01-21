-- name: CreateGamble :exec
INSERT INTO gambles (id, initiator_id, state, created_at, join_deadline)
VALUES ($1, $2, $3, $4, $5);

-- name: GetGamble :one
SELECT id, initiator_id, state, created_at, join_deadline
FROM gambles
WHERE id = $1;

-- name: GetGambleParticipants :many
SELECT p.gamble_id, p.user_id, p.lootbox_bets, u.username
FROM gamble_participants p
JOIN users u ON p.user_id = u.user_id
WHERE p.gamble_id = $1;

-- name: JoinGamble :exec
INSERT INTO gamble_participants (gamble_id, user_id, lootbox_bets)
VALUES ($1, $2, $3);

-- name: UpdateGambleState :exec
UPDATE gambles 
SET state = $1 
WHERE id = $2;

-- name: UpdateGambleStateIfMatches :execresult
UPDATE gambles 
SET state = $1 
WHERE id = $2 AND state = $3;

-- name: SaveOpenedItem :exec
INSERT INTO gamble_opened_items (gamble_id, user_id, item_id, quantity, value)
VALUES ($1, $2, $3, $4, $5);

-- name: GetActiveGamble :one
SELECT id, initiator_id, state, created_at, join_deadline
FROM gambles
WHERE state IN ('Joining', 'Opening')
LIMIT 1;
