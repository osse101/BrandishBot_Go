package domain

import (
	"time"

	"github.com/google/uuid"
)

// GambleState represents the current state of a gamble
type GambleState string

const (
	GambleStateCreated   GambleState = "Created"
	GambleStateJoining   GambleState = "Joining"
	GambleStateOpening   GambleState = "Opening"
	GambleStateCompleted GambleState = "Completed"
	GambleStateRefunded  GambleState = "Refunded"
)

// Event types
const (
	EventGambleStarted = "GambleStarted"
)

// Gamble represents a multiplayer lootbox gamble session
type Gamble struct {
	ID           uuid.UUID     `json:"id"`
	InitiatorID  string        `json:"initiator_id"`
	State        GambleState   `json:"state"`
	CreatedAt    time.Time     `json:"created_at"`
	JoinDeadline time.Time     `json:"join_deadline"`
	Participants []Participant `json:"participants,omitempty"`
	WinnerID     *string       `json:"winner_id,omitempty"`
	TotalValue   int64         `json:"total_value,omitempty"`
}

// LootboxBet represents a wager of a specific lootbox item
type LootboxBet struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}

// Participant represents a user who has joined the gamble
type Participant struct {
	GambleID    uuid.UUID    `json:"gamble_id"`
	UserID      string       `json:"user_id"`
	LootboxBets []LootboxBet `json:"lootbox_bets"`
	Username    string       `json:"username,omitempty"` // Populated for display
}

// GambleOpenedItem represents an item opened during the gamble
type GambleOpenedItem struct {
	GambleID uuid.UUID `json:"gamble_id"`
	UserID   string    `json:"user_id"`
	ItemID   int       `json:"item_id"`
	Value    int64     `json:"value"`
}

// GambleResult contains the outcome of a completed gamble
type GambleResult struct {
	GambleID   uuid.UUID          `json:"gamble_id"`
	WinnerID   string             `json:"winner_id"`
	TotalValue int64              `json:"total_value"`
	Items      []GambleOpenedItem `json:"items"`
}
