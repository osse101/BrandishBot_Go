package slots

import (
	"testing"
)

func TestCalculatePayout(t *testing.T) {
	tests := []struct {
		name        string
		resultType  ResultType
		betAmount   int
		wantAmount  int
		wantMult    float64
		wantTrigger string
	}{
		{
			name:        "Three of a kind (Lemon)",
			resultType:  ResultLemonThreeMatch,
			betAmount:   100,
			wantAmount:  50,
			wantMult:    0.5,
			wantTrigger: TriggerNormal,
		},
		{
			name:        "Three of a kind (Star) - Mega Jackpot",
			resultType:  ResultStarThreeMatch,
			betAmount:   100,
			wantAmount:  7500,
			wantMult:    75.0,
			wantTrigger: TriggerJackpot,
		},
		{
			name:        "Two of a kind (1 and 2)",
			resultType:  ResultLemonTwoMatch,
			betAmount:   100,
			wantAmount:  25,
			wantMult:    0.25,
			wantTrigger: TriggerNormal,
		},
		{
			name:        "Two of a kind (2 and 3)",
			resultType:  ResultLemonTwoMatch,
			betAmount:   100,
			wantAmount:  25,
			wantMult:    0.25,
			wantTrigger: TriggerNormal,
		},
		{
			name:        "Two of a kind (1 and 3)",
			resultType:  ResultNoMatch,
			betAmount:   100,
			wantAmount:  0,
			wantMult:    0.0,
			wantTrigger: TriggerNormal,
		},
		{
			name:        "No match",
			resultType:  ResultNoMatch,
			betAmount:   100,
			wantAmount:  0,
			wantMult:    0.0,
			wantTrigger: TriggerNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{}
			gotAmount, gotMult, gotTrigger := s.calculatePayout(tt.resultType, tt.betAmount)
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
