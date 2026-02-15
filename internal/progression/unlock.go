package progression

import (
	"context"
	"fmt"
)

// IsFeatureUnlocked checks if a feature is available
func (s *service) IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error) {
	// Check cache first (hottest query in the system)
	if unlocked, found := s.unlockCache.Get(featureKey, 1); found {
		return unlocked, nil
	}

	// Cache miss - query database
	unlocked, err := s.repo.IsNodeUnlocked(ctx, featureKey, 1)
	if err != nil {
		return false, err
	}

	// Cache the result
	s.unlockCache.Set(featureKey, 1, unlocked)

	return unlocked, nil
}

// IsItemUnlocked checks if an item is available
func (s *service) IsItemUnlocked(ctx context.Context, itemName string) (bool, error) {
	// Item names are prefixed with "item_"
	nodeKey := fmt.Sprintf("item_%s", itemName)

	// Check cache first
	if unlocked, found := s.unlockCache.Get(nodeKey, 1); found {
		return unlocked, nil
	}

	// Cache miss - query database
	unlocked, err := s.repo.IsNodeUnlocked(ctx, nodeKey, 1)
	if err != nil {
		return false, err
	}

	// Cache the result
	s.unlockCache.Set(nodeKey, 1, unlocked)

	return unlocked, nil
}

func (s *service) IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error) {
	// Check cache first
	if unlocked, found := s.unlockCache.Get(nodeKey, level); found {
		return unlocked, nil
	}

	// Cache miss - query database
	unlocked, err := s.repo.IsNodeUnlocked(ctx, nodeKey, level)
	if err != nil {
		return false, err
	}

	// Cache the result
	s.unlockCache.Set(nodeKey, level, unlocked)

	return unlocked, nil
}

// AreItemsUnlocked checks if multiple items are unlocked in a single batch operation.
// Returns a map of itemName -> unlocked status.
// This is much more efficient than calling IsItemUnlocked N times.
func (s *service) AreItemsUnlocked(ctx context.Context, itemNames []string) (map[string]bool, error) {
	if len(itemNames) == 0 {
		return make(map[string]bool), nil
	}

	result := make(map[string]bool, len(itemNames))
	uncachedKeys := make([]string, 0)
	uncachedNames := make([]string, 0)

	// Check cache first for all items
	for _, itemName := range itemNames {
		nodeKey := fmt.Sprintf("item_%s", itemName)
		if unlocked, found := s.unlockCache.Get(nodeKey, 1); found {
			result[itemName] = unlocked
		} else {
			uncachedKeys = append(uncachedKeys, nodeKey)
			uncachedNames = append(uncachedNames, itemName)
		}
	}

	// If all were cached, return early
	if len(uncachedKeys) == 0 {
		return result, nil
	}

	// Query DB for uncached items and populate cache
	for i, nodeKey := range uncachedKeys {
		unlocked, err := s.repo.IsNodeUnlocked(ctx, nodeKey, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to check unlock status for %s: %w", nodeKey, err)
		}
		s.unlockCache.Set(nodeKey, 1, unlocked)
		result[uncachedNames[i]] = unlocked
	}

	return result, nil
}
