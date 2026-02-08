package domain

import "time"

// Quest represents a weekly quest definition
type Quest struct {
	QuestID         int       `json:"quest_id"`
	QuestKey        string    `json:"quest_key"`
	QuestType       string    `json:"quest_type"` // 'buy_items', 'sell_items', 'earn_money', 'craft_recipe', 'perform_searches'
	Description     string    `json:"description"`
	TargetCategory  *string   `json:"target_category,omitempty"`   // For: buy_items, sell_items
	TargetRecipeKey *string   `json:"target_recipe_key,omitempty"` // For: craft_recipe
	BaseRequirement int       `json:"base_requirement"`
	BaseRewardMoney int       `json:"base_reward_money"`
	BaseRewardXp    int       `json:"base_reward_xp"`
	Active          bool      `json:"active"`
	WeekNumber      int       `json:"week_number"`
	Year            int       `json:"year"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// QuestProgress represents user progress on a quest
type QuestProgress struct {
	UserID           string     `json:"user_id"`
	QuestID          int        `json:"quest_id"`
	ProgressCurrent  int        `json:"progress_current"`
	ProgressRequired int        `json:"progress_required"`
	RewardMoney      int        `json:"reward_money"`
	RewardXp         int        `json:"reward_xp"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	ClaimedAt        *time.Time `json:"claimed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Joined fields
	QuestKey        string  `json:"quest_key,omitempty"`
	QuestType       string  `json:"quest_type,omitempty"`
	Description     string  `json:"description,omitempty"`
	TargetCategory  *string `json:"target_category,omitempty"`   // For: buy_items, sell_items
	TargetRecipeKey *string `json:"target_recipe_key,omitempty"` // For: craft_recipe
}

// WeeklySale represents a weekly item category sale (7-day rotation)
type WeeklySale struct {
	WeekOffset      int     `json:"week_offset"`               // Which week in the rotation (0-indexed)
	TargetCategory  *string `json:"target_category,omitempty"` // Item category on sale (nil = all items)
	DiscountPercent float64 `json:"discount_percent"`
	Description     string  `json:"description"`
}

// QuestTemplate represents a quest template from config
type QuestTemplate struct {
	QuestKey        string  `json:"quest_key"`
	QuestType       string  `json:"quest_type"`
	Description     string  `json:"description"`
	TargetCategory  *string `json:"target_category,omitempty"`   // For: buy_items, sell_items
	TargetRecipeKey *string `json:"target_recipe_key,omitempty"` // For: craft_recipe
	BaseRequirement int     `json:"base_requirement"`
	BaseRewardMoney int     `json:"base_reward_money"`
	BaseRewardXp    int     `json:"base_reward_xp"`
}

// WeeklySaleConfig represents the weekly sales configuration
type WeeklySaleConfig struct {
	Version       string       `json:"version"`
	SalesSchedule []WeeklySale `json:"sales_schedule"`
}

// QuestPoolConfig represents the quest pool configuration
type QuestPoolConfig struct {
	Version   string          `json:"version"`
	QuestPool []QuestTemplate `json:"quest_pool"`
}

// Quest type constants
const (
	QuestTypeBuyItems        = "buy_items"        // Buy X items of target category
	QuestTypeSellItems       = "sell_items"       // Sell X items
	QuestTypeEarnMoney       = "earn_money"       // Earn X money from sales
	QuestTypeCraftRecipe     = "craft_recipe"     // Perform recipe (upgrade/disassemble) X times
	QuestTypePerformSearches = "perform_searches" // Perform X searches
	// Extensible: add new quest types as needed
)
