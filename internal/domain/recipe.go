package domain

import "time"

// RecipeCost represents a single material requirement for a recipe
type RecipeCost struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}

// Recipe represents a crafting recipe for upgrading items
type Recipe struct {
	ID           int           `json:"recipe_id"`
	TargetItemID int           `json:"target_item_id"`
	BaseCost     []RecipeCost  `json:"base_cost"`
	CreatedAt    time.Time     `json:"created_at,omitempty"`
}

// RecipeUnlock tracks which recipes a user has unlocked
type RecipeUnlock struct {
	UserID     string    `json:"user_id"`
	RecipeID   int       `json:"recipe_id"`
	UnlockedAt time.Time `json:"unlocked_at,omitempty"`
}
