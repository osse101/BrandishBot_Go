package crafting

import (
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// ItemUpgradedPayload represents the data for an item upgraded event
type ItemUpgradedPayload struct {
	UserID        string `json:"user_id"`
	ItemName      string `json:"item_name"`
	Quantity      int    `json:"quantity"`
	RecipeKey     string `json:"recipe_key,omitempty"`
	IsMasterwork  bool   `json:"is_masterwork"`
	BonusQuantity int    `json:"bonus_quantity"`
	Timestamp     int64  `json:"timestamp"`
}

// ItemDisassembledPayload represents the data for an item disassembled event
type ItemDisassembledPayload struct {
	UserID              string         `json:"user_id"`
	ItemName            string         `json:"item_name"`
	Quantity            int            `json:"quantity"`
	RecipeKey           string         `json:"recipe_key,omitempty"`
	IsPerfectSalvage    bool           `json:"is_perfect_salvage"`
	PerfectSalvageCount int            `json:"perfect_salvage_count"`
	Multiplier          float64        `json:"multiplier"`
	Outputs             map[string]int `json:"outputs"`
	Timestamp           int64          `json:"timestamp"`
}

// NewItemUpgradedEvent creates a new event for an item upgrade
func NewItemUpgradedEvent(userID, itemName string, quantity int, recipeKey string, isMasterwork bool, bonusQuantity int) event.Event {
	return event.Event{
		Version: event.EventSchemaVersion,
		Type:    domain.EventTypeItemUpgraded,
		Payload: ItemUpgradedPayload{
			UserID:        userID,
			ItemName:      itemName,
			Quantity:      quantity,
			RecipeKey:     recipeKey,
			IsMasterwork:  isMasterwork,
			BonusQuantity: bonusQuantity,
			Timestamp:     time.Now().Unix(),
		},
		Metadata: map[string]interface{}{
			domain.MetadataKeyItemName: itemName,
			domain.MetadataKeyQuantity: quantity,
			domain.MetadataKeySource:   "crafting",
		},
	}
}

// NewItemDisassembledEvent creates a new event for an item disassemble
func NewItemDisassembledEvent(userID, itemName string, quantity int, recipeKey string, isPerfectSalvage bool, perfectSalvageCount int, multiplier float64, outputs map[string]int) event.Event {
	return event.Event{
		Version: event.EventSchemaVersion,
		Type:    domain.EventTypeItemDisassembled,
		Payload: ItemDisassembledPayload{
			UserID:              userID,
			ItemName:            itemName,
			Quantity:            quantity,
			RecipeKey:           recipeKey,
			IsPerfectSalvage:    isPerfectSalvage,
			PerfectSalvageCount: perfectSalvageCount,
			Multiplier:          multiplier,
			Outputs:             outputs,
			Timestamp:           time.Now().Unix(),
		},
		Metadata: map[string]interface{}{
			domain.MetadataKeyItemName: itemName,
			domain.MetadataKeyQuantity: quantity,
			domain.MetadataKeySource:   "crafting",
		},
	}
}
