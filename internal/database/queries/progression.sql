-- name: GetNodeByKey :one
SELECT id, node_key, node_type, display_name, description,
       max_level, unlock_cost, tier, size, category, sort_order, created_at
FROM progression_nodes
WHERE node_key = $1
LIMIT 1;

-- name: GetNodeByID :one
SELECT id, node_key, node_type, display_name, description,
       max_level, unlock_cost, tier, size, category, sort_order, created_at
FROM progression_nodes
WHERE id = $1;

-- name: GetAllNodes :many
SELECT id, node_key, node_type, display_name, description,
       max_level, unlock_cost, tier, size, category, sort_order, created_at
FROM progression_nodes
ORDER BY sort_order, id;

-- name: InsertNode :one
INSERT INTO progression_nodes (node_key, node_type, display_name, description, max_level, unlock_cost, tier, size, category, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: UpdateNode :exec
UPDATE progression_nodes 
SET node_type = $2, display_name = $3, description = $4,
    max_level = $5, unlock_cost = $6, tier = $7, size = $8, category = $9, sort_order = $10
WHERE id = $1;

-- name: GetUnlock :one
SELECT id, node_id, current_level, unlocked_at, unlocked_by, engagement_score
FROM progression_unlocks
WHERE node_id = $1 AND current_level = $2;

-- name: GetAllUnlocks :many
SELECT id, node_id, current_level, unlocked_at, unlocked_by, engagement_score
FROM progression_unlocks
ORDER BY unlocked_at;

-- name: IsNodeUnlocked :one
SELECT EXISTS(
    SELECT 1 FROM progression_unlocks pu
    JOIN progression_nodes pn ON pu.node_id = pn.id
    WHERE pn.node_key = $1 AND pu.current_level >= $2
);

-- name: UnlockNode :exec
INSERT INTO progression_unlocks (node_id, current_level, unlocked_by, engagement_score)
VALUES ($1, $2, $3, $4)
ON CONFLICT (node_id, current_level) DO NOTHING;

-- name: RelockNode :exec
DELETE FROM progression_unlocks WHERE node_id = $1 AND (current_level = $2 OR $2 = 0);

-- name: GetActiveVoting :one
SELECT id, node_id, target_level, vote_count, voting_started_at, voting_ends_at, is_active
FROM progression_voting
WHERE is_active = true
ORDER BY voting_started_at DESC
LIMIT 1;

-- name: StartVoting :exec
INSERT INTO progression_voting (node_id, target_level, vote_count, voting_ends_at, is_active)
VALUES ($1, $2, 0, $3, true)
ON CONFLICT (node_id, target_level) DO UPDATE
SET voting_started_at = CURRENT_TIMESTAMP, voting_ends_at = $3, is_active = true, vote_count = 0;

-- name: GetVoting :one
SELECT id, node_id, target_level, vote_count, voting_started_at, voting_ends_at, is_active
FROM progression_voting
WHERE node_id = $1 AND target_level = $2;

-- name: IncrementVote :exec
UPDATE progression_voting
SET vote_count = vote_count + 1
WHERE node_id = $1 AND target_level = $2;

-- name: EndVoting :exec
UPDATE progression_voting
SET is_active = false
WHERE node_id = $1 AND target_level = $2;

-- name: HasUserVoted :one
SELECT EXISTS(
    SELECT 1 FROM user_votes
    WHERE user_id = $1 AND node_id = $2 AND target_level = $3
);

-- name: RecordUserVote :exec
INSERT INTO user_votes (user_id, node_id, target_level)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, node_id, target_level) DO NOTHING;

-- name: UnlockUserProgression :exec
INSERT INTO user_progression (user_id, progression_type, progression_key, metadata)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, progression_type, progression_key) DO NOTHING;

-- name: IsUserProgressionUnlocked :one
SELECT EXISTS(
    SELECT 1 FROM user_progression
    WHERE user_id = $1 AND progression_type = $2 AND progression_key = $3
);

-- name: GetUserProgressions :many
SELECT user_id, progression_type, progression_key, unlocked_at, metadata
FROM user_progression
WHERE user_id = $1 AND progression_type = $2
ORDER BY unlocked_at;

-- name: RecordEngagement :exec
INSERT INTO engagement_metrics (user_id, metric_type, metric_value, metadata, recorded_at)
VALUES ($1, $2, $3, $4, COALESCE(sqlc.arg(recorded_at)::timestamp, CURRENT_TIMESTAMP));

-- name: GetEngagementMetricsAggregated :many
SELECT metric_type, SUM(metric_value)::bigint as total
FROM engagement_metrics
GROUP BY metric_type;

-- name: GetEngagementMetricsAggregatedSince :many
SELECT metric_type, SUM(metric_value)::bigint as total
FROM engagement_metrics
WHERE recorded_at >= $1
GROUP BY metric_type;

