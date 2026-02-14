package prediction

import (
	"testing"
)

func TestCalculateContribution(t *testing.T) {
	s := &service{}

	tests := []struct {
		name            string
		points          int
		wantMin         int
		wantMax         int
		wantApproximate int
	}{
		{
			name:            "Zero points",
			points:          0,
			wantMin:         0,
			wantMax:         0,
			wantApproximate: 0,
		},
		{
			name:            "Negative points",
			points:          -100,
			wantMin:         0,
			wantMax:         0,
			wantApproximate: 0,
		},
		{
			name:            "1k points",
			points:          1000,
			wantMin:         0,
			wantMax:         0,
			wantApproximate: 0,
		},
		{
			name:            "10k points",
			points:          10000,
			wantMin:         0,
			wantMax:         2,
			wantApproximate: 1,
		},
		{
			name:            "100k points",
			points:          100000,
			wantMin:         25,
			wantMax:         27,
			wantApproximate: 26,
		},
		{
			name:            "1M points",
			points:          1000000,
			wantMin:         49,
			wantMax:         50,
			wantApproximate: 50,
		},
		{
			name:            "500 points (less than 1k)",
			points:          500,
			wantMin:         0,
			wantMax:         0,
			wantApproximate: 0,
		},
		{
			name:            "50k points",
			points:          50000,
			wantMin:         15,
			wantMax:         20,
			wantApproximate: 18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.calculateContribution(tt.points)

			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("calculateContribution(%d) = %d, want between %d and %d (approximately %d)",
					tt.points, got, tt.wantMin, tt.wantMax, tt.wantApproximate)
			}

			// Also check approximate value for more precise tests
			if tt.wantApproximate > 0 {
				diff := got - tt.wantApproximate
				if diff < 0 {
					diff = -diff
				}
				if diff > 1 {
					t.Errorf("calculateContribution(%d) = %d, want approximately %d (tolerance ±1)",
						tt.points, got, tt.wantApproximate)
				}
			}
		})
	}
}

func TestCalculateContribution_EdgeCases(t *testing.T) {
	s := &service{}

	tests := []struct {
		name   string
		points int
	}{
		{
			name:   "Very large value",
			points: 10000000,
		},
		{
			name:   "Very small positive value",
			points: 1,
		},
		{
			name:   "Exact power of 10",
			points: 100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.calculateContribution(tt.points)

			// Should always return a positive value for positive input
			if tt.points > 0 && got < 0 {
				t.Errorf("calculateContribution(%d) = %d, expected positive value", tt.points, got)
			}

			// Should always return 0 for non-positive input
			if tt.points <= 0 && got != 0 {
				t.Errorf("calculateContribution(%d) = %d, expected 0", tt.points, got)
			}
		})
	}
}

// TestCalculateContribution_Monotonic verifies that the conversion function is monotonically increasing
func TestCalculateContribution_Monotonic(t *testing.T) {
	s := &service{}

	// Test that larger inputs always produce larger or equal outputs
	testPoints := []int{100, 1000, 5000, 10000, 50000, 100000, 500000, 1000000}

	var lastResult int
	for _, points := range testPoints {
		result := s.calculateContribution(points)
		if result < lastResult {
			t.Errorf("Non-monotonic: contribution(%d) = %d < contribution(previous) = %d",
				points, result, lastResult)
		}
		lastResult = result
	}
}
