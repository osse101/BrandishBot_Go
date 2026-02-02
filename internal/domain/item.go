package domain

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

// ShineLevel represents the visual rarity and quality of an item
type ShineLevel string

const (
	ShineCommon    ShineLevel = "COMMON"
	ShineUncommon  ShineLevel = "UNCOMMON"
	ShineRare      ShineLevel = "RARE"
	ShineEpic      ShineLevel = "EPIC"
	ShineLegendary ShineLevel = "LEGENDARY"
	ShinePoor      ShineLevel = "POOR"
	ShineJunk      ShineLevel = "JUNK"
	ShineCursed    ShineLevel = "CURSED"
)

// Shine multipliers (Boosts item value and Gamble Score)
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
