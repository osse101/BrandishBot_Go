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
	ContentType    []string `json:"content_type" db:"content_type"` // Content type categorization (weapon, material, etc.)
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

// Quality multipliers (Boosts item value and Gamble Score)
const (
	MultCommon    = 1.0
	MultUncommon  = 1.1
	MultRare      = 1.25
	MultEpic      = 1.5
	MultLegendary = 2.0
	MultPoor      = 0.8
	MultJunk      = 0.6
	MultCursed    = 0.4
)

// IsCurrency returns true if this item is a currency (should not have quality variations)
func (i *Item) IsCurrency() bool {
	for _, t := range i.Types {
		if t == "currency" {
			return true
		}
	}
	return false
}

// Item tag constants (from item_types / tags in items.json)
const (
	CompostableTag = "compostable"
	NoUseTag       = "no-use"
)

// Content type constants (from "type" field in items.json)
const (
	ContentTypeWeapon    = "weapon"
	ContentTypeExplosive = "explosive"
	ContentTypeDefense   = "defense"
	ContentTypeHealing   = "healing"
	ContentTypeMaterial  = "material"
	ContentTypeContainer = "container"
	ContentTypeUtility   = "utility"
	ContentTypeMagical   = "magical"
)

// HasTag checks if a tags slice contains the specified tag.
func HasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// HasType checks if a content types slice contains the specified type.
func HasType(contentTypes []string, t string) bool {
	for _, ct := range contentTypes {
		if ct == t {
			return true
		}
	}
	return false
}
