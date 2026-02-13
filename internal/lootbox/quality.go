package lootbox

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// qualityThreshold defines a mapping between a roll threshold and a quality level.
type qualityThreshold struct {
	threshold float64
	quality   domain.QualityLevel
}

// qualityThresholds defines the ordered list of thresholds for determining item quality.
// The order is critical: checks are performed from rarest (lowest roll) to most common.
var qualityThresholds = []qualityThreshold{
	{QualityLegendaryThreshold, domain.QualityLegendary},
	{QualityEpicThreshold, domain.QualityEpic},
	{QualityRareThreshold, domain.QualityRare},
	{QualityUncommonThreshold, domain.QualityUncommon},
	{QualityCommonThreshold, domain.QualityCommon},
	{QualityPoorThreshold, domain.QualityPoor},
	{QualityJunkThreshold, domain.QualityJunk},
}

// calculateQuality determines the visual rarity "quality" and value multiplier of a drop based on a roll.
// The boxQuality level shifts the constraints: a more rare box makes it easier to get rare item quality levels.
func (s *service) calculateQuality(roll float64, boxQuality domain.QualityLevel, canUpgrade bool) (domain.QualityLevel, float64) {
	dist := s.getQualityDistance(boxQuality)
	bonus := 0.03 * float64(dist)

	// Default to Cursed if no threshold is met (roll > QualityJunkThreshold + bonus)
	quality := domain.QualityCursed

	for _, qt := range qualityThresholds {
		if roll <= qt.threshold+bonus {
			quality = qt.quality
			break
		}
	}

	// Critical Quality Upgrade: 1% chance to upgrade the quality level (locked by progression)
	if canUpgrade && s.rnd() < CriticalQualityUpgradeChance {
		quality = s.getNextQualityLevel(quality)
	}

	return quality, utils.GetQualityMultiplier(quality)
}

func (s *service) getNextQualityLevel(q domain.QualityLevel) domain.QualityLevel {
	switch q {
	case domain.QualityCursed:
		return domain.QualityJunk
	case domain.QualityJunk:
		return domain.QualityPoor
	case domain.QualityPoor:
		return domain.QualityCommon
	case domain.QualityCommon:
		return domain.QualityUncommon
	case domain.QualityUncommon:
		return domain.QualityRare
	case domain.QualityRare:
		return domain.QualityEpic
	case domain.QualityEpic:
		return domain.QualityLegendary
	default:
		return q
	}
}

func (s *service) getQualityDistance(quality domain.QualityLevel) int {
	switch quality {
	case domain.QualityLegendary:
		return 4
	case domain.QualityEpic:
		return 3
	case domain.QualityRare:
		return 2
	case domain.QualityUncommon:
		return 1
	case domain.QualityCommon:
		return 0
	case domain.QualityPoor:
		return -1
	case domain.QualityJunk:
		return -2
	case domain.QualityCursed:
		return -3
	default:
		return 0
	}
}
