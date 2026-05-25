package slots

import (
	"testing"
)

func TestSelectWeightedSymbol(t *testing.T) {
	tests := []struct {
		name       string
		rollValue  int
		wantSymbol string
	}{
		{"Lemon (lower bound)", 0, SymbolLemon},
		{"Lemon (upper bound)", 399, SymbolLemon},
		{"Cherry (lower bound)", 400, SymbolCherry},
		{"Cherry (upper bound)", 649, SymbolCherry},
		{"Bell", 650, SymbolBell},
		{"Bar", 800, SymbolBar},
		{"Seven", 895, SymbolSeven},
		{"Diamond", 965, SymbolDiamond},
		{"Star", 990, SymbolStar},
		{"Star (upper bound)", 999, SymbolStar},
		{"Out of bounds (fallback to Lemon)", 1000, SymbolLemon},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				rng: func(max int) int {
					return tt.rollValue
				},
			}
			if got := s.selectWeightedSymbol(); got != tt.wantSymbol {
				t.Errorf("service.selectWeightedSymbol() = %v, want %v", got, tt.wantSymbol)
			}
		})
	}
}

func TestCalculatePayout(t *testing.T) {
	tests := []struct {
		name        string
		reel1       string
		reel2       string
		reel3       string
		betAmount   int
		wantAmount  int
		wantMult    float64
		wantTrigger string
	}{
		{
			name:  "Three of a kind (Lemon)",
			reel1: SymbolLemon, reel2: SymbolLemon, reel3: SymbolLemon,
			betAmount:   100,
			wantAmount:  50,
			wantMult:    0.5,
			wantTrigger: TriggerNormal,
		},
		{
			name:  "Three of a kind (Star) - Mega Jackpot",
			reel1: SymbolStar, reel2: SymbolStar, reel3: SymbolStar,
			betAmount:   100,
			wantAmount:  50000,
			wantMult:    500.0,
			wantTrigger: TriggerMegaJackpot,
		},
		{
			name:  "Two of a kind (1 and 2)",
			reel1: SymbolLemon, reel2: SymbolLemon, reel3: SymbolCherry,
			betAmount:   100,
			wantAmount:  10,
			wantMult:    0.1,
			wantTrigger: TriggerNormal,
		},
		{
			name:  "Two of a kind (2 and 3)",
			reel1: SymbolCherry, reel2: SymbolLemon, reel3: SymbolLemon,
			betAmount:   100,
			wantAmount:  10,
			wantMult:    0.1,
			wantTrigger: TriggerNormal,
		},
		{
			name:  "Two of a kind (1 and 3)",
			reel1: SymbolLemon, reel2: SymbolCherry, reel3: SymbolLemon,
			betAmount:   100,
			wantAmount:  10,
			wantMult:    0.1,
			wantTrigger: TriggerNormal,
		},
		{
			name:  "No match",
			reel1: SymbolLemon, reel2: SymbolCherry, reel3: SymbolBell,
			betAmount:   100,
			wantAmount:  0,
			wantMult:    0.0,
			wantTrigger: TriggerNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{}
			gotAmount, gotMult, gotTrigger := s.calculatePayout(tt.reel1, tt.reel2, tt.reel3, tt.betAmount)
			if gotAmount != tt.wantAmount {
				t.Errorf("calculatePayout() gotAmount = %v, want %v", gotAmount, tt.wantAmount)
			}
			if gotMult != tt.wantMult {
				t.Errorf("calculatePayout() gotMult = %v, want %v", gotMult, tt.wantMult)
			}
			if gotTrigger != tt.wantTrigger {
				t.Errorf("calculatePayout() gotTrigger = %v, want %v", gotTrigger, tt.wantTrigger)
			}
		})
	}
}

