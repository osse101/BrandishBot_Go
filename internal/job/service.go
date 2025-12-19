package job

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// Repository defines the data access interface for job operations
type Repository interface {
	GetAllJobs(ctx context.Context) ([]domain.Job, error)
	GetJobByKey(ctx context.Context, jobKey string) (*domain.Job, error)
	GetUserJobs(ctx context.Context, userID string) ([]domain.UserJob, error)
	GetUserJob(ctx context.Context, userID string, jobID int) (*domain.UserJob, error)
	UpsertUserJob(ctx context.Context, userJob *domain.UserJob) error
	RecordJobXPEvent(ctx context.Context, event *domain.JobXPEvent) error
	GetJobLevelBonuses(ctx context.Context, jobID int, level int) ([]domain.JobLevelBonus, error)
}

// ProgressionService defines the interface for progression system
type ProgressionService interface {
	IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error)
	GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error)
}

// Service defines the job system business logic
type Service interface {
	// Core operations
	GetAllJobs(ctx context.Context) ([]domain.Job, error)
	GetUserJobs(ctx context.Context, userID string) ([]domain.UserJobInfo, error)
	GetPrimaryJob(ctx context.Context, userID string) (*domain.UserJobInfo, error)

	// XP operations
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
	GetJobLevel(ctx context.Context, userID, jobKey string) (int, error)
	GetJobBonus(ctx context.Context, userID, jobKey, bonusType string) (float64, error)

	// Level calculations
	CalculateLevel(totalXP int64) int
	GetXPForLevel(level int) int64
	GetXPProgress(currentXP int64) (currentLevel int, xpToNext int64)
}

type service struct {
	repo           Repository
	progressionSvc ProgressionService
}

// NewService creates a new job service
func NewService(repo Repository, progressionSvc ProgressionService) Service {
	return &service{
		repo:           repo,
		progressionSvc: progressionSvc,
	}
}

// GetAllJobs returns all job definitions
func (s *service) GetAllJobs(ctx context.Context) ([]domain.Job, error) {
	return s.repo.GetAllJobs(ctx)
}

// GetUserJobs returns all jobs with user progress
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
	var result []domain.UserJobInfo
	for _, job := range jobs {
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
func (s *service) GetPrimaryJob(ctx context.Context, userID string) (*domain.UserJobInfo, error) {
	userJobs, err := s.GetUserJobs(ctx, userID)
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
func (s *service) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
	log := logger.FromContext(ctx)

	// Check if jobs_xp feature is unlocked
	unlocked, err := s.progressionSvc.IsFeatureUnlocked(ctx, "feature_jobs_xp")
	if err != nil {
		return nil, fmt.Errorf("failed to check jobs_xp unlock: %w", err)
	}
	if !unlocked {
		log.Debug("Jobs XP system not unlocked yet")
		return nil, fmt.Errorf("jobs XP system not unlocked")
	}

	// Get job
	job, err := s.repo.GetJobByKey(ctx, jobKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	// Get current progress
	currentProgress, err := s.repo.GetUserJob(ctx, userID, job.ID)
	if currentProgress == nil {
		// Initialize new user job
		currentProgress = &domain.UserJob{
			UserID:        userID,
			JobID:         job.ID,
			CurrentXP:     0,
			CurrentLevel:  0,
			XPGainedToday: 0,
		}
	}

	// Apply XP boost multiplier (TODO: get from progression system)
	xpMultiplier := s.getXPMultiplier(ctx)
	actualAmount := int(float64(baseAmount) * xpMultiplier)

	// Check daily cap (TODO: get from progression system)
	dailyCap := s.getDailyCap(ctx)
	if currentProgress.XPGainedToday+int64(actualAmount) > int64(dailyCap) {
		remaining := int64(dailyCap) - currentProgress.XPGainedToday
		if remaining <= 0 {
			log.Info("Daily XP cap reached", "user_id", userID, "job", jobKey)
			return nil, fmt.Errorf("daily XP cap reached for %s", jobKey)
		}
		actualAmount = int(remaining)
	}

	oldLevel := currentProgress.CurrentLevel
	newXP := currentProgress.CurrentXP + int64(actualAmount)
	newLevel := s.CalculateLevel(newXP)

	// Get max level cap
	maxLevel, err := s.getMaxJobLevel(ctx)
	if err != nil {
		log.Warn("Failed to get max level, using default", "error", err)
		maxLevel = DefaultMaxLevel
	}
	if newLevel > maxLevel {
		newLevel = maxLevel
	}

	// Update progress
	now := time.Now()
	currentProgress.CurrentXP = newXP
	currentProgress.CurrentLevel = newLevel
	currentProgress.XPGainedToday += int64(actualAmount)
	currentProgress.LastXPGain = &now

	err = s.repo.UpsertUserJob(ctx, currentProgress)
	if err != nil {
		return nil, fmt.Errorf("failed to update user job: %w", err)
	}

	// Record event
	event := &domain.JobXPEvent{
		ID:             uuid.New(),
		UserID:         userID,
		JobID:          job.ID,
		XPAmount:       actualAmount,
		SourceType:     source,
		SourceMetadata: metadata,
		RecordedAt:     now,
	}
	err = s.repo.RecordJobXPEvent(ctx, event)
	if err != nil {
		log.Error("Failed to record XP event", "error", err)
		// Don't fail the operation if logging fails
	}

	log.Info("Awarded job XP",
		"user_id", userID,
		"job", jobKey,
		"xp", actualAmount,
		"new_level", newLevel,
		"leveled_up", newLevel > oldLevel,
	)

	return &domain.XPAwardResult{
		JobKey:    jobKey,
		XPGained:  actualAmount,
		NewXP:     newXP,
		NewLevel:  newLevel,
		LeveledUp: newLevel > oldLevel,
	}, nil
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
	// TODO: Query jobs_xp_boost node level from progression system
	// For now, return default (no boost)
	return DefaultXPMultiplier
}

func (s *service) getDailyCap(ctx context.Context) int {
	// TODO: Scale with jobs_xp_boost node level
	return DefaultDailyCap
}

func (s *service) getMaxJobLevel(ctx context.Context) (int, error) {
	// TODO: Get from jobs_xp node unlock level
	// Level 1 = max 10, Level 2 = max 20, etc.
	return DefaultMaxLevel, nil
}
