package domain

// PredictionWinner represents the user who won the prediction
type PredictionWinner struct {
	Username   string `json:"username"`
	PlatformID string `json:"platform_id"`
	PointsWon  int    `json:"points_won"`
}

// PredictionParticipant represents a user who participated in the prediction
type PredictionParticipant struct {
	Username    string `json:"username"`
	PlatformID  string `json:"platform_id"`
	PointsSpent int    `json:"points_spent"`
}

// PredictionOutcomeRequest represents the request to process a prediction outcome
type PredictionOutcomeRequest struct {
	Platform         string                  `json:"platform" validate:"required,oneof=twitch youtube"`
	Winner           PredictionWinner        `json:"winner" validate:"required"`
	TotalPointsSpent int                     `json:"total_points_spent" validate:"required,min=0"`
	Participants     []PredictionParticipant `json:"participants" validate:"required,min=1,dive"`
}

// PredictionResult represents the result of processing a prediction outcome
type PredictionResult struct {
	TotalPoints           int    `json:"total_points"`
	ContributionAwarded   int    `json:"contribution_awarded"`
	ParticipantsProcessed int    `json:"participants_processed"`
	WinnerXPAwarded       int    `json:"winner_xp_awarded"`
	Message               string `json:"message"`
}
