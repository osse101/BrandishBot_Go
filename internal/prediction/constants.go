package prediction

const (
	// XP rewards
	WinnerXP      = 100 // XP awarded to the prediction winner
	ParticipantXP = 10  // XP awarded to each participant

	// Logarithmic conversion formula parameters
	// Formula: 1 + (log10(points/1000) / 3) * 99 + 10
	PointsScaleDivisor = 1000.0 // Scale points to thousands
	LogDivisor         = 3.0    // Divisor for log component
	ScaleMultiplier    = 99.0   // Multiplier for scaling range
	BaseContribution   = 1.0    // Base contribution value
	BonusContribution  = 10.0   // Bonus contribution added to formula

	// Job and stat identifiers
	GamblerJobKey         = "gambler"
	PredictionStatType    = "prediction_participation"
	TotalPointsMetricType = "prediction_total_points"
)
