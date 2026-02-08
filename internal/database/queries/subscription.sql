-- name: GetUserSubscription :one
SELECT
    us.user_id,
    us.platform,
    us.tier_id,
    us.status,
    us.subscribed_at,
    us.expires_at,
    us.last_verified_at,
    us.created_at,
    us.updated_at,
    st.tier_name,
    st.display_name,
    st.tier_level
FROM user_subscriptions us
JOIN subscription_tiers st ON us.tier_id = st.tier_id
WHERE us.user_id = $1 AND us.platform = $2;

-- name: GetUserSubscriptions :many
SELECT
    us.user_id,
    us.platform,
    us.tier_id,
    us.status,
    us.subscribed_at,
    us.expires_at,
    us.last_verified_at,
    us.created_at,
    us.updated_at,
    st.tier_name,
    st.display_name,
    st.tier_level
FROM user_subscriptions us
JOIN subscription_tiers st ON us.tier_id = st.tier_id
WHERE us.user_id = $1
ORDER BY us.platform;

-- name: UpsertSubscription :exec
INSERT INTO user_subscriptions (
    user_id,
    platform,
    tier_id,
    status,
    subscribed_at,
    expires_at,
    last_verified_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (user_id, platform) DO UPDATE
SET
    tier_id = EXCLUDED.tier_id,
    status = EXCLUDED.status,
    subscribed_at = EXCLUDED.subscribed_at,
    expires_at = EXCLUDED.expires_at,
    last_verified_at = EXCLUDED.last_verified_at,
    updated_at = NOW();

-- name: GetExpiringSubscriptions :many
SELECT
    us.user_id,
    us.platform,
    us.tier_id,
    us.status,
    us.subscribed_at,
    us.expires_at,
    us.last_verified_at,
    us.created_at,
    us.updated_at,
    st.tier_name,
    st.display_name,
    st.tier_level
FROM user_subscriptions us
JOIN subscription_tiers st ON us.tier_id = st.tier_id
WHERE us.status = 'active' AND us.expires_at < $1
ORDER BY us.expires_at;

-- name: MarkSubscriptionExpired :exec
UPDATE user_subscriptions
SET status = 'expired', updated_at = NOW()
WHERE user_id = $1 AND platform = $2;

-- name: DeleteSubscription :exec
DELETE FROM user_subscriptions
WHERE user_id = $1 AND platform = $2;

-- name: GetTierByPlatformAndName :one
SELECT tier_id, platform, tier_name, display_name, tier_level, created_at
FROM subscription_tiers
WHERE platform = $1 AND tier_name = $2;

-- name: GetAllTiers :many
SELECT tier_id, platform, tier_name, display_name, tier_level, created_at
FROM subscription_tiers
ORDER BY platform, tier_level;

-- name: RecordSubscriptionHistory :exec
INSERT INTO subscription_history (
    user_id,
    platform,
    tier_id,
    event_type,
    subscribed_at,
    expires_at,
    metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetUserSubscriptionHistory :many
SELECT
    history_id,
    user_id,
    platform,
    tier_id,
    event_type,
    subscribed_at,
    expires_at,
    metadata,
    created_at
FROM subscription_history
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;
