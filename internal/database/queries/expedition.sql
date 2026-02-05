-- name: CreateExpedition :exec
INSERT INTO expeditions (id, initiator_id, expedition_type, state, created_at, join_deadline, completion_deadline, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetExpedition :one
SELECT id, initiator_id, expedition_type, state, created_at, join_deadline, completion_deadline, completed_at, metadata
FROM expeditions
WHERE id = $1;

-- name: GetExpeditionParticipants :many
SELECT ep.expedition_id, ep.user_id, ep.joined_at, ep.rewards, ep.username, ep.is_leader, ep.job_levels, ep.final_money, ep.final_xp, ep.final_items, u.username as u_username
FROM expedition_participants ep
JOIN users u ON ep.user_id = u.user_id
WHERE ep.expedition_id = $1;

-- name: AddExpeditionParticipant :exec
INSERT INTO expedition_participants (expedition_id, user_id, joined_at, username)
VALUES ($1, $2, $3, $4);

-- name: UpdateExpeditionState :exec
UPDATE expeditions
SET state = $1
WHERE id = $2;

-- name: UpdateExpeditionStateIfMatches :execresult
UPDATE expeditions
SET state = $1
WHERE id = $2 AND state = $3;

-- name: GetActiveExpedition :one
SELECT id, initiator_id, expedition_type, state, created_at, join_deadline, completion_deadline, completed_at, metadata
FROM expeditions
WHERE state IN ('Recruiting', 'InProgress')
ORDER BY created_at DESC
LIMIT 1;

-- name: CompleteExpedition :exec
UPDATE expeditions
SET state = 'Completed', completed_at = now()
WHERE id = $1;

-- name: SaveExpeditionParticipantRewards :exec
UPDATE expedition_participants
SET rewards = $3
WHERE expedition_id = $1 AND user_id = $2;

-- name: UpdateExpeditionParticipantResults :exec
UPDATE expedition_participants
SET is_leader = $3, job_levels = $4, final_money = $5, final_xp = $6, final_items = $7
WHERE expedition_id = $1 AND user_id = $2;

-- name: GetLastCompletedExpedition :one
SELECT id, initiator_id, expedition_type, state, created_at, join_deadline, completion_deadline, completed_at, metadata
FROM expeditions
WHERE state = 'Completed'
ORDER BY completed_at DESC
LIMIT 1;

-- name: SaveExpeditionJournalEntry :exec
INSERT INTO expedition_journal_entries (expedition_id, turn_number, encounter_type, outcome, skill_checked, skill_passed, primary_member, narrative, fatigue, purse)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetExpeditionJournalEntries :many
SELECT id, expedition_id, turn_number, encounter_type, outcome, skill_checked, skill_passed, primary_member, narrative, fatigue, purse, created_at
FROM expedition_journal_entries
WHERE expedition_id = $1
ORDER BY turn_number ASC;
