package domain

// GambleParticipantOutcome holds per-participant outcome data for gamble events
type GambleParticipantOutcome struct {
	UserID         string `json:"user_id"`
	Score          int64  `json:"score"`
	LootboxCount   int    `json:"lootbox_count"`
	IsWinner       bool   `json:"is_winner"`
	IsNearMiss     bool   `json:"is_near_miss"`
	IsCritFail     bool   `json:"is_crit_fail"`
	IsTieBreakLost bool   `json:"is_tie_break_lost"`
}

// GambleCompletedPayloadV2 enriches V1 with per-participant outcome data
type GambleCompletedPayloadV2 struct {
	GambleID         string                     `json:"gamble_id"`
	WinnerID         string                     `json:"winner_id"`
	TotalValue       int64                      `json:"total_value"`
	ParticipantCount int                        `json:"participant_count"`
	Participants     []GambleParticipantOutcome `json:"participants"`
	Timestamp        int64                      `json:"timestamp"`
}

// GambleParticipatedPayload fires when a user starts or joins a gamble
type GambleParticipatedPayload struct {
	GambleID     string `json:"gamble_id"`
	UserID       string `json:"user_id"`
	LootboxCount int    `json:"lootbox_count"`
	Source       string `json:"source"` // "start" or "join"
	Timestamp    int64  `json:"timestamp"`
}

// ItemSoldPayload is the event payload for item.sold events
type ItemSoldPayload struct {
	UserID       string `json:"user_id"`
	ItemName     string `json:"item_name"`
	ItemCategory string `json:"item_category"`
	Quantity     int    `json:"quantity"`
	TotalValue   int    `json:"total_value"`
	Timestamp    int64  `json:"timestamp"`
}

// ItemBoughtPayload is the event payload for item.bought events
type ItemBoughtPayload struct {
	UserID       string `json:"user_id"`
	ItemName     string `json:"item_name"`
	ItemCategory string `json:"item_category"`
	Quantity     int    `json:"quantity"`
	TotalValue   int    `json:"total_value"`
	Timestamp    int64  `json:"timestamp"`
}

// SearchPerformedPayload is the event payload for search.performed events
type SearchPerformedPayload struct {
	UserID         string `json:"user_id"`
	Success        bool   `json:"success"`
	IsCritical     bool   `json:"is_critical"`
	IsNearMiss     bool   `json:"is_near_miss"`
	IsCriticalFail bool   `json:"is_critical_fail"`
	IsFirstDaily   bool   `json:"is_first_daily"`
	XPAmount       int    `json:"xp_amount"`
	ItemName       string `json:"item_name,omitempty"`
	Quantity       int    `json:"quantity,omitempty"`
	Timestamp      int64  `json:"timestamp"`
}

// HarvestCompletedPayload is the event payload for harvest.completed events
type HarvestCompletedPayload struct {
	UserID       string  `json:"user_id"`
	HoursElapsed float64 `json:"hours_elapsed"`
	XPAmount     int     `json:"xp_amount"`
	Spoiled      bool    `json:"spoiled"`
	Timestamp    int64   `json:"timestamp"`
}

// CompostHarvestedPayload is the event payload for compost.harvested events
type CompostHarvestedPayload struct {
	UserID     string `json:"user_id"`
	InputValue int    `json:"input_value"`
	XPAmount   int    `json:"xp_amount"`
	IsSludge   bool   `json:"is_sludge"`
	Timestamp  int64  `json:"timestamp"`
}

// ExpeditionRewardedPayload is the event payload for expedition.rewarded events
type ExpeditionRewardedPayload struct {
	ExpeditionID string         `json:"expedition_id"`
	UserID       string         `json:"user_id"`
	JobXP        map[string]int `json:"job_xp"` // jobKey -> xp amount
	Timestamp    int64          `json:"timestamp"`
}

// PredictionParticipantPayload is the event payload for prediction.participated events
type PredictionParticipantPayload struct {
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
	XP         int    `json:"xp"`
	IsWinner   bool   `json:"is_winner"`
	Timestamp  int64  `json:"timestamp"`
}

// QuestClaimedPayloadV1 is the typed event payload for quest.claimed events
type QuestClaimedPayloadV1 struct {
	UserID      string `json:"user_id"`
	QuestKey    string `json:"quest_key"`
	QuestID     int64  `json:"quest_id"`
	RewardMoney int    `json:"reward_money"`
	RewardXP    int    `json:"reward_xp"`
	Timestamp   int64  `json:"timestamp"`
}

// ItemUsedPayload is the event payload for item.used events
type ItemUsedPayload struct {
	UserID    string                 `json:"user_id"`
	ItemName  string                 `json:"item_name"`
	Quantity  int                    `json:"quantity"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp int64                  `json:"timestamp"`
}
