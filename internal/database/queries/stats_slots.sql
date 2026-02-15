-- name: GetUserSlotsStats :one
-- Calculate aggregate slots statistics for a user within a time period
SELECT
    COUNT(*) FILTER (WHERE event_type = 'slots_spin') as total_spins,
    COUNT(*) FILTER (WHERE event_type = 'slots_win') as total_wins,
    COALESCE(SUM((event_data->>'bet_amount')::int) FILTER (WHERE event_type = 'slots_spin'), 0) as total_bet,
    COALESCE(SUM((event_data->>'payout_amount')::int) FILTER (WHERE event_type = 'slots_spin'), 0) as total_payout,
    COUNT(*) FILTER (WHERE event_type = 'slots_mega_jackpot') as mega_jackpots_hit,
    COALESCE(MAX((event_data->>'payout_amount')::int) FILTER (WHERE event_type = 'slots_spin'), 0) as biggest_win
FROM stats_events
WHERE user_id = sqlc.arg(user_id)
  AND created_at >= sqlc.arg(start_time)
  AND created_at <= sqlc.arg(end_time)
  AND event_type LIKE 'slots_%';

-- name: GetSlotsLeaderboardByProfit :many
-- Get top users by net profit (total payout - total bet) for a time period
SELECT
    se.user_id,
    u.username,
    COALESCE(SUM((se.event_data->>'payout_amount')::int) - SUM((se.event_data->>'bet_amount')::int), 0) as net_profit,
    COUNT(*) FILTER (WHERE se.event_type = 'slots_spin') as total_spins
FROM stats_events se
JOIN users u ON se.user_id = u.user_id
WHERE se.event_type = 'slots_spin'
  AND se.created_at >= sqlc.arg(start_time)
  AND se.created_at <= sqlc.arg(end_time)
GROUP BY se.user_id, u.username
ORDER BY net_profit DESC
LIMIT sqlc.arg(result_limit);

-- name: GetSlotsLeaderboardByWinRate :many
-- Get top users by win rate for a time period (minimum spins required)
SELECT
    se.user_id,
    u.username,
    COUNT(*) FILTER (WHERE se.event_type = 'slots_spin') as total_spins,
    COUNT(*) FILTER (WHERE se.event_type = 'slots_win') as total_wins,
    CASE
        WHEN COUNT(*) FILTER (WHERE se.event_type = 'slots_spin') > 0
        THEN (COUNT(*) FILTER (WHERE se.event_type = 'slots_win')::float / COUNT(*) FILTER (WHERE se.event_type = 'slots_spin')::float * 100)
        ELSE 0
    END as win_rate
FROM stats_events se
JOIN users u ON se.user_id = u.user_id
WHERE se.event_type LIKE 'slots_%'
  AND se.created_at >= sqlc.arg(start_time)
  AND se.created_at <= sqlc.arg(end_time)
GROUP BY se.user_id, u.username
HAVING COUNT(*) FILTER (WHERE se.event_type = 'slots_spin') >= sqlc.arg(min_spins)::int8
ORDER BY win_rate DESC
LIMIT sqlc.arg(result_limit);

-- name: GetSlotsLeaderboardByMegaJackpots :many
-- Get top users by mega jackpots hit for a time period
SELECT
    se.user_id,
    u.username,
    COUNT(*) as mega_jackpots_hit
FROM stats_events se
JOIN users u ON se.user_id = u.user_id
WHERE se.event_type = 'slots_mega_jackpot'
  AND se.created_at >= sqlc.arg(start_time)
  AND se.created_at <= sqlc.arg(end_time)
GROUP BY se.user_id, u.username
ORDER BY mega_jackpots_hit DESC
LIMIT sqlc.arg(result_limit);
