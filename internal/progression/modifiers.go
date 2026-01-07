package progression

// ModifierType defines how a progression node modifies a value
type ModifierType string

const (
	// ModifierTypeMultiplicative: value * (1 + level * perLevelValue)
	// Example: 1.0 base * (1 + 2 * 0.1) = 1.2 at level 2
	ModifierTypeMultiplicative ModifierType = "multiplicative"

	// ModifierTypeLinear: value + (level * perLevelValue)
	// Example: 500 base + (2 * 100) = 700 at level 2
	ModifierTypeLinear ModifierType = "linear"

	// ModifierTypeFixed: baseValue + (level * perLevelValue)
	// Example: 0 base + (2 * 10) = 20 at level 2 (ignores input base)
	ModifierTypeFixed ModifierType = "fixed"

	// ModifierTypePercentage: baseValue + (level * perLevelValue)
	// Example: 0.05 base + (2 * 0.01) = 0.07 at level 2
	ModifierTypePercentage ModifierType = "percentage"
)

// ValueModifier represents a progression-based value modification with current state
type ValueModifier struct {
	NodeKey       string       // Progression node key
	ModifierType  ModifierType // How to apply the modifier
	BaseValue     float64      // Starting/default value
	PerLevelValue float64      // Value added/multiplied per level
	CurrentLevel  int          // Current node level
	MaxValue      *float64     // Optional cap
	MinValue      *float64     // Optional floor
}

// ModifierConfig is stored in progression nodes to define value modifications
// This is the JSON structure stored in the database
type ModifierConfig struct {
	FeatureKey    string       `json:"feature_key"`     // e.g., "job_xp_multiplier"
	ModifierType  ModifierType `json:"modifier_type"`   // multiplicative, linear, fixed, percentage
	PerLevelValue float64      `json:"per_level_value"` // e.g., 0.1 for +10% per level
	BaseValue     float64      `json:"base_value"`      // Default when node level = 0
	MaxValue      *float64     `json:"max_value,omitempty"`
	MinValue      *float64     `json:"min_value,omitempty"`
}
