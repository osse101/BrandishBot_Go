package slots

const (
	SymbolLemon   = "LEMON"
	SymbolCherry  = "CHERRY"
	SymbolBell    = "BELL"
	SymbolBar     = "BAR"
	SymbolSeven   = "SEVEN"
	SymbolDiamond = "DIAMOND"
	SymbolStar    = "STAR"
)

const (
	MinBetAmount = 10
	MaxBetAmount = 10000
)

const (
	BigWinThreshold    = 10.0
	JackpotThreshold   = 50.0
	TwoMatchMultiplier = 0.1
)

var SymbolWeights = map[string]int{
	SymbolLemon:   400,
	SymbolCherry:  250,
	SymbolBell:    150,
	SymbolBar:     95,
	SymbolSeven:   70,
	SymbolDiamond: 25,
	SymbolStar:    10,
}

var PayoutMultipliers = map[string]float64{
	SymbolLemon:   0.5,
	SymbolCherry:  2.0,
	SymbolBell:    5.0,
	SymbolBar:     10.0,
	SymbolSeven:   25.0,
	SymbolDiamond: 100.0,
	SymbolStar:    500.0,
}

const (
	TriggerNormal      = "normal"
	TriggerBigWin      = "big_win"
	TriggerJackpot     = "jackpot"
	TriggerMegaJackpot = "mega_jackpot"
)
