package job

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// ProgressionService defines the interface for progression system
type ProgressionService interface {
	IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error)
	GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
	IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error)
}

// Service defines the job system business logic
type Service interface {
	// Core operations
	GetUserJobs(ctx context.Context, userID string) ([]domain.UserJobInfo, error)
	GetUserJobsByPlatform(ctx context.Context, platform, platformID string) ([]domain.UserJobInfo, error)
	GetPrimaryJob(ctx context.Context, platform, platformID string) (*domain.UserJobInfo, error)
	GetJobBonus(ctx context.Context, userID, jobKey string, bonusType string) (float64, error)

	// XP operations
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
	AwardXPByPlatform(ctx context.Context, platform, platformID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
	GetJobLevel(ctx context.Context, userID, jobKey string) (int, error)

	// Daily reset operations
	ResetDailyJobXP(ctx context.Context) (int64, error)
	GetDailyResetStatus(ctx context.Context) (*domain.DailyResetStatus, error)

	// Utility
	GetAllJobs(ctx context.Context) ([]domain.Job, error)
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	CalculateLevel(totalXP int64) int
	GetXPForLevel(level int) int64
	GetXPProgress(currentXP int64) (currentLevel int, xpToNext int64)
}

type service struct {
	repo           repository.Job
	progressionSvc ProgressionService
	statsSvc       stats.Service
	eventBus       event.Bus
	publisher      *event.ResilientPublisher
	rnd            func() float64 // For RNG

	// Cache for daily reset status
	resetCache   *domain.DailyResetStatus
	resetCacheMu sync.RWMutex
}

// NewService creates a new job service
func NewService(repo repository.Job, progressionSvc ProgressionService, statsSvc stats.Service, eventBus event.Bus, publisher *event.ResilientPublisher) Service {
	return &service{
		repo:           repo,
		progressionSvc: progressionSvc,
		statsSvc:       statsSvc,
		eventBus:       eventBus,
		publisher:      publisher,
		rnd:            utils.RandomFloat,
	}
}

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
	maxLevel, err := s.getMaxJobLevel(ctx)
	if err != nil {
		log.Warn("Failed to get max job level, defaulting to default", "error", err)
		maxLevel = DefaultMaxLevel
	}

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

// AwardXP awards XP to a user for a specific job
func (s *service) AwardXP(ctx context.Context, userID string, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {

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

func (s *service) AwardXPByPlatform(ctx context.Context, platform string, platformID string, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
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
		if s.statsSvc != nil {
			_ = s.statsSvc.RecordUserEvent(ctx, userID, domain.EventJobXPCritical, map[string]interface{}{
				"job":        jobKey,
				"base_xp":    baseAmount,
				"bonus_xp":   actualAmount - baseAmount,
				"multiplier": EpiphanyMultiplier,
				"source":     source,
			})
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
	maxLevel, err := s.getMaxJobLevel(ctx)
	if err != nil {
		logger.FromContext(ctx).Warn("Failed to get max level, using default", "error", err)
		maxLevel = DefaultMaxLevel
	}
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

func (s *service) recordXPAndLevelUpEvents(ctx context.Context, userID, jobKey string, jobID int, actualAmount int, oldLevel, newLevel int, source string, metadata map[string]interface{}, now *time.Time) {
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
	if s.statsSvc != nil {
		_ = s.statsSvc.RecordUserEvent(ctx, userID, domain.EventJobLevelUp, map[string]interface{}{
			"job":       jobKey,
			"level":     newLevel,
			"old_level": oldLevel,
		})
	}

	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventJobLevelUp),
			Payload: map[string]interface{}{
				"user_id":   userID,
				"job_key":   jobKey,
				"new_level": newLevel,
				"old_level": oldLevel,
			},
			Metadata: map[string]interface{}{
				"source": source,
			},
		})
	}
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

// CalculateLevel determines the level from total XP using the formula:
// XP for level N = BaseXP * (N ^ LevelExponent)
func (s *service) CalculateLevel(totalXP int64) int {
	level, _ := s.calculateLevelAndNextXP(totalXP)
	return level
}

// GetXPForLevel returns the XP required to reach a specific level from level 0
func (s *service) GetXPForLevel(level int) int64 {
	if level <= 0 {
		return 0
	}

	cumulative := int64(0)
	for i := 1; i <= level; i++ {
		cumulative += int64(BaseXP * math.Pow(float64(i), LevelExponent))
	}

	return cumulative
}

// GetXPProgress returns current level and XP needed for next level
func (s *service) GetXPProgress(currentXP int64) (currentLevel int, xpToNext int64) {
	var xpForNext int64
	currentLevel, xpForNext = s.calculateLevelAndNextXP(currentXP)
	xpToNext = xpForNext - currentXP
	return
}

// calculateLevelAndNextXP computes the level and the cumulative XP required for the NEXT level
// This optimized helper avoids double iteration in GetXPProgress
func (s *service) calculateLevelAndNextXP(totalXP int64) (int, int64) {
	if totalXP <= 0 {
		return 0, int64(BaseXP)
	}

	level := 0
	cumulative := int64(0)

	for level < MaxIterationLevel {
		nextLevel := level + 1
		xpForNextLevel := int64(BaseXP * math.Pow(float64(nextLevel), LevelExponent))

		if cumulative+xpForNextLevel > totalXP {
			return level, cumulative + xpForNextLevel
		}
		cumulative += xpForNextLevel
		level = nextLevel
	}

	// Max level reached, calculate theoretical next level requirement
	nextLevel := level + 1
	xpForNextLevel := int64(BaseXP * math.Pow(float64(nextLevel), LevelExponent))
	return level, cumulative + xpForNextLevel
}

// Helper functions (TODO: integrate with progression system)

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

func (s *service) getMaxJobLevel(ctx context.Context) (int, error) {
	// Apply progression modifier for job level cap (linear: +10 per level)
	// Base cap is DefaultMaxLevel, upgrade_job_level_cap adds +10 per level (max +30 at level 3)
	modified, err := s.progressionSvc.GetModifiedValue(ctx, "job_level_cap", float64(DefaultMaxLevel))
	if err != nil {
		log := logger.FromContext(ctx)
		log.Warn("Failed to get job level cap modifier, using default", "error", err)
		return DefaultMaxLevel, nil
	}
	return int(modified), nil
}

// ResetDailyJobXP resets the daily XP counters for all users
// Returns the number of records affected
func (s *service) ResetDailyJobXP(ctx context.Context) (int64, error) {
	log := logger.FromContext(ctx)
	log.Info("Starting daily job XP reset")

	recordsAffected, err := s.repo.ResetDailyJobXP(ctx)
	if err != nil {
		log.Error("Daily XP reset failed", "error", err)
		return 0, err
	}

	// Update the reset state in the database
	now := time.Now().UTC()
	if err := s.repo.UpdateDailyResetTime(ctx, now, recordsAffected); err != nil {
		log.Warn("Failed to update reset state", "error", err)
		// Don't fail the reset operation itself, just warn
	} else {
		// Update cache
		s.resetCacheMu.Lock()
		s.resetCache = &domain.DailyResetStatus{
			LastResetTime:   now,
			RecordsAffected: recordsAffected,
		}
		s.resetCacheMu.Unlock()
	}

	log.Info("Daily XP reset completed", "records_affected", recordsAffected)

	// Publish event
	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeDailyResetComplete),
			Payload: map[string]interface{}{
				"reset_time":       time.Now().UTC(),
				"records_affected": recordsAffected,
			},
		})
	}

	return recordsAffected, nil
}

