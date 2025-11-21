package domain

// Item represents an item in the system
type Item struct {
	ID          int    `json:"item_id"`
	Name        string `json:"item_name"`
	Description string `json:"item_description"`
	BaseValue   int    `json:"base_value"`
}