func TestDetermineWinType(t *testing.T) {
	tests := []struct {
		name       string
		multiplier float64
		want       string
	}{
		{"Mega Jackpot lower bound", 100.0, TriggerMegaJackpot},
		{"Mega Jackpot above", 500.0, TriggerMegaJackpot},
		{"Jackpot lower bound", 50.0, TriggerJackpot},
		{"Jackpot above", 75.0, TriggerJackpot},
		{"Big Win lower bound", 10.0, TriggerBigWin},
		{"Big Win above", 25.0, TriggerBigWin},
		{"Normal", 5.0, TriggerNormal},
		{"Normal low", 0.1, TriggerNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{}
			if got := s.determineWinType(tt.multiplier); got != tt.want {
				t.Errorf("determineWinType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name      string
		reel1     string
		reel2     string
		reel3     string
		betAmount int
		amount    int
		trigger   string
		want      string
	}{
		{
			name:  "Loss (amount 0)",
			reel1: SymbolLemon, reel2: SymbolCherry, reel3: SymbolBell,
			betAmount: 100,
			amount:    0,
			trigger:   TriggerNormal,
			want:      "Better luck next time! You lost 100 money.",
		},
		{
			name:  "Mega Jackpot",
			reel1: SymbolStar, reel2: SymbolStar, reel3: SymbolStar,
			betAmount: 10,
			amount:    5000,
			trigger:   TriggerMegaJackpot,
			want:      "🌟 MEGA JACKPOT! 🌟 You won 5000 money (net +4990)!",
		},
		{
			name:  "Jackpot",
			reel1: SymbolDiamond, reel2: SymbolDiamond, reel3: SymbolDiamond,
			betAmount: 10,
			amount:    1000,
			trigger:   TriggerJackpot,
			want:      "💎 JACKPOT! 💎 You won 1000 money (net +990)!",
		},
		{
			name:  "Big Win",
			reel1: SymbolBar, reel2: SymbolBar, reel3: SymbolBar,
			betAmount: 10,
			amount:    100,
			trigger:   TriggerBigWin,
			want:      "🎉 BIG WIN! You won 100 money (net +90)!",
		},
		{
			name:  "Normal Win - Net Positive",
			reel1: SymbolCherry, reel2: SymbolCherry, reel3: SymbolCherry,
			betAmount: 10,
			amount:    20,
			trigger:   TriggerNormal,
			want:      "You won 20 money (net +10)!",
		},
		{
			name:  "Normal Win - Break Even (net 0)",
			reel1: "A", reel2: "A", reel3: "A",
			betAmount: 10,
			amount:    10,
			trigger:   TriggerNormal,
			want:      "You broke even! 10 money returned.",
		},
		{
			name:  "Two Match - Consolation (amount > 0, net < 0)",
			reel1: SymbolLemon, reel2: SymbolLemon, reel3: SymbolBell,
			betAmount: 100,
			amount:    10,
			trigger:   TriggerNormal,
			want:      "Consolation! You got 10 back. (net -90)",
		},
		{
			name:  "Three Match - Net Negative",
			reel1: SymbolLemon, reel2: SymbolLemon, reel3: SymbolLemon,
			betAmount: 100,
			amount:    50,
			trigger:   TriggerNormal,
			want:      "No luck! You won 50 money (net -50).",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{}
			got := s.formatMessage(tt.reel1, tt.reel2, tt.reel3, tt.betAmount, tt.amount, tt.trigger)
			if got != tt.want {
				t.Errorf("formatMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSpinReels(t *testing.T) {
	rolls := []int{100, 500, 700}
	rollIdx := 0
	s := &service{
		rng: func(max int) int {
			val := rolls[rollIdx]
			rollIdx++
			return val
		},
	}

	r1, r2, r3 := s.spinReels()
	if r1 != SymbolLemon || r2 != SymbolCherry || r3 != SymbolBell {
		t.Errorf("spinReels() = %v, %v, %v, want Lemon, Cherry, Bell", r1, r2, r3)
	}
}
