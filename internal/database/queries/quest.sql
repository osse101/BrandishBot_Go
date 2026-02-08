-- name: GetActiveQuests :many
SELECT * FROM quests
WHERE active = TRUE
ORDER BY quest_id;

-- name: GetActiveQuestsForWeek :many
SELECT * FROM quests
WHERE active = TRUE AND year = $1 AND week_number = $2
ORDER BY quest_id;

-- name: CreateQuest :one
INSERT INTO quests (
    quest_key, quest_type, description, target_category, target_recipe_key,
    base_requirement, base_reward_money, base_reward_xp,
    active, week_number, year
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: DeactivateAllQuests :exec
UPDATE quests SET active = FALSE WHERE active = TRUE;

-- name: GetUserQuestProgress :many
SELECT
    qp.*,
    q.quest_key,
    q.quest_type,
    q.description,
    q.target_category,
    q.target_recipe_key
FROM quest_progress qp
JOIN quests q ON qp.quest_id = q.quest_id
WHERE qp.user_id = $1
ORDER BY qp.started_at DESC;

-- name: GetUserActiveQuestProgress :many
SELECT
    qp.*,
    q.quest_key,
    q.quest_type,
    q.description,
    q.target_category,
    q.target_recipe_key
FROM quest_progress qp
JOIN quests q ON qp.quest_id = q.quest_id
WHERE qp.user_id = $1 AND q.active = TRUE
ORDER BY qp.quest_id;

-- name: CreateQuestProgress :one
INSERT INTO quest_progress (
    user_id, quest_id, progress_current, progress_required,
    reward_money, reward_xp
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: IncrementQuestProgress :exec
UPDATE quest_progress
SET progress_current = progress_current + $3, updated_at = NOW()
WHERE user_id = $1 AND quest_id = $2;

-- name: CreateQuestProgressForUser :one
INSERT INTO quest_progress (
    user_id, quest_id, progress_current, progress_required,
    reward_money, reward_xp
) VALUES ($1, $2, 0, $3, $4, $5)
ON CONFLICT (user_id, quest_id) DO NOTHING
RETURNING *;

-- name: CompleteQuest :exec
UPDATE quest_progress
SET completed_at = NOW(), updated_at = NOW()
WHERE user_id = $1 AND quest_id = $2 AND completed_at IS NULL;

-- name: ClaimQuestReward :exec
UPDATE quest_progress
SET claimed_at = NOW(), updated_at = NOW()
WHERE user_id = $1 AND quest_id = $2 AND claimed_at IS NULL;

-- name: GetUnclaimedCompletedQuests :many
SELECT
    qp.*,
    q.quest_key,
    q.quest_type,
    q.description,
    q.target_category,
    q.target_recipe_key
FROM quest_progress qp
JOIN quests q ON qp.quest_id = q.quest_id
WHERE qp.user_id = $1
  AND qp.completed_at IS NOT NULL
  AND qp.claimed_at IS NULL
ORDER BY qp.completed_at ASC;

-- name: ResetInactiveQuestProgress :execresult
DELETE FROM quest_progress
WHERE quest_id IN (
    SELECT quest_id FROM quests WHERE active = FALSE
);

-- name: GetWeeklyQuestResetState :one
SELECT * FROM weekly_quest_reset_state WHERE id = 1;

-- name: UpdateWeeklyQuestResetState :exec
UPDATE weekly_quest_reset_state
SET last_reset_time = $1, week_number = $2, year = $3,
    quests_generated = $4, progress_reset = $5
WHERE id = 1;
