package progression

import (
	"fmt"
	"math"
)

// NodeSize represents the size/scope of a progression node
type NodeSize string

const (
	NodeSizeSmall  NodeSize = "small"
	NodeSizeMedium NodeSize = "medium"
	NodeSizeLarge  NodeSize = "large"
)

// Base unlock costs by size
var baseCosts = map[NodeSize]int{
	NodeSizeSmall:  200,
	NodeSizeMedium: 400,
	NodeSizeLarge:  800,
}

// CalculateUnlockCost computes the unlock cost for a node based on tier and size
// Formula: baseCost[size] * (1.30^tier)
// Supports arbitrary tier numbers (tier >= 0) with exponential scaling
// Tier 0 returns the base cost (1.30^0 = 1)
func CalculateUnlockCost(tier int, size NodeSize) (int, error) {
	// Validate tier
	if tier < 0 {
		return 0, fmt.Errorf("invalid tier %d: must be >= 0", tier)
	}

	// Validate size
	baseCost, ok := baseCosts[size]
	if !ok {
		return 0, fmt.Errorf("invalid size %s: must be small, medium, or large", size)
	}

	// Calculate exponential tier multiplier: 1.30^tier
	tierMultiplier := math.Pow(1.30, float64(tier))

	// Final cost = baseCost * tierMultiplier
	cost := float64(baseCost) * tierMultiplier

	return int(math.Round(cost)), nil
}

// ValidateTier checks if a tier value is valid (>= 0)
func ValidateTier(tier int) error {
	if tier < 0 {
		return fmt.Errorf("tier must be >= 0, got %d", tier)
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
