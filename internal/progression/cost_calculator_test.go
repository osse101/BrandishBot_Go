package progression

import (
	"math"
	"testing"
)

func TestCalculateUnlockCost(t *testing.T) {
	tests := []struct {
		name    string
		tier    int
		size    NodeSize
		want    int
		wantErr bool
	}{
		// Foundation tier (auto-unlocked, tier multiplier = 1.50^0 = 1)
		{"tier 0 small", 0, NodeSizeSmall, expectedCost(BaseCostSmall, 0), false},
		{"tier 0 medium", 0, NodeSizeMedium, expectedCost(BaseCostMedium, 0), false},
		{"tier 0 large", 0, NodeSizeLarge, expectedCost(BaseCostLarge, 0), false},

		// Tier 1 (1.50^1 = 1.5)
		{"tier 1 small", 1, NodeSizeSmall, expectedCost(BaseCostSmall, 1), false},
		{"tier 1 medium", 1, NodeSizeMedium, expectedCost(BaseCostMedium, 1), false},
		{"tier 1 large", 1, NodeSizeLarge, expectedCost(BaseCostLarge, 1), false},

		// Tier 2 (1.50^2 = 2.25)
		{"tier 2 small", 2, NodeSizeSmall, expectedCost(BaseCostSmall, 2), false},
		{"tier 2 medium", 2, NodeSizeMedium, expectedCost(BaseCostMedium, 2), false},
		{"tier 2 large", 2, NodeSizeLarge, expectedCost(BaseCostLarge, 2), false},

		// Tier 3 (1.50^3 = 3.375)
		{"tier 3 small", 3, NodeSizeSmall, expectedCost(BaseCostSmall, 3), false},
		{"tier 3 medium", 3, NodeSizeMedium, expectedCost(BaseCostMedium, 3), false},
		{"tier 3 large", 3, NodeSizeLarge, expectedCost(BaseCostLarge, 3), false},

		// Tier 4 (1.50^4 = 5.0625)
		{"tier 4 small", 4, NodeSizeSmall, expectedCost(BaseCostSmall, 4), false},
		{"tier 4 medium", 4, NodeSizeMedium, expectedCost(BaseCostMedium, 4), false},
		{"tier 4 large", 4, NodeSizeLarge, expectedCost(BaseCostLarge, 4), false},

		// Higher tiers (demonstrating exponential scaling)
		{"tier 5 small", 5, NodeSizeSmall, expectedCost(BaseCostSmall, 5), false},
		{"tier 10 small", 10, NodeSizeSmall, expectedCost(BaseCostSmall, 10), false},

		// Error cases
		{"invalid tier negative", -1, NodeSizeSmall, 0, true},
		{"invalid size", 1, "huge", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateUnlockCost(tt.tier, tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateUnlockCost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CalculateUnlockCost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateTier(t *testing.T) {
	tests := []struct {
		name    string
		tier    int
		wantErr bool
	}{
		{"tier 0 valid", 0, false},
		{"tier 1 valid", 1, false},
		{"tier 2 valid", 2, false},
		{"tier 3 valid", 3, false},
		{"tier 4 valid", 4, false},
		{"tier 5 valid", 5, false},
		{"tier 10 valid", 10, false},
		{"tier 100 valid", 100, false},
		{"tier -1 invalid", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTier(tt.tier)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTier() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSize(t *testing.T) {
	tests := []struct {
		name    string
		size    string
		wantErr bool
	}{
		{"small valid", "small", false},
		{"medium valid", "medium", false},
		{"large valid", "large", false},
		{"huge invalid", "huge", true},
		{"tiny invalid", "tiny", true},
		{"empty invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// expectedCost is a helper to calculate unlock cost using the same formula as the implementation
func expectedCost(baseCost int, tier int) int {
	return int(math.Round(float64(baseCost) * math.Pow(TierScalingFactor, float64(tier))))
}
