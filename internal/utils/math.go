package utils

import (
	"crypto/rand"
	"math"
	"math/big"
	mrand "math/rand/v2"
)

func RandomFloat() float64 {
	return mrand.Float64() //nolint:gosec // Game logic randomness, not security critical
}

func RandomInt(min, max int) int {
	if min > max {
		return min
	}
	return mrand.IntN(max-min+1) + min //nolint:gosec // Game logic randomness, not security critical
}

func SecureRandomInt(max int) int {
	if max <= 0 {
		panic("max must be > 0")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}
	return int(n.Int64())
}

func SecureRandomIntRange(min, max int) int {
	if min > max {
		return min
	}
	return SecureRandomInt(max-min+1) + min
}

func SecureRandomFloat() float64 {
	max := big.NewInt(1 << 53)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}
	return float64(n.Int64()) / float64(1<<53)
}

func Geometric(p float64) int {
	if p >= 1 {
		return 0
	}
	if p <= 0 {
		return math.MaxInt
	}

	u := 1.0 - SecureRandomFloat()

	if u <= 0 {
		u = 1.0e-10
	}

	return int(math.Floor(math.Log(u) / math.Log(1.0-p)))
}

func DiminishingReturns(value, scale float64) float64 {
	if value < 0 {
		return 0
	}
	return value / (value + scale)
}
