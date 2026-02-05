package expedition

import "math"

// ScaleEffect scales a base count by party size using the divisor
// Formula: baseCount * ceil(partySize / divisor)
func ScaleEffect(baseCount int, partySize int, divisor int) int {
	if divisor <= 0 || partySize <= 0 {
		return baseCount
	}
	return baseCount * int(math.Ceil(float64(partySize)/float64(divisor)))
}