-- name: GetUserEngagementAggregated :many
SELECT metric_type, SUM(metric_value)::bigint as total
FROM engagement_metrics
WHERE user_id = $1
GROUP BY metric_type;

-- name: GetEngagementWeights :many
SELECT metric_type, weight FROM engagement_weights;

-- name: CountUnlocks :one
SELECT COUNT(*) FROM progression_unlocks;

-- name: GetTotalEngagementScore :one
SELECT COALESCE(SUM(metric_value), 0)::bigint FROM engagement_metrics;

-- name: RecordReset :exec
INSERT INTO progression_resets (reset_by, reason, nodes_reset_count, engagement_score_at_reset)
VALUES ($1, $2, $3, $4);

-- name: ClearUnlocksExceptRoot :exec
DELETE FROM progression_unlocks
WHERE node_id != (SELECT id FROM progression_nodes WHERE node_key = 'progression_system');

-- name: ClearAllVoting :exec
DELETE FROM progression_voting;

-- name: ClearAllUserVotes :exec
DELETE FROM user_votes;

-- name: ClearAllUserProgression :exec
DELETE FROM user_progression;

-- name: CreateVotingSession :one
INSERT INTO progression_voting_sessions (status)
VALUES ('voting')
RETURNING id;

-- name: AddVotingOption :exec
INSERT INTO progression_voting_options (session_id, node_id, target_level, vote_count)
VALUES ($1, $2, $3, 0);

-- name: GetActiveSession :one
SELECT id, started_at, ended_at, voting_deadline, winning_option_id, status
FROM progression_voting_sessions
WHERE status = ('voting')::text
ORDER BY started_at DESC
LIMIT 1;

-- name: GetMostRecentSession :one
SELECT id, started_at, ended_at, voting_deadline, winning_option_id, status
FROM progression_voting_sessions
ORDER BY started_at DESC
LIMIT 1;

-- name: GetSessionByID :one
SELECT id, started_at, ended_at, voting_deadline, winning_option_id, status
FROM progression_voting_sessions
WHERE id = $1;

-- name: GetSessionOptions :many
SELECT o.id, o.session_id, o.node_id, o.target_level, o.vote_count, o.last_highest_vote_at
FROM progression_voting_options o
WHERE o.session_id = $1
ORDER BY o.id;

-- name: IncrementOptionVote :exec
UPDATE progression_voting_options
SET vote_count = vote_count + 1
WHERE id = $1;

-- name: UpdateOptionLastHighest :exec
UPDATE progression_voting_options o
SET last_highest_vote_at = NOW()
WHERE o.id = $1
  AND o.vote_count = (
      SELECT MAX(vote_count) FROM progression_voting_options
      WHERE session_id = o.session_id
  )
  AND (o.last_highest_vote_at IS NULL OR EXISTS (
      SELECT 1 FROM progression_voting_options o2
      WHERE o2.session_id = o.session_id
        AND o2.id != o.id
        AND o2.vote_count = o.vote_count
  ));

-- name: EndVotingSession :exec
UPDATE progression_voting_sessions
SET ended_at = NOW(),
    winning_option_id = $2,
    status = 'completed'
WHERE id = $1;

-- name: FreezeVotingSession :exec
UPDATE progression_voting_sessions
SET status = 'frozen'
WHERE id = $1 AND status = 'voting';

-- name: ResumeVotingSession :exec
UPDATE progression_voting_sessions
SET status = 'voting'
WHERE id = $1 AND status = 'frozen';

-- name: GetActiveOrFrozenSession :one
SELECT id, started_at, ended_at, voting_deadline, winning_option_id, status
FROM progression_voting_sessions
WHERE status IN ('voting', 'frozen')
ORDER BY started_at DESC
LIMIT 1;

-- name: GetSessionVoters :many
SELECT DISTINCT user_id
FROM user_votes
WHERE session_id = $1;

-- name: HasUserVotedInSession :one
-- Read-only check for whether a user has voted in a session.
-- Does NOT prevent concurrent votes - use HasUserVotedInSessionForUpdate for that.
-- Safe for: tests, display logic, non-critical status checks.
-- For concurrent-safe voting: Use HasUserVotedInSessionForUpdate within a transaction.
SELECT EXISTS(
    SELECT 1 FROM user_votes
    WHERE user_id = $1 AND session_id = $2
);

-- name: HasUserVotedInSessionForUpdate :one
-- Locks the user's vote record for the session to prevent concurrent vote attempts.
-- Returns true if user has already voted in this session.
-- Must be used within a transaction.
SELECT EXISTS(
    SELECT 1 FROM user_votes
    WHERE user_id = $1 AND session_id = $2
    FOR UPDATE
);

-- name: RecordUserSessionVote :exec
INSERT INTO user_votes (user_id, session_id, option_id, node_id, target_level)
VALUES ($1, $2, $3, $4, 1);

