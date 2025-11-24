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
	TargetItemID int           `json:"item_id"` // Changed from TargetItemID to item_id for consistency
	BaseCost     []RecipeCost  `json:"base_cost"`
	CreatedAt    time.Time     `json:"created_at,omitempty"`
}

// RecipeUnlock tracks which recipes a user has unlocked
type RecipeUnlock struct {
	UserID     string    `json:"user_id"`
	RecipeID   int       `json:"recipe_id"`
	UnlockedAt time.Time `json:"unlocked_at,omitempty"`
}

// DisassembleRecipe represents a recipe for disassembling items
type DisassembleRecipe struct {
	ID               int            `json:"recipe_id"`
	SourceItemID     int            `json:"source_item_id"`
	QuantityConsumed int            `json:"quantity_consumed"`
	Outputs          []RecipeOutput `json:"outputs"`
	CreatedAt        time.Time      `json:"created_at,omitempty"`
}

// RecipeOutput represents the materials produced from disassembling
type RecipeOutput struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}

// RecipeAssociation links upgrade and disassemble recipes
type RecipeAssociation struct {
	UpgradeRecipeID     int `json:"upgrade_recipe_id"`
	DisassembleRecipeID int `json:"disassemble_recipe_id"`
}
