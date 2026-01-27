package progression

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// ParsePrerequisite parses a prerequisite string (static or dynamic)
// Returns: isDynamic, dynamicPrerequisite, staticKey, error
func ParsePrerequisite(prereqStr string) (isDynamic bool, dynamic *domain.DynamicPrerequisite, staticKey string, err error) {
	if !strings.HasPrefix(prereqStr, "-") {
		return false, nil, prereqStr, nil // Static prerequisite
	}

	// Dynamic prerequisite: parse -type:param1:param2...
	parts := strings.Split(prereqStr[1:], ":") // Remove "-" prefix

	switch parts[0] {
	case "nodes_unlocked_below_tier":
		if len(parts) != 3 {
			return false, nil, "", fmt.Errorf("invalid syntax: expected -nodes_unlocked_below_tier:tier:count, got %s", prereqStr)
		}
		tier, err := strconv.Atoi(parts[1])
		if err != nil {
			return false, nil, "", fmt.Errorf("invalid tier in %s: %w", prereqStr, err)
		}
		count, err := strconv.Atoi(parts[2])
		if err != nil {
			return false, nil, "", fmt.Errorf("invalid count in %s: %w", prereqStr, err)
		}
		return true, &domain.DynamicPrerequisite{Type: "nodes_unlocked_below_tier", Tier: tier, Count: count}, "", nil

	case "total_nodes_unlocked":
		if len(parts) != 2 {
			return false, nil, "", fmt.Errorf("invalid syntax: expected -total_nodes_unlocked:count, got %s", prereqStr)
		}
		count, err := strconv.Atoi(parts[1])
		if err != nil {
			return false, nil, "", fmt.Errorf("invalid count in %s: %w", prereqStr, err)
		}
		return true, &domain.DynamicPrerequisite{Type: "total_nodes_unlocked", Count: count}, "", nil

	default:
		return false, nil, "", fmt.Errorf("unknown dynamic prerequisite type: %s", parts[0])
	}
}

// ValidateDynamicPrerequisite validates parsed dynamic prerequisite parameters
func ValidateDynamicPrerequisite(prereq *domain.DynamicPrerequisite) error {
	if prereq == nil {
		return fmt.Errorf("prerequisite is nil")
	}

	if prereq.Count <= 0 {
		return fmt.Errorf("count must be > 0, got %d", prereq.Count)
	}

	if prereq.Type == "nodes_unlocked_below_tier" {
		if err := ValidateTier(prereq.Tier); err != nil {
			return fmt.Errorf("invalid tier: %w", err)
		}
	}

	return nil
}
