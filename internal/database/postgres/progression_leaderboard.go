package postgres

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GetContributionLeaderboard returns top contributors
func (r *progressionRepository) GetContributionLeaderboard(ctx context.Context, limit int) ([]domain.ContributionLeaderboardEntry, error) {
	query := `
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
			ROW_NUMBER() OVER (ORDER BY total_contribution DESC) as rank
		FROM user_contributions
		ORDER BY total_contribution DESC
		LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get contribution leaderboard: %w", err)
	}
	defer rows.Close()

	leaderboard := make([]domain.ContributionLeaderboardEntry, 0)
	for rows.Next() {
		var entry domain.ContributionLeaderboardEntry
		if err := rows.Scan(&entry.UserID, &entry.Contribution, &entry.Rank); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		leaderboard = append(leaderboard, entry)
	}

	return leaderboard, nil
}
