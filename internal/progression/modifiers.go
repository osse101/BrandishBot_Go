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
func (s *service) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	// 1. Check cache first
	if cached, ok := s.modifierCache.Get(featureKey); ok {
		return cached.Value, nil
	}

	// 2. Get ALL modifiers for this feature
	modifiers, err := s.GetAllModifiersForFeature(ctx, featureKey)
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

// GetModifierForFeature retrieves the modifier configuration and current level for a feature
func (s *service) GetModifierForFeature(ctx context.Context, featureKey string) (*ValueModifier, error) {
	// Query repository for node with this feature_key
	node, currentLevel, err := s.repo.GetNodeByFeatureKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get node for feature %s: %w", featureKey, err)
	}
	if node == nil || node.ModifierConfig == nil {
		// No modifier configured for this feature
		return nil, nil
	}

	// Build ValueModifier from node's ModifierConfig
	modifier := &ValueModifier{
		NodeKey:       node.NodeKey,
		ModifierType:  ModifierType(node.ModifierConfig.ModifierType),
		BaseValue:     node.ModifierConfig.BaseValue,
		PerLevelValue: node.ModifierConfig.PerLevelValue,
		CurrentLevel:  currentLevel,
		MaxValue:      node.ModifierConfig.MaxValue,
		MinValue:      node.ModifierConfig.MinValue,
	}

	return modifier, nil
}

// GetAllModifiersForFeature retrieves ALL modifiers for a feature key
func (s *service) GetAllModifiersForFeature(ctx context.Context, featureKey string) ([]*ValueModifier, error) {
	nodes, levels, err := s.repo.GetAllNodesByFeatureKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes for feature %s: %w", featureKey, err)
	}

	modifiers := make([]*ValueModifier, 0, len(nodes))
	for i, node := range nodes {
		if node.ModifierConfig != nil {
			modifier := &ValueModifier{
				NodeKey:       node.NodeKey,
				ModifierType:  ModifierType(node.ModifierConfig.ModifierType),
				BaseValue:     node.ModifierConfig.BaseValue,
				PerLevelValue: node.ModifierConfig.PerLevelValue,
				CurrentLevel:  levels[i],
				MaxValue:      node.ModifierConfig.MaxValue,
				MinValue:      node.ModifierConfig.MinValue,
			}
			modifiers = append(modifiers, modifier)
		}
	}

	return modifiers, nil
}
