package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDiminishingReturns verifies the diminishing returns formula
func TestDiminishingReturns(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		scale    float64
		expected float64
		desc     string
	}{
		{
			name:     "zero value returns zero",
			value:    0,
			scale:    100,
			expected: 0,
			desc:     "No input should give no output",
		},
		{
			name:     "negative value returns zero",
			value:    -50,
			scale:    100,
			expected: 0,
			desc:     "Negative values should be bounded to zero",
		},
		{
			name:     "value equals scale gives 0.5",
			value:    100,
			scale:    100,
			expected: 0.5,
			desc:     "When value equals scale, output should be 50% of maximum",
		},
		{
			name:     "very small value approaches zero",
			value:    0.1,
			scale:    100,
			expected: 0.001,
			desc:     "Small values should give proportionally small returns",
		},
		{
			name:     "very large value approaches 1",
			value:    10000,
			scale:    10,
			expected: 0.9990009990009990,
			desc:     "Large values should asymptotically approach 1",
		},
		{
			name:     "double the scale halves the effectiveness",
			value:    50,
			scale:    100,
			expected: 0.3333333333333333,
			desc:     "Higher scale means slower progression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiminishingReturns(tt.value, tt.scale)
			assert.InDelta(t, tt.expected, result, 0.0001, tt.desc)
			
			// All results should be between 0 and 1
			assert.GreaterOrEqual(t, result, 0.0, "Result should never be negative")
			assert.LessOrEqual(t, result, 1.0, "Result should never exceed 1")
		})
	}
}

// TestDiminishingReturns_Properties verifies mathematical properties
func TestDiminishingReturns_Properties(t *testing.T) {
	t.Run("monotonically increasing", func(t *testing.T) {
		// Increasing input should always increase output
		scale := 100.0
		prev := DiminishingReturns(0, scale)
		
		for value := 10.0; value <= 1000; value += 10 {
			current := DiminishingReturns(value, scale)
			assert.Greater(t, current, prev,
				"Output should increase as input increases")
			prev = current
		}
	})

	t.Run("bounded between 0 and 1", func(t *testing.T) {
		// Test with various scales and values
		testCases := []struct {
			value float64
			scale float64
		}{
			{0, 1},
			{100, 50},
			{1000, 100},
			{0.001, 0.001},
			{999999, 1},
		}

		for _, tc := range testCases {
			result := DiminishingReturns(tc.value, tc.scale)
			assert.GreaterOrEqual(t, result, 0.0,
				"Should be >= 0 for value=%v, scale=%v", tc.value, tc.scale)
			assert.LessOrEqual(t, result, 1.0,
				"Should be <= 1 for value=%v, scale=%v", tc.value, tc.scale)
		}
	})

	t.Run("asymptotic behavior toward 1", func(t *testing.T) {
		scale := 10.0
		
		// As value increases, should get closer to 1
		result100 := DiminishingReturns(100, scale)
		result1000 := DiminishingReturns(1000, scale)
		result10000 := DiminishingReturns(10000, scale)
		
		assert.Greater(t, result1000, result100)
		assert.Greater(t, result10000, result1000)
		
		// Very large values should be very close to 1
		assert.Greater(t, result10000, 0.99)
	})
}

// TestRandomInt tests the random integer generator
func TestRandomInt(t *testing.T) {
	t.Run("returns value within range", func(t *testing.T) {
		min, max := 1, 10
		
		// Test multiple times to catch probabilistic issues
		for i := 0; i < 100; i++ {
			result := RandomInt(min, max)
			assert.GreaterOrEqual(t, result, min,
				"Result should be >= min")
			assert.LessOrEqual(t, result, max,
				"Result should be <= max")
		}
	})

	t.Run("handles min equals max", func(t *testing.T) {
		value := 42
		result := RandomInt(value, value)
		assert.Equal(t, value, result,
			"Should return the value when min==max")
	})

	t.Run("handles inverted range gracefully", func(t *testing.T) {
		// When min > max, should return min
		result := RandomInt(10, 5)
		assert.Equal(t, 10, result,
			"Should return min when min > max")
	})

	t.Run("handles negative ranges", func(t *testing.T) {
		min, max := -10, -1
		
		for i := 0; i < 50; i++ {
			result := RandomInt(min, max)
			assert.GreaterOrEqual(t, result, min)
			assert.LessOrEqual(t, result, max)
		}
	})

	t.Run("produces different values over multiple calls", func(t *testing.T) {
		// With a range of 1-100, we should see variety
		// (this could theoretically fail, but probability is extremely low)
		results := make(map[int]bool)
		
		for i := 0; i < 100; i++ {
			result := RandomInt(1, 100)
			results[result] = true
		}
		
		// We should have gotten at least 10 different values
		assert.GreaterOrEqual(t, len(results), 10,
			"Should produce varied results, not same value repeatedly")
	})
}

// TestRandomFloat tests the random float generator
func TestRandomFloat(t *testing.T) {
	t.Run("returns value between 0 and 1", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			result := RandomFloat()
			assert.GreaterOrEqual(t, result, 0.0,
				"Should be >= 0")
			assert.LessOrEqual(t, result, 1.0,
				"Should be <= 1")
		}
	})

	t.Run("produces varied results", func(t *testing.T) {
		results := make([]float64, 100)
		allSame := true
		
		for i := 0; i < 100; i++ {
			results[i] = RandomFloat()
			if i > 0 && results[i] != results[0] {
				allSame = false
			}
		}
		
		assert.False(t, allSame,
			"Should produce different values, not all identical")
	})
}
