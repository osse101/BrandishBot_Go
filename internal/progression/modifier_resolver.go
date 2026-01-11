package progression

import "math"

// ApplyModifier calculates the final value based on modifier type and level
func ApplyModifier(modifier *ValueModifier, baseValue float64) float64 {
	level := modifier.CurrentLevel

	var result float64

	switch modifier.ModifierType {
	case ModifierTypeMultiplicative:
		// value * (1 + level * perLevelValue)
		// Example: 1.0 * (1 + 2 * 0.1) = 1.2 at level 2
		result = baseValue * (1 + float64(level)*modifier.PerLevelValue)

	case ModifierTypeLinear:
		// value + (level * perLevelValue)
		// Example: 500 + (2 * 100) = 700 at level 2
		result = baseValue + (float64(level) * modifier.PerLevelValue)

	case ModifierTypeFixed:
		// baseValue + (level * perLevelValue)
		// Ignores input baseValue, calculates from modifier's base
		// Example: 0 + (2 * 10) = 20 at level 2
		result = modifier.BaseValue + (float64(level) * modifier.PerLevelValue)

	case ModifierTypePercentage:
		// baseValue + (level * perLevelValue)
		// Example: 0.05 + (2 * 0.01) = 0.07 at level 2
		result = baseValue + (float64(level) * modifier.PerLevelValue)

	default:
		return baseValue
	}

	// Apply bounds
	if modifier.MaxValue != nil {
		result = math.Min(result, *modifier.MaxValue)
	}
	if modifier.MinValue != nil {
		result = math.Max(result, *modifier.MinValue)
	}

	return result
}
