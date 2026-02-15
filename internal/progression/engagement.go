package progression

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// RecordEngagement records user engagement event
func (s *service) RecordEngagement(ctx context.Context, userID string, metricType string, value int) error {
	metric := &domain.EngagementMetric{
		UserID:      userID,
		MetricType:  metricType,
		MetricValue: value,
		RecordedAt:  time.Now(),
	}

	if err := s.repo.RecordEngagement(ctx, metric); err != nil {
		return err
	}

	// Try to get weights from cache first
	weight := s.getCachedWeight(metricType)

	// If not in cache or expired, fetch from DB
	if weight == 0.0 {
		weights, err := s.repo.GetEngagementWeights(ctx)
		if err != nil {
			// Log warning but don't fail, use default weight of 1.0 if not found
			logger.FromContext(ctx).Warn("Failed to get engagement weights, using default", "error", err)
			// We could fallback to hardcoded defaults here if critical
		} else {
			// Cache weights for future use (5 minute TTL)
			s.cacheWeights(weights)
			if w, ok := weights[metricType]; ok {
				weight = w
			}
		}
	}

	// Fallback defaults if still no weight found
	if weight == 0.0 {
		switch metricType {
		case "message":
			weight = 1.0
		case "command":
			weight = 2.0
		case "item_crafted":
			weight = 3.0 // Note: Migration sets this to 200, this is just code fallback
		default:
			weight = 1.0 // Safe default
		}
	}

	// If we have a weight, calculate score
	if weight > 0 {
		baseScore := float64(value) * weight

		// Apply progression rate modifier (stacks multiplicatively across all three upgrades)
		// upgrade_progression_basic, upgrade_progression_two, upgrade_progression_three
		modifiedScore, err := s.GetModifiedValue(ctx, "progression_rate", baseScore)
		if err != nil {
			// Log warning but continue with base score if modifier fails
			logger.FromContext(ctx).Warn("Failed to apply progression_rate modifier, using base score", "error", err)
			modifiedScore = baseScore
		}

		// Apply Scholar bonus (per-user contribution multiplier)
		scholarMultiplier := s.calculateScholarBonus(ctx, userID)
		if scholarMultiplier > 1.0 {
			log := logger.FromContext(ctx)
			log.Info("Applying Scholar bonus",
				"user_id", userID,
				"base_score", modifiedScore,
				"multiplier", scholarMultiplier)
			modifiedScore = modifiedScore * scholarMultiplier
		}

		score := int(modifiedScore)
		if score > 0 {
			if err := s.AddContribution(ctx, score); err != nil {
				logger.FromContext(ctx).Warn("Failed to add contribution from engagement", "error", err)
			}
		}
	}

	return nil
}

// GetEngagementScore returns total community engagement score
func (s *service) GetEngagementScore(ctx context.Context) (int, error) {
	// Get score since last unlock (or beginning)
	return s.repo.GetEngagementScore(ctx, nil)
}

// calculateScholarBonus calculates the contribution multiplier from Scholar job
// Returns 1.0 + (level × 0.10), e.g., level 5 = 1.5x multiplier
func (s *service) calculateScholarBonus(ctx context.Context, userID string) float64 {
	log := logger.FromContext(ctx)

	if s.jobService == nil {
		return 1.0
	}

	level, err := s.jobService.GetJobLevel(ctx, userID, job.JobKeyScholar)
	if err != nil {
		log.Warn("Failed to get Scholar level, using 1.0x multiplier", "error", err)
		return 1.0
	}

	if level == 0 {
		return 1.0
	}

	// 10% bonus per level
	multiplier := 1.0 + (float64(level) * job.ScholarBonusPerLevel / 100.0)
	return multiplier
}

// GetUserEngagement returns user's contribution breakdown
func (s *service) GetUserEngagement(ctx context.Context, platform, platformID string) (*domain.ContributionBreakdown, error) {
	// Convert platform_id to internal user ID
	user, err := s.user.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.repo.GetUserEngagement(ctx, user.ID)
}

// GetUserEngagementByUsername returns user's contribution breakdown by username
func (s *service) GetUserEngagementByUsername(ctx context.Context, platform, username string) (*domain.ContributionBreakdown, error) {
	// Convert username to internal user ID
	user, err := s.user.GetUserByPlatformUsername(ctx, platform, username)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.repo.GetUserEngagement(ctx, user.ID)
}

// GetContributionLeaderboard retrieves top contributors
func (s *service) GetContributionLeaderboard(ctx context.Context, limit int) ([]domain.ContributionLeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 10 // Default to top 10
	}
	return s.repo.GetContributionLeaderboard(ctx, limit)
}

