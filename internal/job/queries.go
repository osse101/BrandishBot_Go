package job

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// GetAllJobs returns all available jobs
func (s *service) GetAllJobs(ctx context.Context) ([]domain.Job, error) {
	return s.repo.GetAllJobs(ctx)
}

// GetUserByPlatformID returns a user by their platform ID
func (s *service) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return s.repo.GetUserByPlatformID(ctx, platform, platformID)
}

// GetUserJobsByPlatform returns all jobs with user progress
func (s *service) GetUserJobsByPlatform(ctx context.Context, platform, platformID string) ([]domain.UserJobInfo, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return s.GetUserJobs(ctx, user.ID)
}

// GetUserJobsByUserID returns all jobs with user progress
func (s *service) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJobInfo, error) {
	log := logger.FromContext(ctx)

	// Get all jobs
	jobs, err := s.repo.GetAllJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}

	// Get user progress for all jobs
	userJobs, err := s.repo.GetUserJobs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user jobs: %w", err)
	}

	// Create map for quick lookup
	progressMap := make(map[int]*domain.UserJob)
	for i := range userJobs {
		progressMap[userJobs[i].JobID] = &userJobs[i]
	}

	// Get max level from progression system
	maxLevel := s.getMaxJobLevel(ctx)

	// Combine job info with progress
	result := make([]domain.UserJobInfo, 0, len(jobs))
	for _, job := range jobs {
		// Check if job is unlocked in progression tree
		// Jobs are nodes in the tree (e.g. "job_blacksmith")
		unlocked, err := s.progressionSvc.IsNodeUnlocked(ctx, job.JobKey, 1)
		if err != nil {
			log.Warn("Failed to check job unlock status", "error", err, "job", job.JobKey)
			continue
		}

		if !unlocked {
			continue
		}

		progress := progressMap[job.ID]
		info := domain.UserJobInfo{
			JobKey:      job.JobKey,
			DisplayName: job.DisplayName,
			Level:       0,
			CurrentXP:   0,
			MaxLevel:    maxLevel,
		}

		if progress != nil {
			info.Level = progress.CurrentLevel
			info.CurrentXP = progress.CurrentXP
			_, xpToNext := s.GetXPProgress(progress.CurrentXP)
			info.XPToNextLevel = xpToNext
		} else {
			info.XPToNextLevel = s.GetXPForLevel(1)
		}

		result = append(result, info)
	}

	return result, nil
}

// GetPrimaryJob returns the user's highest-level job
func (s *service) GetPrimaryJob(ctx context.Context, platform string, platformID string) (*domain.UserJobInfo, error) {
	userJobs, err := s.GetUserJobsByPlatform(ctx, platform, platformID)
	if err != nil {
		return nil, err
	}

	if len(userJobs) == 0 {
		return nil, nil
	}

	// Find job with highest level
	var primary *domain.UserJobInfo
	for i := range userJobs {
		job := &userJobs[i]
		if primary == nil {
			primary = job
			continue
		}

		if job.Level > primary.Level {
			primary = job
		} else if job.Level == primary.Level {
			// Tie identifier: most XP
			if job.CurrentXP > primary.CurrentXP {
				primary = job
			}
			// Further tie-breaking (e.g. alphabetical) could go here
		}
	}

	return primary, nil
}

// GetJobBonus returns the bonus value for a specific job and bonus type
func (s *service) GetJobBonus(ctx context.Context, userID, jobKey, bonusType string) (float64, error) {
	level, err := s.GetJobLevel(ctx, userID, jobKey)
	if err != nil || level == 0 {
		return 0, err
	}

	job, err := s.repo.GetJobByKey(ctx, jobKey)
	if err != nil {
		return 0, err
	}

	bonuses, err := s.repo.GetJobLevelBonuses(ctx, job.ID, level)
	if err != nil {
		return 0, err
	}

	// Find the highest applicable bonus of the requested type
	var bestBonus float64
	for _, bonus := range bonuses {
		if bonus.BonusType == bonusType && bonus.BonusValue > bestBonus {
			bestBonus = bonus.BonusValue
		}
	}

	return bestBonus, nil
}

func (s *service) GetUserJobByPlatform(ctx context.Context, platform, platformID, jobKey string) (*domain.UserJob, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, err
	}
	job, err := s.repo.GetJobByKey(ctx, jobKey)
	if err != nil {
		return nil, err
	}
	return s.repo.GetUserJob(ctx, user.ID, job.ID)
}

// GetJobLevel returns the user's level for a specific job
func (s *service) GetJobLevel(ctx context.Context, userID, jobKey string) (int, error) {
	job, err := s.repo.GetJobByKey(ctx, jobKey)
	if err != nil {
		return 0, err
	}

	progress, err := s.repo.GetUserJob(ctx, userID, job.ID)
	if err != nil {
		return 0, err
	}
	if progress == nil {
		return 0, nil
	}

	return progress.CurrentLevel, nil
}
