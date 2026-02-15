package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)


// ExpeditionMetadata stores expedition configuration and results
type ExpeditionMetadata struct {
	Difficulty      string             `json:"difficulty,omitempty"`
	MinParticipants int                `json:"min_participants,omitempty"`
	MaxParticipants int                `json:"max_participants,omitempty"`
	LootTableKey    string             `json:"loot_table_key,omitempty"`
	SuccessRate     float64            `json:"success_rate,omitempty"`
	Modifiers       map[string]float64 `json:"modifiers,omitempty"`
}

// ExpeditionRewards represents rewards for a participant
type ExpeditionRewards struct {
	Items []string `json:"items"`
	XP    int      `json:"xp,omitempty"`
	Money int      `json:"money,omitempty"`
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
	Username     string             `json:"username"`
	JoinedAt     time.Time          `json:"joined_at"`
	IsLeader     bool               `json:"is_leader"`
	JobLevels    map[string]int     `json:"job_levels,omitempty"`
	FinalMoney   int                `json:"final_money,omitempty"`
	FinalXP      int                `json:"final_xp,omitempty"`
	FinalItems   []string           `json:"final_items,omitempty"`
	Rewards      *ExpeditionRewards `json:"rewards,omitempty"`
}

// ExpeditionDetails includes expedition and all participants
type ExpeditionDetails struct {
	Expedition   Expedition              `json:"expedition"`
	Participants []ExpeditionParticipant `json:"participants"`
}

// PartyMemberState tracks runtime state during an expedition turn loop
type PartyMemberState struct {
	UserID      uuid.UUID
	Username    string
	JobLevels   map[string]int // All job levels: {"blacksmith": 5, "explorer": 12, ...}
	IsConscious bool
	IsDebuffed  bool
	TempSkills  []ExpeditionSkill
	PrizeMoney  int
	PrizeItems  []string
}

// ExpeditionTurn records a single turn in the expedition
type ExpeditionTurn struct {
	TurnNumber    int             `json:"turn_number"`
	EncounterType EncounterType   `json:"encounter_type"`
	Outcome       OutcomeType     `json:"outcome"`
	SkillChecked  ExpeditionSkill `json:"skill_checked"`
	SkillPassed   bool            `json:"skill_passed"`
	PrimaryMember string          `json:"primary_member"`
	Narrative     string          `json:"narrative"`
	Fatigue       int             `json:"fatigue"`
	PurseAfter    int             `json:"purse_after"`
}

// ExpeditionResult captures the final outcome of an expedition
type ExpeditionResult struct {
	TotalTurns    int                 `json:"total_turns"`
	Won           bool                `json:"won"`
	AllKnockedOut bool                `json:"all_knocked_out"`
	FinalFatigue  int                 `json:"final_fatigue"`
	PartyRewards  []PartyMemberReward `json:"party_rewards"`
	Journal       []ExpeditionTurn    `json:"journal"`
}

// PartyMemberReward captures rewards for a single party member
type PartyMemberReward struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Money    int       `json:"money"`
	Items    []string  `json:"items"`
	XP       int       `json:"xp"`
	IsLeader bool      `json:"is_leader"`
}

// ExpeditionJournalEntry is the DB-persisted form of a journal entry
type ExpeditionJournalEntry struct {
	ID            int       `json:"id"`
	ExpeditionID  uuid.UUID `json:"expedition_id"`
	TurnNumber    int       `json:"turn_number"`
	EncounterType string    `json:"encounter_type"`
	Outcome       string    `json:"outcome"`
	SkillChecked  string    `json:"skill_checked"`
	SkillPassed   bool      `json:"skill_passed"`
	PrimaryMember string    `json:"primary_member"`
	Narrative     string    `json:"narrative"`
	Fatigue       int       `json:"fatigue"`
	Purse         int       `json:"purse"`
	CreatedAt     time.Time `json:"created_at"`
}

// ExpeditionStatus represents the current state of the expedition system
type ExpeditionStatus struct {
	HasActive       bool               `json:"has_active"`
	ActiveDetails   *ExpeditionDetails `json:"active_details,omitempty"`
	CooldownExpires *time.Time         `json:"cooldown_expires,omitempty"`
	OnCooldown      bool               `json:"on_cooldown"`
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
