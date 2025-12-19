package domain

// Item represents an item in the system with three-layer naming:
// - InternalName: stable code identifier (e.g., "weapon_blaster")
// - PublicName: user-facing command name (e.g., "missile")
// - DefaultDisplay: fallback display name (e.g., "Ray Gun")
type Item struct {
	ID             int    `json:"item_id"`
	InternalName   string `json:"internal_name"`
	PublicName     string `json:"public_name"`
	DefaultDisplay string `json:"default_display"`
	Description    string `json:"description"`
	BaseValue      int    `json:"base_value"`
	Handler        string `json:"handler"`
}
