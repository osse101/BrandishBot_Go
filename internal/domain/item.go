package domain

import "time"

// Item represents an item in the system with three-layer naming:
// - InternalName: stable code identifier (e.g., "weapon_blaster")
// - PublicName: user-facing command name (e.g., "missile")
// - DefaultDisplay: fallback display name (e.g., "Ray Gun")
type Item struct {
	ID             int      `json:"item_id" db:"item_id"`
	InternalName   string   `json:"internal_name" db:"internal_name"`
	PublicName     string   `json:"public_name" db:"public_name"`
	DefaultDisplay string   `json:"default_display" db:"default_display"`
	Description    string   `json:"description" db:"item_description"`
	BaseValue      int      `json:"base_value" db:"base_value"`     // Buy price
	SellPrice      *int     `json:"sell_price,omitempty"`           // Calculated sell price (only set for sellable items)
	Types          []string `json:"types" db:"types"`               // Populated from join/separate query
	Handler        *string  `json:"handler,omitempty" db:"handler"` // Nullable: some items have no handler
}

// QualityLevel represents the visual rarity and quality of an item
type QualityLevel string

const (
	QualityCommon    QualityLevel = "COMMON"
	QualityUncommon  QualityLevel = "UNCOMMON"
	QualityRare      QualityLevel = "RARE"
	QualityEpic      QualityLevel = "EPIC"
	QualityLegendary QualityLevel = "LEGENDARY"
	QualityPoor      QualityLevel = "POOR"
	QualityJunk      QualityLevel = "JUNK"
	QualityCursed    QualityLevel = "CURSED"
)

// GetTimeoutAdjustment returns the timeout adjustment in seconds based on quality level
// Distance from common * 10s
func (s QualityLevel) GetTimeoutAdjustment() time.Duration {
	qualityModifier := map[QualityLevel]time.Duration{
		QualityCursed:    -30 * time.Second,
		QualityJunk:      -20 * time.Second,
		QualityPoor:      -10 * time.Second,
		QualityCommon:    0 * time.Second,
		QualityUncommon:  10 * time.Second,
		QualityRare:      20 * time.Second,
		QualityEpic:      30 * time.Second,
		QualityLegendary: 40 * time.Second,
	}

	if modifier, ok := qualityModifier[s]; ok {
		return modifier
	}
	return 0
}
