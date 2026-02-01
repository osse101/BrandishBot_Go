package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CompostDeposit represents a composting deposit
type CompostDeposit struct {
	ID          uuid.UUID        `json:"id"`
	UserID      uuid.UUID        `json:"user_id"`
	ItemKey     string           `json:"item_key"`
	Quantity    int              `json:"quantity"`
	DepositedAt time.Time        `json:"deposited_at"`
	ReadyAt     time.Time        `json:"ready_at"`
	HarvestedAt *time.Time       `json:"harvested_at,omitempty"`
	GemsAwarded *int             `json:"gems_awarded,omitempty"`
	Metadata    *CompostMetadata `json:"metadata,omitempty"`
}

// CompostMetadata stores additional deposit information
type CompostMetadata struct {
	ItemRarity     string `json:"item_rarity,omitempty"`
	ConversionRate int    `json:"conversion_rate,omitempty"` // Gems per item
	BonusApplied   bool   `json:"bonus_applied,omitempty"`
}

// CompostStatus represents a user's compost status
type CompostStatus struct {
	ActiveDeposits   []CompostDeposit `json:"active_deposits"`
	ReadyCount       int              `json:"ready_count"`
	TotalGemsPending int              `json:"total_gems_pending"`
}

// MarshalCompostMetadata converts CompostMetadata to JSONB
func MarshalCompostMetadata(metadata CompostMetadata) ([]byte, error) {
	return json.Marshal(metadata)
}

// UnmarshalCompostMetadata converts JSONB to CompostMetadata
func UnmarshalCompostMetadata(data []byte) (*CompostMetadata, error) {
	var metadata CompostMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}
