package job

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// AwardXP awards XP to a user for a specific job
func (s *service) AwardXP(ctx context.Context, userID string, jobKey string, baseAmount int, source string, metadata domain.JobXPMetadata) (*domain.XPAwardResult, error) {
	// Check if specific job is unlocked
	jobUnlocked, err := s.progressionSvc.IsNodeUnlocked(ctx, jobKey, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to check job unlock: %w", err)
	}
	if !jobUnlocked {
		return nil, fmt.Errorf("job %s is not unlocked: %w", jobKey, domain.ErrFeatureLocked)
	}

	job, err := s.repo.GetJobByKey(ctx, jobKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	currentProgress, err := s.getOrCreateUserJob(ctx, userID, job.ID)
	if err != nil {
		return nil, err
	}

	actualAmount := s.calculateActualXP(ctx, userID, jobKey, baseAmount, source)

	if err := s.checkDailyCap(ctx, userID, jobKey, currentProgress, &actualAmount, source); err != nil {
		return nil, err
	}

	oldLevel := currentProgress.CurrentLevel
	newXP := currentProgress.CurrentXP + int64(actualAmount)
	newLevel := s.calculateNewLevel(ctx, newXP)

	now := time.Now()
	if err := s.updateUserJobProgress(ctx, currentProgress, newXP, newLevel, actualAmount, &now); err != nil {
		return nil, err
	}

	s.recordXPAndLevelUpEvents(ctx, userID, jobKey, job.ID, actualAmount, oldLevel, newLevel, source, metadata, &now)

	return &domain.XPAwardResult{
		JobKey:    jobKey,
		XPGained:  actualAmount,
		NewXP:     newXP,
		NewLevel:  newLevel,
		LeveledUp: newLevel > oldLevel,
	}, nil
}

func (s *service) AwardXPByPlatform(ctx context.Context, platform string, platformID string, jobKey string, baseAmount int, source string, metadata domain.JobXPMetadata) (*domain.XPAwardResult, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.AwardXP(ctx, user.ID, jobKey, baseAmount, source, metadata)
}

func (s *service) getOrCreateUserJob(ctx context.Context, userID string, jobID int) (*domain.UserJob, error) {
	currentProgress, err := s.repo.GetUserJob(ctx, userID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user job: %w", err)
	}
	if currentProgress == nil {
		currentProgress = &domain.UserJob{
			UserID:        userID,
			JobID:         jobID,
			CurrentXP:     0,
			CurrentLevel:  0,
			XPGainedToday: 0,
		}
	}
	return currentProgress, nil
}

func (s *service) calculateActualXP(ctx context.Context, userID, jobKey string, baseAmount int, source string) int {
	xpMultiplier := s.getXPMultiplier(ctx)
	actualAmount := int(float64(baseAmount) * xpMultiplier)

	if s.rnd() < EpiphanyChance {
		actualAmount = int(float64(actualAmount) * EpiphanyMultiplier)
		logger.FromContext(ctx).Info("Job Epiphany triggered!", "user_id", userID, "job", jobKey, "base_amount", baseAmount, "bonus_amount", actualAmount-baseAmount)
		if s.publisher != nil {
			s.publisher.PublishWithRetry(ctx, event.NewJobXPCriticalEvent(userID, jobKey, baseAmount, actualAmount-baseAmount, EpiphanyMultiplier, source))
		}
	}
	return actualAmount
}

func (s *service) checkDailyCap(ctx context.Context, userID, jobKey string, currentProgress *domain.UserJob, actualAmount *int, source string) error {
	// Skip daily cap for rare candy and harvest
	if source == SourceRareCandy || source == SourceHarvest {
		logger.FromContext(ctx).Info("Bypassing daily XP cap",
			"user_id", userID, "job", jobKey, "xp", *actualAmount, "source", source)
		return nil
	}

	dailyCap := s.getDailyCap(ctx)
	if currentProgress.XPGainedToday+int64(*actualAmount) > int64(dailyCap) {
		remaining := int64(dailyCap) - currentProgress.XPGainedToday
		if remaining <= 0 {
			logger.FromContext(ctx).Info("Daily XP cap reached", "user_id", userID, "job", jobKey)
			return fmt.Errorf("daily XP cap reached for %s: %w", jobKey, domain.ErrDailyCapReached)
		}
		*actualAmount = int(remaining)
	}
	return nil
}

func (s *service) calculateNewLevel(ctx context.Context, newXP int64) int {
	newLevel := s.CalculateLevel(newXP)
	maxLevel := s.getMaxJobLevel(ctx)
	if newLevel > maxLevel {
		newLevel = maxLevel
	}
	return newLevel
}

func (s *service) updateUserJobProgress(ctx context.Context, progress *domain.UserJob, newXP int64, newLevel int, actualAmount int, now *time.Time) error {
	progress.CurrentXP = newXP
	progress.CurrentLevel = newLevel
	progress.XPGainedToday += int64(actualAmount)
	progress.LastXPGain = now

	err := s.repo.UpsertUserJob(ctx, progress)
	if err != nil {
		return fmt.Errorf("failed to update user job: %w", err)
	}
	return nil
}

func (s *service) recordXPAndLevelUpEvents(ctx context.Context, userID, jobKey string, jobID int, actualAmount int, oldLevel, newLevel int, source string, metadata domain.JobXPMetadata, now *time.Time) {
	log := logger.FromContext(ctx)

	// Record XP event
	xpEvent := &domain.JobXPEvent{
		ID:             uuid.New(),
		UserID:         userID,
		JobID:          jobID,
		XPAmount:       actualAmount,
		SourceType:     source,
		SourceMetadata: metadata,
		RecordedAt:     *now,
	}
	if err := s.repo.RecordJobXPEvent(ctx, xpEvent); err != nil {
		log.Error("Failed to record XP event", "error", err)
	}

	log.Info("Awarded job XP", "user_id", userID, "job", jobKey, "xp", actualAmount, "new_level", newLevel, "leveled_up", newLevel > oldLevel)

	if newLevel > oldLevel {
		s.handleLevelUp(ctx, userID, jobKey, oldLevel, newLevel, source)
	}
}

func (s *service) handleLevelUp(ctx context.Context, userID, jobKey string, oldLevel, newLevel int, source string) {
	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.NewJobLevelUpEvent(userID, jobKey, oldLevel, newLevel, source))
	}
}

