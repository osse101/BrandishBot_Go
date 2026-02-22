package progression

import (
	"context"
	"fmt"
)

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

// GetModifiedValue retrieves a feature value modified by progression nodes
// Returns the modified value or the baseValue on error (safe fallback)
// Supports stacking multiple modifiers with the same feature_key (multiplicative)
func (s *service) GetModifiedValue(ctx context.Context, userID string, featureKey string, baseValue float64) (float64, error) {
	// 1. Check cache first
	if cached, ok := s.modifierCache.Get(featureKey); ok {
		return cached.Value, nil
	}

	// 2. Get ALL modifiers for this feature
	modifiers, err := s.GetAllModifiersForFeature(ctx, userID, featureKey)
	if err != nil {
		// Fallback to base value on error
		return baseValue, err
	}
	if len(modifiers) == 0 {
		// No modifiers configured for this feature
		return baseValue, nil
	}

	// 3. Apply all modifiers (stacks multiplicatively)
	value := baseValue
	totalLevel := 0
	for _, modifier := range modifiers {
		value = ApplyModifier(modifier, value)
		totalLevel += modifier.CurrentLevel
	}

	// 4. Cache with total level across all modifiers
	s.modifierCache.Set(featureKey, value, totalLevel)

	return value, nil
}

// GetAllModifiersForFeature retrieves ALL active modifiers for a feature key
func (s *service) GetAllModifiersForFeature(ctx context.Context, userID string, featureKey string) ([]*ValueModifier, error) {
	configs, err := s.repo.GetBonusModifiers(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get bonus modifiers: %w", err)
	}

	modifiers := make([]*ValueModifier, 0, len(configs))
	for _, config := range configs {
		var currentLevel int

		// If this is a job bonus, get the user's specific job level
		if config.SourceType == "job" {
			if userID == "" || s.jobService == nil {
				continue // Skip user-specific bonuses if no user context available
			}
			level, err := s.jobService.GetJobLevel(ctx, userID, config.NodeKey)
			if err != nil {
				// Don't fail the entire calculation; just log and skip this modifier
				continue
			}
			currentLevel = level
		} else if config.SourceType == "progression" {
			currentLevel = config.ProgressionLevel
		}

		if currentLevel > 0 {
			modifier := &ValueModifier{
				NodeKey:       config.NodeKey,
				ModifierType:  ModifierType(config.ModifierType),
				BaseValue:     config.BaseValue,
				PerLevelValue: config.PerLevelValue,
				CurrentLevel:  currentLevel,
				MaxValue:      config.MaxValue,
				MinValue:      config.MinValue,
			}
			modifiers = append(modifiers, modifier)
		}
	}
	return modifiers, nil
}
