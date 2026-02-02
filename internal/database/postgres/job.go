package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// JobRepository implements the job repository for PostgreSQL
type JobRepository struct {
	db *pgxpool.Pool
	q  *generated.Queries
}

// NewJobRepository creates a new JobRepository
func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{
		db: db,
		q:  generated.New(db),
	}
}

func (r *JobRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	row, err := r.q.GetUserByPlatformID(ctx, generated.GetUserByPlatformIDParams{
		Name:           platform,
		PlatformUserID: platformID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user core data: %w", err)
	}
	return mapUserAndLinks(ctx, r.q, row.UserID, row.Username)
}

// GetAllJobs retrieves all job definitions
func (r *JobRepository) GetAllJobs(ctx context.Context) ([]domain.Job, error) {
	rows, err := r.q.GetAllJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}

	jobs := make([]domain.Job, 0, len(rows))
	for _, row := range rows {
		jobs = append(jobs, domain.Job{
			ID:                 int(row.ID),
			JobKey:             row.JobKey,
			DisplayName:        row.DisplayName,
			Description:        row.Description.String,
			AssociatedFeatures: row.AssociatedFeatures,
			CreatedAt:          row.CreatedAt.Time,
		})
	}

	return jobs, nil
}

// GetJobByKey retrieves a job by its key
func (r *JobRepository) GetJobByKey(ctx context.Context, jobKey string) (*domain.Job, error) {
	row, err := r.q.GetJobByKey(ctx, jobKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf(ErrMsgJobNotFound, jobKey, err)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &domain.Job{
		ID:                 int(row.ID),
		JobKey:             row.JobKey,
		DisplayName:        row.DisplayName,
		Description:        row.Description.String,
		AssociatedFeatures: row.AssociatedFeatures,
		CreatedAt:          row.CreatedAt.Time,
	}, nil
}

// GetUserJobs retrieves all job progress for a user
func (r *JobRepository) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJob, error) {
	userUUID, err := parseUserUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.q.GetUserJobs(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user jobs: %w", err)
	}

	userJobs := make([]domain.UserJob, 0, len(rows))
	for _, row := range rows {
		lastXPGain := row.LastXpGain.Time
		userJobs = append(userJobs, domain.UserJob{
			UserID:        row.UserID.String(),
			JobID:         int(row.JobID),
			CurrentXP:     row.CurrentXp,
			CurrentLevel:  int(row.CurrentLevel),
			XPGainedToday: row.XpGainedToday.Int64,
			LastXPGain:    &lastXPGain,
		})
	}

	return userJobs, nil
}