// GetDailyResetStatus returns information about the daily reset state
func (s *service) GetDailyResetStatus(ctx context.Context) (*domain.DailyResetStatus, error) {
	s.resetCacheMu.RLock()
	cache := s.resetCache
	s.resetCacheMu.RUnlock()

	var lastReset time.Time
	var recordsAffected int64

	if cache != nil {
		lastReset = cache.LastResetTime
		recordsAffected = cache.RecordsAffected
	} else {
		var err error
		lastReset, recordsAffected, err = s.repo.GetLastDailyResetTime(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get reset status: %w", err)
		}

		// Fill cache
		s.resetCacheMu.Lock()
		s.resetCache = &domain.DailyResetStatus{
			LastResetTime:   lastReset,
			RecordsAffected: recordsAffected,
		}
		s.resetCacheMu.Unlock()
	}

	// Calculate next reset time (00:00 UTC+7)
	// UTC+7 is 7 hours ahead of UTC. 00:00 UTC+7 is 17:00 UTC of previous day.
	location := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(location)
	nextReset := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	if !nextReset.After(now) {
		nextReset = nextReset.AddDate(0, 0, 1)
	}

	return &domain.DailyResetStatus{
		LastResetTime:   lastReset,
		NextResetTime:   nextReset.UTC(),
		RecordsAffected: recordsAffected,
	}, nil
}