// GetEngagementVelocity calculates engagement velocity over a period
func (s *service) GetEngagementVelocity(ctx context.Context, days int) (*domain.VelocityMetrics, error) {
	if days <= 0 {
		days = 7
	}

	since := time.Now().AddDate(0, 0, -days)
	totals, err := s.repo.GetDailyEngagementTotals(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily totals: %w", err)
	}

	totalPoints := 0
	sampleSize := len(totals)

	if sampleSize == 0 {
		return &domain.VelocityMetrics{
			PointsPerDay: 0,
			Trend:        "stable",
			PeriodDays:   days,
			SampleSize:   0,
			TotalPoints:  0,
		}, nil
	}

	orderedDays := make([]time.Time, 0, len(totals))
	for day, points := range totals {
		totalPoints += points
		orderedDays = append(orderedDays, day)
	}

	// Sort days
	for i := 0; i < len(orderedDays)-1; i++ {
		for j := 0; j < len(orderedDays)-i-1; j++ {
			if orderedDays[j].After(orderedDays[j+1]) {
				orderedDays[j], orderedDays[j+1] = orderedDays[j+1], orderedDays[j]
			}
		}
	}

	avg := float64(totalPoints) / float64(days)

	// Trend detection
	trend := "stable"
	if sampleSize >= 2 {
		half := sampleSize / 2
		firstHalfSum := 0
		secondHalfSum := 0

		for i := 0; i < half; i++ {
			firstHalfSum += totals[orderedDays[i]]
		}
		for i := half; i < sampleSize; i++ {
			secondHalfSum += totals[orderedDays[i]]
		}

		firstHalfAvg := float64(firstHalfSum) / float64(half)
		secondHalfAvg := float64(secondHalfSum) / float64(sampleSize-half)

		if secondHalfAvg > firstHalfAvg*1.1 {
			trend = "increasing"
		} else if secondHalfAvg < firstHalfAvg*0.9 {
			trend = "decreasing"
		}
	}

	return &domain.VelocityMetrics{
		PointsPerDay: avg,
		Trend:        trend,
		PeriodDays:   days,
		SampleSize:   sampleSize,
		TotalPoints:  totalPoints,
	}, nil
}

// EstimateUnlockTime predicts when a node will unlock
func (s *service) EstimateUnlockTime(ctx context.Context, nodeKey string) (*domain.UnlockEstimate, error) {
	node, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	if node == nil {
		return nil, fmt.Errorf("node not found: %s", nodeKey)
	}

	// Get current velocity (7 days default)
	velocity, err := s.GetEngagementVelocity(ctx, 7)
	if err != nil {
		return nil, err
	}

	// Get current progress
	var currentProgress int
	progress, _ := s.repo.GetActiveUnlockProgress(ctx)
	if progress != nil && progress.NodeID != nil && *progress.NodeID == node.ID {
		currentProgress = progress.ContributionsAccumulated
	}

	// Check if already unlocked (max level)
	isUnlocked, _ := s.repo.IsNodeUnlocked(ctx, nodeKey, node.MaxLevel)
	if isUnlocked {
		return &domain.UnlockEstimate{
			NodeKey:             nodeKey,
			EstimatedDays:       0,
			Confidence:          "high",
			RequiredPoints:      0,
			CurrentProgress:     node.UnlockCost,
			CurrentVelocity:     velocity.PointsPerDay,
			EstimatedUnlockDate: func() *time.Time { t := time.Now(); return &t }(),
		}, nil
	}

	required := node.UnlockCost - currentProgress
	if required <= 0 {
		required = 0
	}

	var estimatedDays float64
	var estimatedDate *time.Time

	if velocity.PointsPerDay > 0 {
		estimatedDays = float64(required) / velocity.PointsPerDay
		t := time.Now().Add(time.Duration(estimatedDays * 24 * float64(time.Hour)))
		estimatedDate = &t
	} else {
		estimatedDays = -1 // Infinite
	}

	confidence := "low"
	if velocity.SampleSize >= 7 {
		if velocity.Trend == "stable" || velocity.Trend == "increasing" {
			confidence = "high"
		} else {
			confidence = "medium"
		}
	} else if velocity.SampleSize >= 3 {
		confidence = "medium"
	}

	return &domain.UnlockEstimate{
		NodeKey:             nodeKey,
		EstimatedDays:       estimatedDays,
		Confidence:          confidence,
		RequiredPoints:      required,
		CurrentProgress:     currentProgress,
		CurrentVelocity:     velocity.PointsPerDay,
		EstimatedUnlockDate: estimatedDate,
	}, nil
}

// getCachedWeight retrieves weight from cache if not expired
func (s *service) getCachedWeight(metricType string) float64 {
	s.weightsMu.RLock()
	defer s.weightsMu.RUnlock()

	// Check if cache is expired
	if time.Now().After(s.weightsExpiry) {
		return 0.0
	}

	if s.cachedWeights == nil {
		return 0.0
	}

	return s.cachedWeights[metricType]
}

// cacheWeights stores engagement weights with 5-minute TTL
func (s *service) cacheWeights(weights map[string]float64) {
	s.weightsMu.Lock()
	defer s.weightsMu.Unlock()

	s.cachedWeights = weights
	s.weightsExpiry = time.Now().Add(5 * time.Minute) // 5 min TTL - weights rarely change
}

// InvalidateWeightCache clears the engagement weight cache
func (s *service) InvalidateWeightCache() {
	s.weightsMu.Lock()
	defer s.weightsMu.Unlock()

	s.cachedWeights = nil
	s.weightsExpiry = time.Time{} // Zero time = always expired
}