// GetUserJobsByPlatform retrieves all job progress for a user by their platform ID
func (r *JobRepository) GetUserJobsByPlatform(ctx context.Context, platform, platformID string) ([]domain.UserJob, error) {
	rows, err := r.q.GetUserJobsByPlatform(ctx, generated.GetUserJobsByPlatformParams{
		Name:           platform,
		PlatformUserID: platformID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query user jobs by platform: %w", err)
	}

	userJobs := make([]domain.UserJob, 0, len(rows))
	for _, row := range rows {
		lastXPGain := row.LastXpGain.Time
		userJobs = append(userJobs, domain.UserJob{
			UserID:        row.UserID.String(),
			JobID:         int(row.JobID),
			CurrentXP:     row.CurrentXp,
			CurrentLevel:  int(row.CurrentLevel),
			XPGainedToday: row.XpGainedToday.Int64,
			LastXPGain:    &lastXPGain,
		})
	}

	return userJobs, nil
}

// GetUserJob retrieves a single user's progress for a specific job
func (r *JobRepository) GetUserJob(ctx context.Context, userID string, jobID int) (*domain.UserJob, error) {
	userUUID, err := parseUserUUID(userID)
	if err != nil {
		return nil, err
	}

	row, err := r.q.GetUserJob(ctx, generated.GetUserJobParams{
		UserID: userUUID,
		JobID:  int32(jobID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not an error, just no progress yet
		}
		return nil, fmt.Errorf("failed to get user job: %w", err)
	}

	lastXPGain := row.LastXpGain.Time
	return &domain.UserJob{
		UserID:        row.UserID.String(),
		JobID:         int(row.JobID),
		CurrentXP:     row.CurrentXp,
		CurrentLevel:  int(row.CurrentLevel),
		XPGainedToday: row.XpGainedToday.Int64,
		LastXPGain:    &lastXPGain,
	}, nil
}

// UpsertUserJob creates or updates a user's job progress
func (r *JobRepository) UpsertUserJob(ctx context.Context, userJob *domain.UserJob) error {
	userUUID, err := parseUserUUID(userJob.UserID)
	if err != nil {
		return err
	}

	var lastXPGain time.Time
	if userJob.LastXPGain != nil {
		lastXPGain = *userJob.LastXPGain
	}

	params := generated.UpsertUserJobParams{
		UserID:        userUUID,
		JobID:         int32(userJob.JobID),
		CurrentXp:     userJob.CurrentXP,
		CurrentLevel:  int32(userJob.CurrentLevel),
		XpGainedToday: pgtype.Int8{Int64: userJob.XPGainedToday, Valid: true},
		LastXpGain:    pgtype.Timestamptz{Time: lastXPGain, Valid: userJob.LastXPGain != nil},
	}

	err = r.q.UpsertUserJob(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert user job: %w", err)
	}

	return nil
}

// RecordJobXPEvent logs an XP gain event
func (r *JobRepository) RecordJobXPEvent(ctx context.Context, event *domain.JobXPEvent) error {
	metadataJSON, err := json.Marshal(event.SourceMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	userUUID, err := parseUserUUID(event.UserID)
	if err != nil {
		return err
	}

	params := generated.RecordJobXPEventParams{
		ID:             event.ID,
		UserID:         userUUID,
		JobID:          int32(event.JobID),
		XpAmount:       int32(event.XPAmount),
		SourceType:     event.SourceType,
		SourceMetadata: metadataJSON,
		RecordedAt:     pgtype.Timestamptz{Time: event.RecordedAt, Valid: true},
	}

	err = r.q.RecordJobXPEvent(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to record XP event: %w", err)
	}

	return nil
}

// GetJobLevelBonuses retrieves bonuses for a job at or below a given level
func (r *JobRepository) GetJobLevelBonuses(ctx context.Context, jobID int, level int) ([]domain.JobLevelBonus, error) {
	rows, err := r.q.GetJobLevelBonuses(ctx, generated.GetJobLevelBonusesParams{
		JobID:    int32(jobID),
		MinLevel: int32(level),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query job bonuses: %w", err)
	}

	bonuses := make([]domain.JobLevelBonus, 0, len(rows))
	for _, row := range rows {
		// Convert pgtype.Numeric to float64 with proper error handling
		bonusValue, err := numericToFloat64(row.BonusValue)
		if err != nil {
			return nil, fmt.Errorf("failed to convert bonus value for job %d: %w", row.JobID, err)
		}

		bonuses = append(bonuses, domain.JobLevelBonus{
			ID:          int(row.ID),
			JobID:       int(row.JobID),
			MinLevel:    int(row.MinLevel),
			BonusType:   row.BonusType,
			BonusValue:  bonusValue,
			Description: row.Description.String,
		})
	}

	return bonuses, nil
}

// ResetDailyJobXP resets the xp_gained_today counter for all users
// Returns the number of records affected
func (r *JobRepository) ResetDailyJobXP(ctx context.Context) (int64, error) {
	result, err := r.q.ResetDailyJobXP(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to reset daily XP: %w", err)
	}

	rows := result.RowsAffected()
	logger.FromContext(ctx).Info("Reset daily XP", "records_affected", rows)

	return rows, nil
}

// GetLastDailyResetTime retrieves the last daily reset time and records affected
func (r *JobRepository) GetLastDailyResetTime(ctx context.Context) (time.Time, int64, error) {
	row, err := r.q.GetLastDailyResetTime(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, 0, nil
		}
		return time.Time{}, 0, fmt.Errorf("failed to get last reset time: %w", err)
	}

	return row.LastResetTime.Time, int64(row.RecordsAffected), nil
}

// UpdateDailyResetTime updates the last reset time and records affected
func (r *JobRepository) UpdateDailyResetTime(ctx context.Context, resetTime time.Time, recordsAffected int64) error {
	err := r.q.UpdateDailyResetTime(ctx, generated.UpdateDailyResetTimeParams{
		LastResetTime:   pgtype.Timestamptz{Time: resetTime, Valid: true},
		RecordsAffected: int32(recordsAffected),
	})
	if err != nil {
		return fmt.Errorf("failed to update reset time: %w", err)
	}

	return nil
}
