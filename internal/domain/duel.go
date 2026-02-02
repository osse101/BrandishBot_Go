package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DuelState represents the state of a duel
type DuelState string

const (
	DuelStatePending    DuelState = "pending"
	DuelStateAccepted   DuelState = "accepted"
	DuelStateInProgress DuelState = "in_progress"
	DuelStateCompleted  DuelState = "completed"
	DuelStateDeclined   DuelState = "declined"
	DuelStateExpired    DuelState = "expired"
)

// DuelStakes represents the stakes/bet terms of a duel
type DuelStakes struct {
	TimeoutDuration int    `json:"timeout_duration"` // Seconds
	WagerItemKey    string `json:"wager_item_key,omitempty"`
	WagerAmount     int    `json:"wager_amount,omitempty"`
}

// DuelResult represents the outcome of a duel
type DuelResult struct {
	WinnerID uuid.UUID `json:"winner_id"`
	LoserID  uuid.UUID `json:"loser_id"`
	Method   string    `json:"method"` // "coin_flip", "dice_roll", etc.
	Details  string    `json:"details,omitempty"`
}

// Duel represents a duel challenge between two users
type Duel struct {
	ID           uuid.UUID   `json:"id"`
	ChallengerID uuid.UUID   `json:"challenger_id"`
	OpponentID   *uuid.UUID  `json:"opponent_id,omitempty"`
	State        DuelState   `json:"state"`
	Stakes       DuelStakes  `json:"stakes"`
	CreatedAt    time.Time   `json:"created_at"`
	ExpiresAt    time.Time   `json:"expires_at"`
	StartedAt    *time.Time  `json:"started_at,omitempty"`
	CompletedAt  *time.Time  `json:"completed_at,omitempty"`
	WinnerID     *uuid.UUID  `json:"winner_id,omitempty"`
	ResultData   *DuelResult `json:"result_data,omitempty"`
}

// MarshalStakes converts DuelStakes to JSONB
func MarshalStakes(stakes DuelStakes) ([]byte, error) {
	return json.Marshal(stakes)
}

// UnmarshalStakes converts JSONB to DuelStakes
func UnmarshalStakes(data []byte) (*DuelStakes, error) {
	var stakes DuelStakes
	if err := json.Unmarshal(data, &stakes); err != nil {
		return nil, err
	}
	return &stakes, nil
}

// MarshalDuelResult converts DuelResult to JSONB
func MarshalDuelResult(result DuelResult) ([]byte, error) {
	return json.Marshal(result)
}

// UnmarshalDuelResult converts JSONB to DuelResult
func UnmarshalDuelResult(data []byte) (*DuelResult, error) {
	var result DuelResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
