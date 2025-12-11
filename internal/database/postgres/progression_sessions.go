package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Voting Session operations (multi-option voting)

func (r *progressionRepository) CreateVotingSession(ctx context.Context) (int, error) {
	query := `
		INSERT INTO progression_voting_sessions (status)
		VALUES ('voting')
		RETURNING id`

	var sessionID int
	err := r.pool.QueryRow(ctx, query).Scan(&sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to create voting session: %w", err)
	}

	return sessionID, nil
}

func (r *progressionRepository) AddVotingOption(ctx context.Context, sessionID, nodeID, targetLevel int) error {
	query := `
		INSERT INTO progression_voting_options (session_id, node_id, target_level, vote_count)
		VALUES ($1, $2, $3, 0)`

	_, err := r.pool.Exec(ctx, query, sessionID, nodeID, targetLevel)
	if err != nil {
		return fmt.Errorf("failed to add voting option: %w", err)
	}

	return nil
}

func (r *progressionRepository) GetActiveSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	// Get session with status 'voting'
	sessionQuery := `
		SELECT id, started_at, ended_at, voting_deadline, winning_option_id, status
		FROM progression_voting_sessions
		WHERE status = 'voting'
		ORDER BY started_at DESC
		LIMIT 1`

	var session domain.ProgressionVotingSession
	err := r.pool.QueryRow(ctx, sessionQuery).Scan(
		&session.ID, &session.StartedAt, &session.EndedAt, &session.VotingDeadline,
		&session.WinningOptionID, &session.Status,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	// Get options
	session.Options, err = r.getSessionOptions(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *progressionRepository) GetSessionByID(ctx context.Context, sessionID int) (*domain.ProgressionVotingSession, error) {
	sessionQuery := `
		SELECT id, started_at, ended_at, voting_deadline, winning_option_id, status
		FROM progression_voting_sessions
		WHERE id = $1`

	var session domain.ProgressionVotingSession
	err := r.pool.QueryRow(ctx, sessionQuery, sessionID).Scan(
		&session.ID, &session.StartedAt, &session.EndedAt, &session.VotingDeadline,
		&session.WinningOptionID, &session.Status,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	session.Options, err = r.getSessionOptions(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *progressionRepository) getSessionOptions(ctx context.Context, sessionID int) ([]domain.ProgressionVotingOption, error) {
	optionsQuery := `
		SELECT o.id, o.session_id, o.node_id, o.target_level, o.vote_count, o.last_highest_vote_at
		FROM progression_voting_options o
		WHERE o.session_id = $1
		ORDER BY o.id`

	rows, err := r.pool.Query(ctx, optionsQuery, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session options: %w", err)
	}
	defer rows.Close()

	options := make([]domain.ProgressionVotingOption, 0)
	for rows.Next() {
		var opt domain.ProgressionVotingOption
		err := rows.Scan(&opt.ID, &opt.SessionID, &opt.NodeID, &opt.TargetLevel, &opt.VoteCount, &opt.LastHighestVoteAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan option: %w", err)
		}

		// Get node details
		node, _ := r.GetNodeByID(ctx, opt.NodeID)
		opt.NodeDetails = node

		options = append(options, opt)
	}

	return options, nil
}

func (r *progressionRepository) IncrementOptionVote(ctx context.Context, optionID int) error {
	// First, increment the vote count
	_, err := r.pool.Exec(ctx, `
		UPDATE progression_voting_options
		SET vote_count = vote_count + 1
		WHERE id = $1`, optionID)
	if err != nil {
		return fmt.Errorf("failed to increment option vote: %w", err)
	}

	// Update last_highest_vote_at if this option now has the highest votes
	_, err = r.pool.Exec(ctx, `
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
		  ))`, optionID)

	return err
}

func (r *progressionRepository) EndVotingSession(ctx context.Context, sessionID int, winningOptionID int) error {
	query := `
		UPDATE progression_voting_sessions
		SET ended_at = NOW(),
		    winning_option_id = $2,
		    status = 'completed'
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, sessionID, winningOptionID)
	if err != nil {
		return fmt.Errorf("failed to end voting session: %w", err)
	}

	return nil
}

func (r *progressionRepository) GetSessionVoters(ctx context.Context, sessionID int) ([]string, error) {
	query := `
		SELECT DISTINCT user_id
		FROM user_votes
		WHERE session_id = $1`

	rows, err := r.pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session voters: %w", err)
	}
	defer rows.Close()

	voters := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan voter: %w", err)
		}
		voters = append(voters, userID)
	}

	return voters, nil
}

func (r *progressionRepository) HasUserVotedInSession(ctx context.Context, userID string, sessionID int) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_votes
			WHERE user_id = $1 AND session_id = $2
		)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, sessionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user vote in session: %w", err)
	}

	return exists, nil
}

func (r *progressionRepository) RecordUserSessionVote(ctx context.Context, userID string, sessionID, optionID, nodeID int) error {
	query := `
		INSERT INTO user_votes (user_id, session_id, option_id, node_id, target_level)
		VALUES ($1, $2, $3, $4, 1)`

	_, err := r.pool.Exec(ctx, query, userID, sessionID, optionID, nodeID)
	if err != nil {
		return fmt.Errorf("failed to record user session vote: %w", err)
	}

	return nil
}

// Unlock Progress tracking

func (r *progressionRepository) CreateUnlockProgress(ctx context.Context) (int, error) {
	query := `
		INSERT INTO progression_unlock_progress (contributions_accumulated)
		VALUES (0)
		RETURNING id`

	var id int
	err := r.pool.QueryRow(ctx, query).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create unlock progress: %w", err)
	}

	return id, nil
}

func (r *progressionRepository) GetActiveUnlockProgress(ctx context.Context) (*domain.UnlockProgress, error) {
	query := `
		SELECT id, node_id, target_level, contributions_accumulated, started_at, unlocked_at, voting_session_id
		FROM progression_unlock_progress
		WHERE unlocked_at IS NULL
		ORDER BY started_at DESC
		LIMIT 1`

	var progress domain.UnlockProgress
	err := r.pool.QueryRow(ctx, query).Scan(
		&progress.ID, &progress.NodeID, &progress.TargetLevel,
		&progress.ContributionsAccumulated, &progress.StartedAt,
		&progress.UnlockedAt, &progress.VotingSessionID,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active unlock progress: %w", err)
	}

	return &progress, nil
}

func (r *progressionRepository) AddContribution(ctx context.Context, progressID int, amount int) error {
	query := `
		UPDATE progression_unlock_progress
		SET contributions_accumulated = contributions_accumulated + $2
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, progressID, amount)
	if err != nil {
		return fmt.Errorf("failed to add contribution: %w", err)
	}

	return nil
}

func (r *progressionRepository) SetUnlockTarget(ctx context.Context, progressID int, nodeID int, targetLevel int, sessionID int) error {
	query := `
		UPDATE progression_unlock_progress
		SET node_id = $2, target_level = $3, voting_session_id = $4
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, progressID, nodeID, targetLevel, sessionID)
	if err != nil {
		return fmt.Errorf("failed to set unlock target: %w", err)
	}

	return nil
}

func (r *progressionRepository) CompleteUnlock(ctx context.Context, progressID int, rolloverPoints int) (int, error) {
	// Mark current progress as complete
	_, err := r.pool.Exec(ctx, `
		UPDATE progression_unlock_progress
		SET unlocked_at = NOW()
		WHERE id = $1`, progressID)
	if err != nil {
		return 0, fmt.Errorf("failed to complete unlock: %w", err)
	}

	// Create new progress entry with rollover points
	query := `
		INSERT INTO progression_unlock_progress (contributions_accumulated)
		VALUES ($1)
		RETURNING id`

	var newID int
	err = r.pool.QueryRow(ctx, query, rolloverPoints).Scan(&newID)
	if err != nil {
		return 0, fmt.Errorf("failed to create next unlock progress: %w", err)
	}

	return newID, nil
}
