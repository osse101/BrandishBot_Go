package slots

// Symbol constants
const (
	SymbolLemon   = "LEMON"
	SymbolCherry  = "CHERRY"
	SymbolBell    = "BELL"
	SymbolBar     = "BAR"
	SymbolSeven   = "SEVEN"
	SymbolDiamond = "DIAMOND"
	SymbolStar    = "STAR"
)

// Betting limits
const (
	MinBetAmount = 10
	MaxBetAmount = 10000
)

// Thresholds for special triggers
const (
	BigWinThreshold    = 10.0 // 10x bet triggers big win
	JackpotThreshold   = 50.0 // 50x bet triggers jackpot
	TwoMatchMultiplier = 0.1  // Consolation prize for 2 matching symbols
)

// Symbol weights for weighted random selection (out of 1000)
var SymbolWeights = map[string]int{
	SymbolLemon:   400, // 40%
	SymbolCherry:  250, // 25%
	SymbolBell:    150, // 15%
	SymbolBar:     95,  // 9.5%
	SymbolSeven:   70,  // 7%
	SymbolDiamond: 25,  // 2.5%
	SymbolStar:    10,  // 1%
}

// PayoutMultipliers defines the payout for 3 matching symbols
var PayoutMultipliers = map[string]float64{
	SymbolLemon:   0.5,   // Lose half bet
	SymbolCherry:  2.0,   // Double bet
	SymbolBell:    5.0,   // 5x payout
	SymbolBar:     10.0,  // 10x payout
	SymbolSeven:   25.0,  // 25x payout
	SymbolDiamond: 100.0, // 100x jackpot
	SymbolStar:    500.0, // 500x mega jackpot
}

// Trigger types for visual effects
const (
	TriggerNormal      = "normal"
	TriggerBigWin      = "big_win"
	TriggerJackpot     = "jackpot"
	TriggerMegaJackpot = "mega_jackpot"
)

// Engagement metric types
const (
	MetricSlotsSpin    = "slots_spin"
	MetricSlotsWin     = "slots_win"
	MetricSlotsBigWin  = "slots_big_win"
	MetricSlotsJackpot = "slots_jackpot"
)
