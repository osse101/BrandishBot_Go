package utils

import (
	"crypto/rand"
	"math"
	"math/big"
	mrand "math/rand"
)

// RandomFloat returns a random float64 between 0.0 and 1.0
func RandomFloat() float64 {
	return mrand.Float64() //nolint:gosec // Game logic randomness, not security critical
}

// RandomInt returns a random integer between min and max (inclusive)
func RandomInt(min, max int) int {
	if min > max {
		return min
	}
	return mrand.Intn(max-min+1) + min //nolint:gosec // Game logic randomness, not security critical
}

// SecureRandomInt returns a cryptographically secure random integer in [0, max).
// It is intended for security-sensitive operations like gambling tie-breaks.
// It panics if max <= 0.
func SecureRandomInt(max int) int {
	if max <= 0 {
		panic("max must be > 0")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		// Panic on failure (e.g. lack of entropy), as proceeding with
		// insecure randomness is not an option for security-sensitive logic.
		panic(err)
	}
	return int(n.Int64())
}

// SecureRandomIntRange returns a cryptographically secure random integer between min and max (inclusive).
func SecureRandomIntRange(min, max int) int {
	if min > max {
		return min
	}
	return SecureRandomInt(max-min+1) + min
}

// SecureRandomFloat returns a cryptographically secure random float64 in [0.0, 1.0).
func SecureRandomFloat() float64 {
	// 1<<53 is the maximum integer representable exactly in float64 without loss of precision (for the significand)
	max := big.NewInt(1 << 53)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}
	return float64(n.Int64()) / float64(1<<53)
}

// Geometric returns a number sampled from a geometric distribution with probability p.
// It returns the number of failures before the first success.
// Supported range for p is (0, 1]. If p <= 0, it returns MaxInt (infinite wait).
// If p >= 1, it returns 0 (immediate success).
func Geometric(p float64) int {
	if p >= 1 {
		return 0
	}
	if p <= 0 {
		return math.MaxInt
	}

	// Geometric distribution sampling using inverse transform method:
	// k = floor(ln(U) / ln(1-p))
	// where U is uniform in (0, 1]

	u := 1.0 - SecureRandomFloat() // SecureRandomFloat returns [0, 1), so u is (0, 1]

	// Avoid log(0) just in case (though u should never be 0 with 1.0 - [0, 1))
	if u <= 0 {
		u = 1.0e-10
	}

	return int(math.Floor(math.Log(u) / math.Log(1.0-p)))
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
