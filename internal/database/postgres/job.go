package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
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

// GetAllJobs retrieves all job definitions
func (r *JobRepository) GetAllJobs(ctx context.Context) ([]domain.Job, error) {
	rows, err := r.q.GetAllJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}

	var jobs []domain.Job
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
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", jobKey)
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
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := r.q.GetUserJobs(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user jobs: %w", err)
	}

	var userJobs []domain.UserJob
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
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	row, err := r.q.GetUserJob(ctx, generated.GetUserJobParams{
		UserID: userUUID,
		JobID:  int32(jobID),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
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
	userUUID, err := uuid.Parse(userJob.UserID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
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

	userUUID, err := uuid.Parse(event.UserID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
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

	var bonuses []domain.JobLevelBonus
	for _, row := range rows {
		// Convert pgtype.Numeric to float64
		// Best effort conversion
		val, _ := row.BonusValue.Float64Value()

		bonuses = append(bonuses, domain.JobLevelBonus{
			ID:          int(row.ID),
			JobID:       int(row.JobID),
			MinLevel:    int(row.MinLevel),
			BonusType:   row.BonusType,
			BonusValue:  val.Float64,
			Description: row.Description.String,
		})
	}

	return bonuses, nil
}

// ResetDailyJobXP resets the xp_gained_today counter for all users
// This should be called by a daily cron job
func (r *JobRepository) ResetDailyJobXP(ctx context.Context) error {
	result, err := r.q.ResetDailyJobXP(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset daily XP: %w", err)
	}

	rows := result.RowsAffected()
	fmt.Printf("Reset daily XP for %d user-job records\n", rows)

	return nil
}
