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
	EventDistanceTraveled   EventType = "distance_traveled"
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
)

// StatsEvent represents a single tracked event
type StatsEvent struct {
	EventID   int64                  `json:"event_id"`
	UserID    string                 `json:"user_id"`
	EventType EventType              `json:"event_type"`
	EventData map[string]interface{} `json:"event_data,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// LootboxEventData represents data for lootbox jackpot/big-win events
type LootboxEventData struct {
	Item   string      `json:"item"`
	Drops  interface{} `json:"drops"` // []lootbox.DroppedItem, but using interface{} to avoid circular deps
	Value  int         `json:"value"`
	Source string      `json:"source"`
}

// ToMap converts LootboxEventData to map for compatibility with existing event recording
func (d *LootboxEventData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"item":   d.Item,
		"drops":  d.Drops,
		"value":  d.Value,
		"source": d.Source,
	}
}

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
