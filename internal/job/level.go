package job

import (
	"math"
)

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
