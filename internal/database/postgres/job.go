package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// JobRepository implements the job repository for PostgreSQL
type JobRepository struct {
	db *pgxpool.Pool
}

// NewJobRepository creates a new JobRepository
func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{db: db}
}

// GetAllJobs retrieves all job definitions
func (r *JobRepository) GetAllJobs(ctx context.Context) ([]domain.Job, error) {
	query := `
		SELECT id, job_key, display_name, description, associated_features, created_at
		FROM jobs
		ORDER BY id
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var job domain.Job
		var features []string
		err := rows.Scan(
			&job.ID,
			&job.JobKey,
			&job.DisplayName,
			&job.Description,
			&features,
			&job.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		job.AssociatedFeatures = features
		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return jobs, nil
}

// GetJobByKey retrieves a job by its key
func (r *JobRepository) GetJobByKey(ctx context.Context, jobKey string) (*domain.Job, error) {
	query := `
		SELECT id, job_key, display_name, description, associated_features, created_at
		FROM jobs
		WHERE job_key = $1
	`

	var job domain.Job
	var features []string
	err := r.db.QueryRow(ctx, query, jobKey).Scan(
		&job.ID,
		&job.JobKey,
		&job.DisplayName,
		&job.Description,
		&features,
		&job.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", jobKey)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	job.AssociatedFeatures = features
	return &job, nil
}

// GetUserJobs retrieves all job progress for a user
func (r *JobRepository) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJob, error) {
	query := `
		SELECT user_id, job_id, current_xp, current_level, xp_gained_today, last_xp_gain
		FROM user_jobs
		WHERE user_id = $1
		ORDER BY current_level DESC, current_xp DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user jobs: %w", err)
	}
	defer rows.Close()

	var userJobs []domain.UserJob
	for rows.Next() {
		var uj domain.UserJob
		err := rows.Scan(
			&uj.UserID,
			&uj.JobID,
			&uj.CurrentXP,
			&uj.CurrentLevel,
			&uj.XPGainedToday,
			&uj.LastXPGain,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user job: %w", err)
		}
		userJobs = append(userJobs, uj)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return userJobs, nil
}

// GetUserJob retrieves a single user's progress for a specific job
func (r *JobRepository) GetUserJob(ctx context.Context, userID string, jobID int) (*domain.UserJob, error) {
	query := `
		SELECT user_id, job_id, current_xp, current_level, xp_gained_today, last_xp_gain
		FROM user_jobs
		WHERE user_id = $1 AND job_id = $2
	`

	var uj domain.UserJob
	err := r.db.QueryRow(ctx, query, userID, jobID).Scan(
		&uj.UserID,
		&uj.JobID,
		&uj.CurrentXP,
		&uj.CurrentLevel,
		&uj.XPGainedToday,
		&uj.LastXPGain,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // Not an error, just no progress yet
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user job: %w", err)
	}

	return &uj, nil
}

// UpsertUserJob creates or updates a user's job progress
func (r *JobRepository) UpsertUserJob(ctx context.Context, userJob *domain.UserJob) error {
	query := `
		INSERT INTO user_jobs (user_id, job_id, current_xp, current_level, xp_gained_today, last_xp_gain)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, job_id)
		DO UPDATE SET
			current_xp = EXCLUDED.current_xp,
			current_level = EXCLUDED.current_level,
			xp_gained_today = EXCLUDED.xp_gained_today,
			last_xp_gain = EXCLUDED.last_xp_gain
	`

	_, err := r.db.Exec(ctx, query,
		userJob.UserID,
		userJob.JobID,
		userJob.CurrentXP,
		userJob.CurrentLevel,
		userJob.XPGainedToday,
		userJob.LastXPGain,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert user job: %w", err)
	}

	return nil
}

// RecordJobXPEvent logs an XP gain event
func (r *JobRepository) RecordJobXPEvent(ctx context.Context, event *domain.JobXPEvent) error {
	query := `
		INSERT INTO job_xp_events (id, user_id, job_id, xp_amount, source_type, source_metadata, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	metadataJSON, err := json.Marshal(event.SourceMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.Exec(ctx, query,
		event.ID,
		event.UserID,
		event.JobID,
		event.XPAmount,
		event.SourceType,
		metadataJSON,
		event.RecordedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to record XP event: %w", err)
	}

	return nil
}

// GetJobLevelBonuses retrieves bonuses for a job at or below a given level
func (r *JobRepository) GetJobLevelBonuses(ctx context.Context, jobID int, level int) ([]domain.JobLevelBonus, error) {
	query := `
		SELECT id, job_id, min_level, bonus_type, bonus_value, description
		FROM job_level_bonuses
		WHERE job_id = $1 AND min_level <= $2
		ORDER BY min_level DESC
	`

	rows, err := r.db.Query(ctx, query, jobID, level)
	if err != nil {
		return nil, fmt.Errorf("failed to query job bonuses: %w", err)
	}
	defer rows.Close()

	var bonuses []domain.JobLevelBonus
	for rows.Next() {
		var bonus domain.JobLevelBonus
		err := rows.Scan(
			&bonus.ID,
			&bonus.JobID,
			&bonus.MinLevel,
			&bonus.BonusType,
			&bonus.BonusValue,
			&bonus.Description,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bonus: %w", err)
		}
		bonuses = append(bonuses, bonus)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return bonuses, nil
}

// ResetDailyJobXP resets the xp_gained_today counter for all users
// This should be called by a daily cron job
func (r *JobRepository) ResetDailyJobXP(ctx context.Context) error {
	query := `
		UPDATE user_jobs
		SET xp_gained_today = 0
	`

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to reset daily XP: %w", err)
	}

	rows := result.RowsAffected()
	fmt.Printf("Reset daily XP for %d user-job records\n", rows)

	return nil
}
