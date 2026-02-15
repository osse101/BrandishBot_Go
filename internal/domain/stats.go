package domain

import "time"

// EventType represents the type of event being tracked
type EventType string

const (
	EventUserRegistered  EventType = "user_registered"
	EventItemAdded       EventType = "item_added"
	EventItemRemoved     EventType = "item_removed"
	EventItemUsed        EventType = "item_used"
	EventItemSold        EventType = "item_sold"
	EventItemBought      EventType = "item_bought"
	EventItemTransferred EventType = "item_transferred"
	EventMessageReceived EventType = "message_received"
	// Gamble events
	EventGambleNearMiss     EventType = "gamble_near_miss"
	EventGambleTieBreakLost EventType = "gamble_tie_break_lost"
	EventGambleCriticalFail EventType = "gamble_critical_fail"
	EventDailyStreak        EventType = "daily_streak"
	// Search events
	EventSearch                  EventType = "search"
	EventSearchNearMiss          EventType = "search_near_miss"
	EventSearchCriticalFail      EventType = "search_critical_fail"
	EventSearchCriticalSuccess   EventType = "search_critical_success"
	EventCraftingCriticalSuccess EventType = "crafting_critical_success"
	EventCraftingPerfectSalvage  EventType = "crafting_perfect_salvage"
	EventJobLevelUp              EventType = "job_level_up"
	EventJobXPCritical           EventType = "job_xp_critical"
	// Lootbox events
	EventLootboxJackpot EventType = "lootbox_jackpot"
	EventLootboxBigWin  EventType = "lootbox_big_win"
	// Slots events
	EventSlotsSpin        EventType = "slots_spin"
	EventSlotsWin         EventType = "slots_win"
	EventSlotsMegaJackpot EventType = "slots_mega_jackpot"
)

// StatsEvent represents a single tracked event
type StatsEvent struct {
	EventID   int64       `json:"event_id"`
	UserID    string      `json:"user_id"`
	EventType EventType   `json:"event_type"`
	EventData interface{} `json:"event_data,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

// CraftingMetadata represents metadata for crafting events
type CraftingMetadata struct {
	ItemName         string  `json:"item_name"`
	OriginalQuantity int     `json:"original_quantity,omitempty"`
	Quantity         int     `json:"quantity,omitempty"`
	MasterworkCount  int     `json:"masterwork_count,omitempty"`
	BonusQuantity    int     `json:"bonus_quantity,omitempty"`
	PerfectCount     int     `json:"perfect_count,omitempty"`
	Multiplier       float64 `json:"multiplier,omitempty"`
}

// SlotsMetadata represents metadata for slots events
type SlotsMetadata struct {
	BetAmount        int     `json:"bet_amount"`
	PayoutAmount     int     `json:"payout_amount"`
	PayoutMultiplier float64 `json:"payout_multiplier"`
	NetProfit        int     `json:"net_profit"`
	IsWin            bool    `json:"is_win"`
	IsNearMiss       bool    `json:"is_near_miss"`
	TriggerType      string  `json:"trigger_type"`
	Reel1            string  `json:"reel1"`
	Reel2            string  `json:"reel2"`
	Reel3            string  `json:"reel3"`
}

// GambleMetadata represents metadata for gamble events
type GambleMetadata struct {
	GambleID    string `json:"gamble_id"`
	Score       int64  `json:"score"`
	WinnerScore int64  `json:"winner_score,omitempty"`
}

// SearchMetadata represents metadata for search events
type SearchMetadata struct {
	IsCritical   bool `json:"is_critical"`
	IsNearMiss   bool `json:"is_near_miss"`
	IsCritFail   bool `json:"is_crit_fail"`
	IsFirstDaily bool `json:"is_first_daily"`
	XPAmount     int  `json:"xp_amount"`
}

// PredictionMetadata represents metadata for prediction events
type PredictionMetadata struct {
	Username string `json:"username"`
	IsWinner bool   `json:"is_winner"`
	Platform string `json:"platform"`
	XP       int    `json:"xp"`
}

// JobMetadata represents metadata for job events
type JobMetadata struct {
	Source string `json:"source"`
}

// StreakMetadata represents metadata for streak events
type StreakMetadata struct {
	Streak int `json:"streak"`
}

// LootboxEventData represents data for lootbox jackpot/big-win events
type LootboxEventData struct {
	Item   string      `json:"item"`
	Drops  interface{} `json:"drops"` // []lootbox.DroppedItem, but using interface{} to avoid circular deps
	Value  int         `json:"value"`
	Source string      `json:"source"`
}

// ToMap converts LootboxEventData to map for compatibility with existing event recording
// ToMap converts LootboxEventData to map - REMOVED

// StatsAggregate represents pre-calculated statistics for a time period
type StatsAggregate struct {
	AggregateID int                    `json:"aggregate_id"`
	Period      string                 `json:"period"` // daily, weekly, monthly
	PeriodStart time.Time              `json:"period_start"`
	PeriodEnd   time.Time              `json:"period_end"`
	Metrics     map[string]interface{} `json:"metrics"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// StatsSummary represents a summary of statistics for API responses
type StatsSummary struct {
	Period      string                 `json:"period"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	TotalEvents int                    `json:"total_events"`
	EventCounts map[EventType]int      `json:"event_counts"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
}

// LeaderboardEntry represents a user's position in a leaderboard
type LeaderboardEntry struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username,omitempty"`
	Count     int    `json:"count"`
	EventType string `json:"event_type"`
}

// SlotsStats represents aggregated slots statistics for a user
type SlotsStats struct {
	UserID          string  `json:"user_id"`
	Username        string  `json:"username,omitempty"`
	TotalSpins      int     `json:"total_spins"`
	TotalWins       int     `json:"total_wins"`
	TotalBet        int     `json:"total_bet"`
	TotalPayout     int     `json:"total_payout"`
	NetProfit       int     `json:"net_profit"`
	WinRate         float64 `json:"win_rate"` // Percentage
	MegaJackpotsHit int     `json:"mega_jackpots_hit"`
	BiggestWin      int     `json:"biggest_win"`
	Period          string  `json:"period,omitempty"`
}
