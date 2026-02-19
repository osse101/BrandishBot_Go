package lootbox

import "github.com/osse101/BrandishBot_Go/internal/domain"

// ============================================================================
// Public domain types
// ============================================================================

// DroppedItem represents an item generated from opening a lootbox.
type DroppedItem struct {
	ItemID       int
	ItemName     string
	Quantity     int
	Value        int
	QualityLevel domain.QualityLevel
}