func (s *service) getXPMultiplier(ctx context.Context) float64 {
	// Apply progression modifier for job XP multiplier
	modified, err := s.progressionSvc.GetModifiedValue(ctx, "job_xp_multiplier", 1.0)
	if err != nil {
		log := logger.FromContext(ctx)
		log.Warn("Failed to get job XP multiplier, using default", "error", err)
		return 1.0 // Fallback to no multiplier
	}
	return modified
}

func (s *service) getDailyCap(ctx context.Context) int {
	// Apply progression modifier for daily job cap
	modified, err := s.progressionSvc.GetModifiedValue(ctx, "job_daily_cap", float64(DefaultDailyCap))
	if err != nil {
		log := logger.FromContext(ctx)
		log.Warn("Failed to get daily cap modifier, using default", "error", err)
		return DefaultDailyCap
	}
	return int(modified)
}

func (s *service) getMaxJobLevel(ctx context.Context) int {
	// Apply progression modifier for job level cap (linear: +10 per level)
	// Base cap is DefaultMaxLevel, upgrade_job_level_cap adds +10 per level (max +30 at level 3)
	modified, err := s.progressionSvc.GetModifiedValue(ctx, "job_level_cap", float64(DefaultMaxLevel))
	if err != nil {
		log := logger.FromContext(ctx)
		log.Warn("Failed to get job level cap modifier, using default", "error", err)
		return DefaultMaxLevel
	}
	return int(modified)
}
