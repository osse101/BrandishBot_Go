package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExpeditionState represents the state of an expedition
type ExpeditionState string

const (
	ExpeditionStateCreated    ExpeditionState = "Created"
	ExpeditionStateRecruiting ExpeditionState = "Recruiting"
	ExpeditionStateInProgress ExpeditionState = "InProgress"
	ExpeditionStateCompleted  ExpeditionState = "Completed"
)

// ExpeditionMetadata stores expedition configuration and results
type ExpeditionMetadata struct {
	Difficulty      string            `json:"difficulty,omitempty"`
	MinParticipants int               `json:"min_participants,omitempty"`
	MaxParticipants int               `json:"max_participants,omitempty"`
	LootTableKey    string            `json:"loot_table_key,omitempty"`
	SuccessRate     float64           `json:"success_rate,omitempty"`
	Modifiers       map[string]float64 `json:"modifiers,omitempty"`
}

// ExpeditionRewards represents rewards for a participant
type ExpeditionRewards struct {
	Items []Item `json:"items"`
	XP    int    `json:"xp,omitempty"`
	Money int    `json:"money,omitempty"`
}

// Expedition represents a group expedition/adventure
type Expedition struct {
	ID                 uuid.UUID           `json:"id"`
	InitiatorID        uuid.UUID           `json:"initiator_id"`
	ExpeditionType     string              `json:"expedition_type"`
	State              ExpeditionState     `json:"state"`
	CreatedAt          time.Time           `json:"created_at"`
	JoinDeadline       time.Time           `json:"join_deadline"`
	CompletionDeadline time.Time           `json:"completion_deadline"`
	CompletedAt        *time.Time          `json:"completed_at,omitempty"`
	Metadata           *ExpeditionMetadata `json:"metadata,omitempty"`
}

// ExpeditionParticipant represents a user participating in an expedition
type ExpeditionParticipant struct {
	ExpeditionID uuid.UUID          `json:"expedition_id"`
	UserID       uuid.UUID          `json:"user_id"`
	JoinedAt     time.Time          `json:"joined_at"`
	Rewards      *ExpeditionRewards `json:"rewards,omitempty"`
}

// ExpeditionDetails includes expedition and all participants
type ExpeditionDetails struct {
	Expedition   Expedition              `json:"expedition"`
	Participants []ExpeditionParticipant `json:"participants"`
}

// MarshalExpeditionMetadata converts ExpeditionMetadata to JSONB
func MarshalExpeditionMetadata(metadata ExpeditionMetadata) ([]byte, error) {
	return json.Marshal(metadata)
}

// UnmarshalExpeditionMetadata converts JSONB to ExpeditionMetadata
func UnmarshalExpeditionMetadata(data []byte) (*ExpeditionMetadata, error) {
	var metadata ExpeditionMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

// MarshalExpeditionRewards converts ExpeditionRewards to JSONB
func MarshalExpeditionRewards(rewards ExpeditionRewards) ([]byte, error) {
	return json.Marshal(rewards)
}

// UnmarshalExpeditionRewards converts JSONB to ExpeditionRewards
func UnmarshalExpeditionRewards(data []byte) (*ExpeditionRewards, error) {
	var rewards ExpeditionRewards
	if err := json.Unmarshal(data, &rewards); err != nil {
		return nil, err
	}
	return &rewards, nil
}
