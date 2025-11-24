package utils

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomFloat returns a random float64 between 0.0 and 1.0
func RandomFloat() float64 {
	return rand.Float64()
}

// RandomInt returns a random integer between min and max (inclusive)
func RandomInt(min, max int) int {
	if min > max {
		return min
	}
	return rand.Intn(max-min+1) + min
}

// DiminishingReturns calculates a value with diminishing returns.
// value: The input value.
// scale: The value at which the output is 50% of the maximum possible output (asymptote).
// formula: value / (value + scale) -> returns a factor between 0 and 1
// To get a result scaled to a max, multiply the result by max.
func DiminishingReturns(value, scale float64) float64 {
	if value < 0 {
		return 0
	}
	return value / (value + scale)
}
