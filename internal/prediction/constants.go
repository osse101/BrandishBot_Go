package prediction

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

const (
	// XP rewards
	WinnerXP      = 100 // XP awarded to the prediction winner
	ParticipantXP = 10  // XP awarded to each participant

	// Item rewards
	GrenadeItemName = domain.ItemGrenade // Item awarded to winner if unlocked
	GrenadeQuantity = 1                  // Number of grenades awarded to winner

	// Logarithmic conversion formula parameters
	// Formula: BaseContribution + (log10(points/PointsScaleDivisor) / LogDivisor) * ScaleMultiplier + BonusContribution
	// Goal: 10,000 points = 1 contribution, 1,000,000 points = 50 contribution
	PointsScaleDivisor = 10000.0 // Scale points starting at 10k
	LogDivisor         = 2.0     // divisor such that log10(1,000,000/10,000) = 2.0
	ScaleMultiplier    = 49.0    // Multiplier to span 1 to 50
	BaseContribution   = 1.0     // Base contribution value at 10k points
	BonusContribution  = 0.0     // No extra bonus needed for this scale

	// Job and stat identifiers
	GamblerJobKey                    = "gambler"
	PredictionStatType               = "prediction_participation"
	TotalPointsMetricType            = "prediction_total_points"
	PredictionContributionMetricType = "prediction_contribution"
)
