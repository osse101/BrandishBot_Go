package progression

import (
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
		// Foundation tier (auto-unlocked)
		{"tier 0 small", 0, NodeSizeSmall, 0, false},
		{"tier 0 medium", 0, NodeSizeMedium, 0, false},
		{"tier 0 large", 0, NodeSizeLarge, 0, false},

		// Basic tier
		{"tier 1 small", 1, NodeSizeSmall, 500, false},
		{"tier 1 medium", 1, NodeSizeMedium, 1000, false},
		{"tier 1 large", 1, NodeSizeLarge, 2000, false},

		// Intermediate tier
		{"tier 2 small", 2, NodeSizeSmall, 1000, false},
		{"tier 2 medium", 2, NodeSizeMedium, 2000, false},
		{"tier 2 large", 2, NodeSizeLarge, 4000, false},

		// Advanced tier
		{"tier 3 small", 3, NodeSizeSmall, 2000, false},
		{"tier 3 medium", 3, NodeSizeMedium, 4000, false},
		{"tier 3 large", 3, NodeSizeLarge, 8000, false},

		// Endgame tier
		{"tier 4 small", 4, NodeSizeSmall, 3000, false},
		{"tier 4 medium", 4, NodeSizeMedium, 6000, false},
		{"tier 4 large", 4, NodeSizeLarge, 12000, false},

		// Error cases
		{"invalid tier negative", -1, NodeSizeSmall, 0, true},
		{"invalid tier too high", 5, NodeSizeSmall, 0, true},
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
		{"tier -1 invalid", -1, true},
		{"tier 5 invalid", 5, true},
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
