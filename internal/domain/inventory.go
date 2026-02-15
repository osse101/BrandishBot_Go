package domain

// InventorySlot represents a single item slot in the user's inventory
type InventorySlot struct {
	ItemID       int          `json:"item_id"`
	Quantity     int          `json:"quantity"`
	QualityLevel QualityLevel `json:"quality,omitempty"` // COMMON/UNCOMMON/RARE/EPIC/LEGENDARY
}

// Inventory represents the structure stored in the JSONB column
type Inventory struct {
	Slots      []InventorySlot `json:"slots"`
	LastUpdate int64           `json:"last_update,omitempty"`
}
