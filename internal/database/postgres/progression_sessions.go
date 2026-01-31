package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Voting Session operations (multi-option voting)

func (r *progressionRepository) CreateVotingSession(ctx context.Context) (int, error) {
	sessionID, err := r.q.CreateVotingSession(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to create voting session: %w", err)
	}

	return int(sessionID), nil
}

func (r *progressionRepository) AddVotingOption(ctx context.Context, sessionID, nodeID, targetLevel int) error {
	err := r.q.AddVotingOption(ctx, generated.AddVotingOptionParams{
		SessionID:   int32(sessionID),
		NodeID:      int32(nodeID),
		TargetLevel: int32(targetLevel),
	})

	if err != nil {
		return fmt.Errorf("failed to add voting option: %w", err)
	}

	return nil
}

func (r *progressionRepository) GetActiveSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	// Status 'voting' is handled in query
	row, err := r.q.GetActiveSession(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	session := &domain.ProgressionVotingSession{
		ID:              int(row.ID),
		StartedAt:       row.StartedAt.Time,
		Status:          row.Status,
		EndedAt:         ptrTime(row.EndedAt),
		VotingDeadline:  row.VotingDeadline.Time,
		WinningOptionID: ptrInt(row.WinningOptionID),
	}

	// Get options
	session.Options, err = r.getSessionOptions(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetMostRecentSession returns the most recent voting session regardless of status
func (r *progressionRepository) GetMostRecentSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	row, err := r.q.GetMostRecentSession(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get most recent session: %w", err)
	}

	session := &domain.ProgressionVotingSession{
		ID:              int(row.ID),
		StartedAt:       row.StartedAt.Time,
		Status:          row.Status,
		EndedAt:         ptrTime(row.EndedAt),
		VotingDeadline:  row.VotingDeadline.Time,
		WinningOptionID: ptrInt(row.WinningOptionID),
	}

	// Get options
	session.Options, err = r.getSessionOptions(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (r *progressionRepository) GetSessionByID(ctx context.Context, sessionID int) (*domain.ProgressionVotingSession, error) {
	row, err := r.q.GetSessionByID(ctx, int32(sessionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	session := &domain.ProgressionVotingSession{
		ID:              int(row.ID),
		StartedAt:       row.StartedAt.Time,
		Status:          row.Status,
		EndedAt:         ptrTime(row.EndedAt),
		VotingDeadline:  row.VotingDeadline.Time,
		WinningOptionID: ptrInt(row.WinningOptionID),
	}

	session.Options, err = r.getSessionOptions(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (r *progressionRepository) getSessionOptions(ctx context.Context, sessionID int) ([]domain.ProgressionVotingOption, error) {
	rows, err := r.q.GetSessionOptions(ctx, int32(sessionID))
	if err != nil {
		return nil, fmt.Errorf("failed to get session options: %w", err)
	}

	options := make([]domain.ProgressionVotingOption, 0, len(rows))
	for _, row := range rows {
		opt := domain.ProgressionVotingOption{
			ID:                int(row.ID),
			SessionID:         int(row.SessionID),
			NodeID:            int(row.NodeID),
			TargetLevel:       int(row.TargetLevel),
			VoteCount:         int(row.VoteCount),
			LastHighestVoteAt: ptrTime(row.LastHighestVoteAt),
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
	err := r.q.IncrementOptionVote(ctx, int32(optionID))
	if err != nil {
		return fmt.Errorf("failed to increment option vote: %w", err)
	}

	// Update last_highest_vote_at if this option now has the highest votes
	err = r.q.UpdateOptionLastHighest(ctx, int32(optionID))
	return err
}

func (r *progressionRepository) EndVotingSession(ctx context.Context, sessionID int, winningOptionID *int) error {
	var optionID pgtype.Int4
	if winningOptionID != nil {
		optionID = pgtype.Int4{Int32: int32(*winningOptionID), Valid: true}
	}
	// else: leave optionID with Valid: false (NULL)

	err := r.q.EndVotingSession(ctx, generated.EndVotingSessionParams{
		ID:              int32(sessionID),
		WinningOptionID: optionID,
	})

	if err != nil {
		return fmt.Errorf("failed to end voting session: %w", err)
	}

	return nil
}

func (r *progressionRepository) FreezeVotingSession(ctx context.Context, sessionID int) error {
	err := r.q.FreezeVotingSession(ctx, int32(sessionID))
	if err != nil {
		return fmt.Errorf("failed to freeze voting session: %w", err)
	}
	return nil
}

func (r *progressionRepository) ResumeVotingSession(ctx context.Context, sessionID int) error {
	err := r.q.ResumeVotingSession(ctx, int32(sessionID))
	if err != nil {
		return fmt.Errorf("failed to resume voting session: %w", err)
	}
	return nil
}

func (r *progressionRepository) GetActiveOrFrozenSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	row, err := r.q.GetActiveOrFrozenSession(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active or frozen session: %w", err)
	}

	session := &domain.ProgressionVotingSession{
		ID:              int(row.ID),
		StartedAt:       row.StartedAt.Time,
		Status:          row.Status,
		EndedAt:         ptrTime(row.EndedAt),
		VotingDeadline:  row.VotingDeadline.Time,
		WinningOptionID: ptrInt(row.WinningOptionID),
	}

	// Get options
	session.Options, err = r.getSessionOptions(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (r *progressionRepository) GetSessionVoters(ctx context.Context, sessionID int) ([]string, error) {
	rows, err := r.q.GetSessionVoters(ctx, pgtype.Int4{Int32: int32(sessionID), Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get session voters: %w", err)
	}

	voters := make([]string, 0)
	voters = append(voters, rows...)

	return voters, nil
}

func (r *progressionRepository) HasUserVotedInSession(ctx context.Context, userID string, sessionID int) (bool, error) {
	return r.q.HasUserVotedInSession(ctx, generated.HasUserVotedInSessionParams{
		UserID:    userID,
		SessionID: pgtype.Int4{Int32: int32(sessionID), Valid: true},
	})
}

func (r *progressionRepository) RecordUserSessionVote(ctx context.Context, userID string, sessionID, optionID, nodeID int) error {
	err := r.q.RecordUserSessionVote(ctx, generated.RecordUserSessionVoteParams{
		UserID:    userID,
		SessionID: pgtype.Int4{Int32: int32(sessionID), Valid: true},
		OptionID:  pgtype.Int4{Int32: int32(optionID), Valid: true},
		NodeID:    int32(nodeID),
	})

	if err != nil {
		return fmt.Errorf("failed to record user session vote: %w", err)
	}

	return nil
}

// CheckAndRecordVoteAtomic atomically checks if a user has voted and records their vote if they haven't.
// This method uses SELECT FOR UPDATE to prevent race conditions when multiple concurrent vote requests arrive.
// Returns domain.ErrUserAlreadyVoted if the user has already voted in this session.
func (r *progressionRepository) CheckAndRecordVoteAtomic(ctx context.Context, userID string, sessionID, optionID, nodeID int) error {
	// Begin transaction
	txHelper, err := beginTx(ctx, r.pool, r.q)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer SafeRollback(ctx, txHelper.Tx())

	// Check if user has already voted with row lock
	hasVoted, err := txHelper.Queries().HasUserVotedInSessionForUpdate(ctx, generated.HasUserVotedInSessionForUpdateParams{
		UserID:    userID,
		SessionID: pgtype.Int4{Int32: int32(sessionID), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to check vote status: %w", err)
	}

	if hasVoted {
		return domain.ErrUserAlreadyVoted
	}

	// Increment vote count for the option
	err = txHelper.Queries().IncrementOptionVote(ctx, int32(optionID))
	if err != nil {
		return fmt.Errorf("failed to increment vote: %w", err)
	}

	// Record the user's vote
	err = txHelper.Queries().RecordUserSessionVote(ctx, generated.RecordUserSessionVoteParams{
		UserID:    userID,
		SessionID: pgtype.Int4{Int32: int32(sessionID), Valid: true},
		OptionID:  pgtype.Int4{Int32: int32(optionID), Valid: true},
		NodeID:    int32(nodeID),
	})
	if err != nil {
		return fmt.Errorf("failed to record user vote: %w", err)
	}

	// Commit transaction
	if err := txHelper.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Unlock Progress tracking

func (r *progressionRepository) CreateUnlockProgress(ctx context.Context) (int, error) {
	id, err := r.q.CreateUnlockProgress(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to create unlock progress: %w", err)
	}

	return int(id), nil
}

func (r *progressionRepository) GetActiveUnlockProgress(ctx context.Context) (*domain.UnlockProgress, error) {
	row, err := r.q.GetActiveUnlockProgress(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active unlock progress: %w", err)
	}

	progress := &domain.UnlockProgress{
		ID:                       int(row.ID),
		ContributionsAccumulated: int(row.ContributionsAccumulated),
		NodeID:                   ptrInt(row.NodeID),
		TargetLevel:              ptrInt(row.TargetLevel),
		UnlockedAt:               ptrTime(row.UnlockedAt),
		VotingSessionID:          ptrInt(row.VotingSessionID),
	}
	if row.StartedAt.Valid {
		progress.StartedAt = row.StartedAt.Time
	}

	return progress, nil
}

func (r *progressionRepository) AddContribution(ctx context.Context, progressID int, amount int) error {
	err := r.q.AddContribution(ctx, generated.AddContributionParams{
		ID:                       int32(progressID),
		ContributionsAccumulated: int32(amount),
	})
	if err != nil {
		return fmt.Errorf("failed to add contribution: %w", err)
	}
	return nil
}

func (r *progressionRepository) SetUnlockTarget(ctx context.Context, progressID int, nodeID int, targetLevel int, sessionID int) error {
	err := r.q.SetUnlockTarget(ctx, generated.SetUnlockTargetParams{
		ID:              int32(progressID),
		NodeID:          pgtype.Int4{Int32: int32(nodeID), Valid: true},
		TargetLevel:     pgtype.Int4{Int32: int32(targetLevel), Valid: true},
		VotingSessionID: pgtype.Int4{Int32: int32(sessionID), Valid: true},
	})

	if err != nil {
		return fmt.Errorf("failed to set unlock target: %w", err)
	}

	return nil
}

func (r *progressionRepository) CompleteUnlock(ctx context.Context, progressID int, rolloverPoints int) (int, error) {
	// Mark current progress as complete
	err := r.q.CompleteUnlock(ctx, int32(progressID))
	if err != nil {
		return 0, fmt.Errorf("failed to complete unlock: %w", err)
	}

	// Create new progress entry with rollover points
	newID, err := r.q.InsertNextUnlockProgress(ctx, int32(rolloverPoints))
	if err != nil {
		return 0, fmt.Errorf("failed to create next unlock progress: %w", err)
	}

	return int(newID), nil
}
