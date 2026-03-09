package search

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// RegionDrop represents a single item drop entry in a region's drop table.
type RegionDrop struct {
	ItemName string `json:"item_name"`
	Weight   int    `json:"weight"`
}

// Region defines a search region with level gating and thematic item drops.
type Region struct {
	Key                   string       `json:"key"`
	Name                  string       `json:"name"`
	RequiredExplorerLevel int          `json:"required_explorer_level"`
	LootboxChanceModifier float64      `json:"lootbox_chance_modifier"`
	ItemDrops             []RegionDrop `json:"item_drops"`
}

// RegionConfig is the top-level JSON structure for search_regions.json.
type RegionConfig struct {
	Version string   `json:"version"`
	Regions []Region `json:"regions"`
}

// LoadSearchRegions reads and parses the search regions config file.
func LoadSearchRegions(path string) ([]Region, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read search regions config: %w", err)
	}

	var config RegionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse search regions config: %w", err)
	}

	if len(config.Regions) == 0 {
		return nil, fmt.Errorf("search regions config has no regions")
	}

	return config.Regions, nil
}

// resolveRegion picks the best region for the user based on explorer level and an optional item hint.
func resolveRegion(regions []Region, explorerLevel int, itemHint string, publicNameIndex map[string]string) *Region {
	if len(regions) == 0 {
		return nil
	}

	var accessible []Region
	for _, r := range regions {
		if explorerLevel >= r.RequiredExplorerLevel {
			accessible = append(accessible, r)
		}
	}
	if len(accessible) == 0 {
		return &regions[0]
	}

	if itemHint == "" {
		best := accessible[0]
		for _, r := range accessible[1:] {
			if r.RequiredExplorerLevel > best.RequiredExplorerLevel {
				best = r
			}
		}
		return &best
	}

	hint := strings.ToLower(strings.TrimSpace(itemHint))
	internalName, found := publicNameIndex[hint]
	if !found {
		return resolveRegion(regions, explorerLevel, "", publicNameIndex)
	}

	var bestRegion *Region
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

	return resolveRegion(regions, explorerLevel, "", publicNameIndex)
}

// rollRegionItemDrop performs a weighted random selection from a region's item drops.
func rollRegionItemDrop(drops []RegionDrop) string {
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
