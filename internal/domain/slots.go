package domain

// SlotsResult represents the outcome of a slots spin
type SlotsResult struct {
	UserID           string  `json:"user_id"`
	Username         string  `json:"username"`
	Reel1            string  `json:"reel1"`             // Symbol name
	Reel2            string  `json:"reel2"`             // Symbol name
	Reel3            string  `json:"reel3"`             // Symbol name
	BetAmount        int     `json:"bet_amount"`        // Amount wagered
	PayoutAmount     int     `json:"payout_amount"`     // Amount won (0 if loss)
	PayoutMultiplier float64 `json:"payout_multiplier"` // Multiplier applied to bet
	Message          string  `json:"message"`           // User-facing result text
	IsWin            bool    `json:"is_win"`            // True if payout > 0
	IsNearMiss       bool    `json:"is_near_miss"`      // True if 2/3 symbols match
	TriggerType      string  `json:"trigger_type"`      // "normal", "big_win", "jackpot", "mega_jackpot"
}

// SlotsCompletedPayload is the event payload for slots.completed events
type SlotsCompletedPayload struct {
	UserID           string  `json:"user_id"`
	Username         string  `json:"username"`
	BetAmount        int     `json:"bet_amount"`
	Reel1            string  `json:"reel1"`
	Reel2            string  `json:"reel2"`
	Reel3            string  `json:"reel3"`
	PayoutAmount     int     `json:"payout_amount"`
	PayoutMultiplier float64 `json:"payout_multiplier"`
	TriggerType      string  `json:"trigger_type"`
	IsWin            bool    `json:"is_win"`
	IsNearMiss       bool    `json:"is_near_miss"`
}
