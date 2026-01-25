package progression

import (
	"fmt"
)

// NodeSize represents the size/scope of a progression node
type NodeSize string

const (
	NodeSizeSmall  NodeSize = "small"
	NodeSizeMedium NodeSize = "medium"
	NodeSizeLarge  NodeSize = "large"
)

// Base unlock costs per tier
const (
	TierFoundation   = 0    // Tier 0: Auto-unlocked root nodes
	TierBasic        = 500  // Tier 1: First unlocks
	TierIntermediate = 1000 // Tier 2: Standard features
	TierAdvanced     = 2000 // Tier 3: Complex features
	TierEndgame      = 3000 // Tier 4: Late-game content
)

var baseCosts = map[int]int{
	0: TierFoundation,
	1: TierBasic,
	2: TierIntermediate,
	3: TierAdvanced,
	4: TierEndgame,
}

// Size multipliers: small:medium:large = 1:2:4
var sizeMultipliers = map[NodeSize]float64{
	NodeSizeSmall:  1.0,
	NodeSizeMedium: 2.0,
	NodeSizeLarge:  4.0,
}

// CalculateUnlockCost computes the unlock cost for a node based on tier, size, and level
// Formula: baseCost[tier] * sizeMultiplier[size]
// Note: MaxLevel doesn't affect cost - each level costs the same
func CalculateUnlockCost(tier int, size NodeSize) (int, error) {
	// Validate tier
	baseCost, ok := baseCosts[tier]
	if !ok {
		return 0, fmt.Errorf("invalid tier %d: must be 0-4", tier)
	}

	// Validate size
	multiplier, ok := sizeMultipliers[size]
	if !ok {
		return 0, fmt.Errorf("invalid size %s: must be small, medium, or large", size)
	}

	// Calculate final cost
	cost := float64(baseCost) * multiplier

	return int(cost), nil
}

// ValidateTier checks if a tier value is valid (0-4)
func ValidateTier(tier int) error {
	if tier < 0 || tier > 4 {
		return fmt.Errorf("tier must be between 0 and 4, got %d", tier)
	}
	return nil
}

// ValidateSize checks if a size value is valid
func ValidateSize(size string) error {
	switch NodeSize(size) {
	case NodeSizeSmall, NodeSizeMedium, NodeSizeLarge:
		return nil
	default:
		return fmt.Errorf("size must be 'small', 'medium', or 'large', got '%s'", size)
	}
}

// FormatUnlockDuration returns a human-readable string for unlock duration based on node size
func FormatUnlockDuration(size string) string {
	switch NodeSize(size) {
	case NodeSizeSmall:
		return "Short"
	case NodeSizeMedium:
		return "Medium"
	case NodeSizeLarge:
		return "Long"
	default:
		return "Mystery"
	}
}
