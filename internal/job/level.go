package job

import (
	"context"
	"fmt"
	"math"
)

// CalculateLevel determines the level from total XP using the formula:
// XP for level N = BaseXP * (N ^ LevelExponent)
func (s *service) CalculateLevel(totalXP int64) int {
	level, _, _ := s.calculateLevelAndNextXP(totalXP)
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

// GetXPProgress returns current level, XP progress within level, and total XP needed for that level
func (s *service) GetXPProgress(currentXP int64) (currentLevel int, levelXP int64, levelRequirement int64, xpToNext int64) {
	var xpForNext, currentLevelStart int64
	currentLevel, currentLevelStart, xpForNext = s.calculateLevelAndNextXP(currentXP)

	levelXP = currentXP - currentLevelStart
	levelRequirement = xpForNext - currentLevelStart
	xpToNext = xpForNext - currentXP
	return
}

// calculateLevelAndNextXP computes the level, cumulative XP for the CURRENT level start, and the cumulative XP for the NEXT level
// This optimized helper avoids double iteration in GetXPProgress
func (s *service) calculateLevelAndNextXP(totalXP int64) (int, int64, int64) {
	if totalXP <= 0 {
		return 0, 0, int64(BaseXP)
	}

	level := 0
	cumulative := int64(0)

	for level < MaxIterationLevel {
		nextLevel := level + 1
		xpForNextLevel := int64(BaseXP * math.Pow(float64(nextLevel), LevelExponent))

		if cumulative+xpForNextLevel > totalXP {
			return level, cumulative, cumulative + xpForNextLevel
		}
		cumulative += xpForNextLevel
		level = nextLevel
	}

	// Max level reached, calculate theoretical next level requirement
	nextLevel := level + 1
	xpForNextLevel := int64(BaseXP * math.Pow(float64(nextLevel), LevelExponent))
	return level, cumulative, cumulative + xpForNextLevel
}

// IsJobFeatureUnlocked determines if a user has the required job level to access a specific feature
func (s *service) IsJobFeatureUnlocked(ctx context.Context, userID string, featureKey string) (bool, error) {
	// First, lookup the required job and level from progression config
	config, err := s.progressionSvc.GetJobUnlockConfig(ctx, featureKey)
	if err != nil {
		// Log error or assume locked if not found
		return false, fmt.Errorf("failed to fetch job unlock config for feature %s: %w", featureKey, err)
	}

	if config == nil {
		return false, nil // No config means not unlocked by job
	}

	// Now check the user's level for that specific job
	level, err := s.GetJobLevel(ctx, userID, config.JobKey)
	if err != nil {
		return false, fmt.Errorf("failed to get job level: %w", err)
	}

	return level >= config.RequiredLevel, nil
}
