package slots

type Symbol string
type ResultType string

const (
	SymbolLemon   Symbol = "LEMON"
	SymbolCherry  Symbol = "CHERRY"
	SymbolBell    Symbol = "BELL"
	SymbolBar     Symbol = "BAR"
	SymbolSeven   Symbol = "SEVEN"
	SymbolDiamond Symbol = "DIAMOND"
	SymbolStar    Symbol = "STAR"
)

var AllSymbols = []Symbol{SymbolLemon, SymbolCherry, SymbolBell, SymbolBar, SymbolSeven, SymbolDiamond, SymbolStar}

const (
	MinBetAmount = 10
	MaxBetAmount = 10000
)

const (
	BigWinThreshold    = 10.0
	JackpotThreshold   = 50.0
	TwoMatchMultiplier = 0.1
)

const (
	ResultNoMatch           ResultType = "no_match"
	ResultLemonTwoMatch     ResultType = "lemon_two_match"
	ResultLemonThreeMatch   ResultType = "lemon_three_match"
	ResultCherryThreeMatch  ResultType = "cherry_three_match"
	ResultBellThreeMatch    ResultType = "bell_three_match"
	ResultBarThreeMatch     ResultType = "bar_three_match"
	ResultSevenThreeMatch   ResultType = "seven_three_match"
	ResultDiamondThreeMatch ResultType = "diamond_three_match"
	ResultStarThreeMatch    ResultType = "star_three_match"
)

var ResultWeights = map[ResultType]float64{
	ResultNoMatch:          0.45,
	ResultLemonTwoMatch:    0.25,
	ResultLemonThreeMatch:  0.15,
	ResultCherryThreeMatch: 0.06,
	ResultBellThreeMatch:   0.035,
	ResultBarThreeMatch:    0.02,
	ResultSevenThreeMatch:  0.01,
	ResultStarThreeMatch:   0.005,
}

var ResultPayouts = map[ResultType]float64{
	ResultNoMatch:          0.0,
	ResultLemonTwoMatch:    0.25,
	ResultLemonThreeMatch:  0.5,
	ResultCherryThreeMatch: 3.0,
	ResultBellThreeMatch:   8.0,
	ResultBarThreeMatch:    15.0,
	ResultSevenThreeMatch:  30.0,
	ResultStarThreeMatch:   75.0,
}

const (
	TriggerNormal      = "normal"
	TriggerBigWin      = "big_win"
	TriggerJackpot     = "jackpot"
	TriggerMegaJackpot = "mega_jackpot"
)
