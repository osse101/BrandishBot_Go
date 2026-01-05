package postgres

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GetContributionLeaderboard returns top contributors
func (r *progressionRepository) GetContributionLeaderboard(ctx context.Context, limit int) ([]domain.ContributionLeaderboardEntry, error) {
	rows, err := r.q.GetContributionLeaderboard(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get contribution leaderboard: %w", err)
	}

	leaderboard := make([]domain.ContributionLeaderboardEntry, 0)
	for _, row := range rows {
		entry := domain.ContributionLeaderboardEntry{
			UserID:       row.UserID,
			Contribution: int(row.TotalContribution),
			Rank:         int(row.Rank),
		}
		leaderboard = append(leaderboard, entry)
	}

	return leaderboard, nil
}
