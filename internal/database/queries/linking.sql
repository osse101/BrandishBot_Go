-- name: CreateToken :exec
INSERT INTO link_tokens (token, source_platform, source_platform_id, state, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetToken :one
SELECT token, source_platform, source_platform_id, 
       COALESCE(target_platform, '') AS target_platform,
       COALESCE(target_platform_id, '') AS target_platform_id,
       state, created_at, expires_at
FROM link_tokens
WHERE token = $1;

-- name: UpdateToken :exec
UPDATE link_tokens
SET target_platform = $2, target_platform_id = $3, state = $4
WHERE token = $1;

-- name: InvalidateTokensForSource :exec
UPDATE link_tokens
SET state = 'expired'
WHERE source_platform = $1 AND source_platform_id = $2 AND state IN ('pending', 'claimed');

-- name: CleanupExpiredTokens :exec
DELETE FROM link_tokens
WHERE expires_at < NOW() - INTERVAL '1 hour';

-- name: GetClaimedTokenForSource :one
SELECT token, source_platform, source_platform_id, 
       COALESCE(target_platform, '') AS target_platform,
       COALESCE(target_platform_id, '') AS target_platform_id,
       state, created_at, expires_at
FROM link_tokens
WHERE source_platform = $1 AND source_platform_id = $2 AND state = 'claimed'
ORDER BY created_at DESC
LIMIT 1;