-- name: CreateUnlockProgress :one
INSERT INTO progression_unlock_progress (contributions_accumulated)
VALUES (0)
RETURNING id;

-- name: GetActiveUnlockProgress :one
SELECT id, node_id, target_level, contributions_accumulated, started_at, unlocked_at, voting_session_id
FROM progression_unlock_progress
WHERE unlocked_at IS NULL
ORDER BY started_at DESC
LIMIT 1;

-- name: AddContribution :exec
UPDATE progression_unlock_progress
SET contributions_accumulated = contributions_accumulated + $2
WHERE id = $1;

-- name: SetUnlockTarget :exec
UPDATE progression_unlock_progress
SET node_id = $2, target_level = $3, voting_session_id = $4
WHERE id = $1;

-- name: CompleteUnlock :exec
UPDATE progression_unlock_progress
SET unlocked_at = NOW()
WHERE id = $1;

-- name: InsertNextUnlockProgress :one
INSERT INTO progression_unlock_progress (contributions_accumulated)
VALUES ($1)
RETURNING id;

-- name: GetNodePrerequisites :many
SELECT n.id, n.node_key, n.node_type, n.display_name, n.description,
       n.max_level, n.unlock_cost, n.tier, n.size, n.category, n.sort_order, n.created_at
FROM progression_nodes n
INNER JOIN progression_prerequisites p ON n.id = p.prerequisite_node_id
WHERE p.node_id = $1
ORDER BY n.sort_order, n.id;

-- name: GetNodeDependents :many
SELECT n.id, n.node_key, n.node_type, n.display_name, n.description,
       n.max_level, n.unlock_cost, n.tier, n.size, n.category, n.sort_order, n.created_at
FROM progression_nodes n
INNER JOIN progression_prerequisites p ON n.id = p.node_id
WHERE p.prerequisite_node_id = $1
ORDER BY n.sort_order, n.id;

-- name: GetContributionLeaderboard :many
WITH user_contributions AS (
    SELECT
        user_id,
        SUM(metric_value) as total_contribution
    FROM engagement_metrics
    GROUP BY user_id
)
SELECT
    user_id,
    total_contribution,
    ROW_NUMBER() OVER (ORDER BY total_contribution DESC)::bigint as rank
FROM user_contributions
ORDER BY total_contribution DESC
LIMIT $1;

-- name: ClearNodePrerequisites :exec
DELETE FROM progression_prerequisites WHERE node_id = $1;

-- name: InsertNodePrerequisite :exec
INSERT INTO progression_prerequisites (node_id, prerequisite_node_id)
VALUES ($1, $2)
ON CONFLICT (node_id, prerequisite_node_id) DO NOTHING;

-- name: GetNodeByFeatureKey :one
SELECT n.*, COALESCE(u.current_level, 0)::int as unlock_level
FROM progression_nodes n
LEFT JOIN progression_unlocks u ON u.node_id = n.id
WHERE n.modifier_config->>'feature_key' = $1
LIMIT 1;

-- name: GetAllNodesByFeatureKey :many
SELECT n.*, COALESCE(u.current_level, 0)::int as unlock_level
FROM progression_nodes n
LEFT JOIN progression_unlocks u ON u.node_id = n.id
WHERE n.modifier_config->>'feature_key' = $1
ORDER BY n.tier ASC, n.id ASC;

-- name: GetDailyEngagementTotals :many
SELECT DATE(recorded_at)::timestamp as day, SUM(em.metric_value * ew.weight)::bigint as total_points
FROM engagement_metrics em
JOIN engagement_weights ew ON em.metric_type = ew.metric_type
WHERE recorded_at >= $1
GROUP BY DATE(recorded_at)
ORDER BY day ASC;

-- name: ClearAllVotingOptions :exec
DELETE FROM progression_voting_options;

-- name: ClearAllVotingSessions :exec
DELETE FROM progression_voting_sessions;

-- name: ClearAllUnlockProgress :exec
DELETE FROM progression_unlock_progress;

-- name: ClearUnlockProgressForNode :exec
DELETE FROM progression_unlock_progress WHERE node_id = $1;

-- name: CountUnlockedNodesBelowTier :one
SELECT COUNT(DISTINCT pu.node_id)::int
FROM progression_unlocks pu
JOIN progression_nodes pn ON pu.node_id = pn.id
WHERE pn.tier < $1;

-- name: CountTotalUnlockedNodes :one
SELECT COUNT(DISTINCT node_id)::int
FROM progression_unlocks;

-- name: GetNodeDynamicPrerequisites :one
SELECT COALESCE(dynamic_prerequisites, '[]'::jsonb)
FROM progression_nodes
WHERE id = $1;

-- name: UpdateNodeDynamicPrerequisites :exec
UPDATE progression_nodes
SET dynamic_prerequisites = $2
WHERE id = $1;
