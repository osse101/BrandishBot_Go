package user

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// SearchRegionDrop represents a single item drop entry in a region's drop table.
type SearchRegionDrop struct {
	ItemName string `json:"item_name"`
	Weight   int    `json:"weight"`
}

// SearchRegion defines a search region with level gating and thematic item drops.
type SearchRegion struct {
	Key                   string             `json:"key"`
	Name                  string             `json:"name"`
	RequiredExplorerLevel int                `json:"required_explorer_level"`
	LootboxChanceModifier float64            `json:"lootbox_chance_modifier"`
	ItemDrops             []SearchRegionDrop `json:"item_drops"`
}

// searchRegionConfig is the top-level JSON structure for search_regions.json.
type searchRegionConfig struct {
	Version string         `json:"version"`
	Regions []SearchRegion `json:"regions"`
}

// loadSearchRegions reads and parses the search regions config file.
func loadSearchRegions(path string) ([]SearchRegion, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read search regions config: %w", err)
	}

	var config searchRegionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse search regions config: %w", err)
	}

	if len(config.Regions) == 0 {
		return nil, fmt.Errorf("search regions config has no regions")
	}

	return config.Regions, nil
}

// resolveRegion picks the best region for the user based on explorer level and an optional item hint.
//
// If itemHint is empty, returns the highest-level region the user qualifies for.
// If itemHint is set (a public_name like "mine"), returns the accessible region
// with the highest weight for an item whose public_name matches the hint.
// Falls back to highest-level region if no match is found.
//
// publicNameIndex maps public_name -> internal_name for item lookup.
func resolveRegion(regions []SearchRegion, explorerLevel int, itemHint string, publicNameIndex map[string]string) *SearchRegion {
	if len(regions) == 0 {
		return nil
	}

	// Filter to accessible regions
	var accessible []SearchRegion
	for _, r := range regions {
		if explorerLevel >= r.RequiredExplorerLevel {
			accessible = append(accessible, r)
		}
	}
	if len(accessible) == 0 {
		return &regions[0] // Fallback to first region (level 0)
	}

	// If no item hint, return highest-level accessible region
	if itemHint == "" {
		best := accessible[0]
		for _, r := range accessible[1:] {
			if r.RequiredExplorerLevel > best.RequiredExplorerLevel {
				best = r
			}
		}
		return &best
	}

	// Resolve public name to internal name
	hint := strings.ToLower(strings.TrimSpace(itemHint))
	internalName, found := publicNameIndex[hint]
	if !found {
		// Hint doesn't match any known public name; fall back to highest-level
		return resolveRegion(regions, explorerLevel, "", publicNameIndex)
	}

	// Find accessible region with highest weight for this item
	var bestRegion *SearchRegion
	bestWeight := 0

	for i, r := range accessible {
		for _, drop := range r.ItemDrops {
			if drop.ItemName == internalName && drop.Weight > bestWeight {
				bestWeight = drop.Weight
				bestRegion = &accessible[i]
			}
		}
	}

	if bestRegion != nil {
		return bestRegion
	}

	// Item exists but isn't in any accessible region's drop table; fall back
	return resolveRegion(regions, explorerLevel, "", publicNameIndex)
}

// rollRegionItemDrop performs a weighted random selection from a region's item drops.
// Returns empty string if the region has no item drops.
func rollRegionItemDrop(drops []SearchRegionDrop) string {
	if len(drops) == 0 {
		return ""
	}

	totalWeight := 0
	for _, d := range drops {
		totalWeight += d.Weight
	}
	if totalWeight == 0 {
		return ""
	}

	roll := utils.SecureRandomIntRange(0, totalWeight-1)
	cumulative := 0
	for _, d := range drops {
		cumulative += d.Weight
		if roll < cumulative {
			return d.ItemName
		}
	}

	return drops[len(drops)-1].ItemName
}

// buildPublicNameIndex creates a map from public_name -> internal_name
// using the service's item cache.
func (s *service) buildPublicNameIndex() map[string]string {
	s.itemCacheMu.RLock()
	defer s.itemCacheMu.RUnlock()

	index := make(map[string]string, len(s.itemCacheByName))
	for internalName, item := range s.itemCacheByName {
		if item.PublicName != "" {
			index[strings.ToLower(item.PublicName)] = internalName
		}
	}
	return index
}
