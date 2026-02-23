-- name: GetBonusModifiers :many
SELECT node_key, source_type, feature_key, modifier_type, base_value, per_level_value, max_value, min_value
FROM bonus_config
WHERE feature_key = $1;

-- name: GetAllBonusModifiers :many
SELECT node_key, source_type, feature_key, modifier_type, base_value, per_level_value, max_value, min_value
FROM bonus_config;

-- name: GetJobFeatureUnlockConfigs :many
SELECT job_key, feature_key, required_level
FROM job_unlock_config;

-- name: GetJobUnlockConfig :one
SELECT job_key, feature_key, required_level
FROM job_unlock_config
WHERE feature_key = $1;

-- name: GetBonusModifiersWithLevel :many
SELECT bc.node_key, bc.source_type, bc.feature_key, bc.modifier_type, bc.base_value, bc.per_level_value, bc.max_value, bc.min_value,
       COALESCE(u.current_level, 0)::int as progression_level
FROM bonus_config bc
LEFT JOIN progression_nodes n ON bc.node_key = n.node_key AND bc.source_type = 'progression'
LEFT JOIN progression_unlocks u ON u.node_id = n.id
WHERE bc.feature_key = $1;

-- name: ClearBonusModifiersForNode :exec
DELETE FROM bonus_config
WHERE node_key = $1;

-- name: InsertBonusModifier :exec
INSERT INTO bonus_config (
    node_key, source_type, feature_key, modifier_type, base_value, per_level_value, max_value, min_value
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);
