-- name: GetAllJobs :many
SELECT id, job_key, display_name, description, associated_features, created_at
FROM jobs
ORDER BY id;

-- name: GetJobByKey :one
SELECT id, job_key, display_name, description, associated_features, created_at
FROM jobs
WHERE job_key = $1;

-- name: GetUserJobs :many
SELECT user_id, job_id, current_xp, current_level, xp_gained_today, last_xp_gain
FROM user_jobs
WHERE user_id = $1
ORDER BY current_level DESC, current_xp DESC;

-- name: GetUserJob :one
SELECT user_id, job_id, current_xp, current_level, xp_gained_today, last_xp_gain
FROM user_jobs
WHERE user_id = $1 AND job_id = $2;

-- name: UpsertUserJob :exec
INSERT INTO user_jobs (user_id, job_id, current_xp, current_level, xp_gained_today, last_xp_gain)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, job_id)
DO UPDATE SET
    current_xp = EXCLUDED.current_xp,
    current_level = EXCLUDED.current_level,
    xp_gained_today = EXCLUDED.xp_gained_today,
    last_xp_gain = EXCLUDED.last_xp_gain;

-- name: RecordJobXPEvent :exec
INSERT INTO job_xp_events (id, user_id, job_id, xp_amount, source_type, source_metadata, recorded_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetJobLevelBonuses :many
SELECT id, job_id, min_level, bonus_type, bonus_value, description
FROM job_level_bonuses
WHERE job_id = $1 AND min_level <= $2
ORDER BY min_level DESC;

-- name: ResetDailyJobXP :execresult
UPDATE user_jobs
SET xp_gained_today = 0;

-- name: GetUserJobsByPlatform :many
SELECT uj.user_id, uj.job_id, uj.current_xp, uj.current_level, uj.xp_gained_today, uj.last_xp_gain
FROM user_jobs uj
JOIN user_platform_links upl ON uj.user_id = upl.user_id
JOIN platforms p ON upl.platform_id = p.platform_id
WHERE p.name = $1 AND upl.platform_user_id = $2
ORDER BY uj.current_level DESC, uj.current_xp DESC;
