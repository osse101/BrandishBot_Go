package repository

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Job defines the data access interface for job operations
type Job interface {
	GetAllJobs(ctx context.Context) ([]domain.Job, error)
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	GetJobByKey(ctx context.Context, jobKey string) (*domain.Job, error)
	GetUserJobs(ctx context.Context, userID string) ([]domain.UserJob, error)
	GetUserJob(ctx context.Context, userID string, jobID int) (*domain.UserJob, error)
	GetUserJobsByPlatform(ctx context.Context, platform, platformID string) ([]domain.UserJob, error)
	UpsertUserJob(ctx context.Context, userJob *domain.UserJob) error
	RecordJobXPEvent(ctx context.Context, event *domain.JobXPEvent) error
	GetJobLevelBonuses(ctx context.Context, jobID int, level int) ([]domain.JobLevelBonus, error)
}
