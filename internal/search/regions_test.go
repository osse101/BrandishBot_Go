package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSearchRegions(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("success", func(t *testing.T) {
		validJSON := `{
			"version": "1.0",
			"regions": [
				{
					"key": "forest",
					"name": "Dark Forest",
					"required_explorer_level": 1,
					"lootbox_chance_modifier": 1.0,
					"item_drops": [
						{"item_name": "wood", "weight": 10}
					]
				}
			]
		}`
		filePath := filepath.Join(tempDir, "valid.json")
		err := os.WriteFile(filePath, []byte(validJSON), 0644)
		require.NoError(t, err)

		regions, err := LoadSearchRegions(filePath)
		require.NoError(t, err)
		require.Len(t, regions, 1)
		assert.Equal(t, "forest", regions[0].Key)
		assert.Equal(t, 1, len(regions[0].ItemDrops))
		assert.Equal(t, "wood", regions[0].ItemDrops[0].ItemName)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadSearchRegions(filepath.Join(tempDir, "missing.json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read search regions config")
	})

	t.Run("invalid json", func(t *testing.T) {
		invalidJSON := `{invalid}`
		filePath := filepath.Join(tempDir, "invalid.json")
		err := os.WriteFile(filePath, []byte(invalidJSON), 0644)
		require.NoError(t, err)

		_, err = LoadSearchRegions(filePath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse search regions config")
	})

	t.Run("empty regions", func(t *testing.T) {
		emptyJSON := `{
			"version": "1.0",
			"regions": []
		}`
		filePath := filepath.Join(tempDir, "empty.json")
		err := os.WriteFile(filePath, []byte(emptyJSON), 0644)
		require.NoError(t, err)

		_, err = LoadSearchRegions(filePath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "search regions config has no regions")
	})
}

func TestResolveRegion(t *testing.T) {
	regions := []Region{
		{
			Key: "starter", RequiredExplorerLevel: 1, ItemDrops: []RegionDrop{
				{ItemName: "wood", Weight: 10},
			},
		},
		{
			Key: "mid", RequiredExplorerLevel: 5, ItemDrops: []RegionDrop{
				{ItemName: "iron", Weight: 10},
				{ItemName: "gold", Weight: 2},
			},
		},
		{
			Key: "endgame", RequiredExplorerLevel: 10, ItemDrops: []RegionDrop{
				{ItemName: "diamond", Weight: 5},
				{ItemName: "gold", Weight: 10},
			},
		},
	}

	publicNameIndex := map[string]string{
		"wood log": "wood",
		"iron ore": "iron",
		"gold ore": "gold",
		"diamond":  "diamond",
	}

	tests := []struct {
		name          string
		explorerLevel int
		itemHint      string
		regions       []Region
		expectedKey   string
		expectedNil   bool
	}{
		{
			name:          "empty regions",
			explorerLevel: 1,
			itemHint:      "",
			regions:       []Region{},
			expectedNil:   true,
		},
		{
			name:          "level too low for everything (fallback to first)",
			explorerLevel: 0,
			itemHint:      "",
			regions:       regions,
			expectedKey:   "starter",
		},
		{
			name:          "level 1 without hint picks highest accessible",
			explorerLevel: 1,
			itemHint:      "",
			regions:       regions,
			expectedKey:   "starter",
		},
		{
			name:          "level 6 without hint picks highest accessible",
			explorerLevel: 6,
			itemHint:      "",
			regions:       regions,
			expectedKey:   "mid",
		},
		{
			name:          "level 20 without hint picks highest accessible",
			explorerLevel: 20,
			itemHint:      "",
			regions:       regions,
			expectedKey:   "endgame",
		},
		{
			name:          "hint found in accessible region",
			explorerLevel: 6,
			itemHint:      "iron ore",
			regions:       regions,
			expectedKey:   "mid",
		},
		{
			name:          "hint found in multiple regions, picks highest weight",
			explorerLevel: 15,
			itemHint:      "gold ore",
			regions:       regions,
			expectedKey:   "endgame", // weight 10 vs 2 in mid
		},
		{
			name:          "hint not in public name index (fallback to highest accessible)",
			explorerLevel: 6,
			itemHint:      "unknown item",
			regions:       regions,
			expectedKey:   "mid",
		},
		{
			name:          "hint not in any accessible drop table (fallback to highest accessible)",
			explorerLevel: 6,
			itemHint:      "diamond", // Needs level 10
			regions:       regions,
			expectedKey:   "mid",
		},
		{
			name:          "hint padding and case insensitive",
			explorerLevel: 15,
			itemHint:      "   gOlD oRe  ",
			regions:       regions,
			expectedKey:   "endgame",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := resolveRegion(tt.regions, tt.explorerLevel, tt.itemHint, publicNameIndex)
			if tt.expectedNil {
				assert.Nil(t, res)
			} else {
				require.NotNil(t, res)
				assert.Equal(t, tt.expectedKey, res.Key)
			}
		})
	}
}

func TestRollRegionItemDrop(t *testing.T) {
	t.Run("empty drops", func(t *testing.T) {
		assert.Equal(t, "", rollRegionItemDrop([]RegionDrop{}))
	})

	t.Run("zero weight drops", func(t *testing.T) {
		assert.Equal(t, "", rollRegionItemDrop([]RegionDrop{
			{ItemName: "wood", Weight: 0},
		}))
	})

	t.Run("single drop", func(t *testing.T) {
		assert.Equal(t, "wood", rollRegionItemDrop([]RegionDrop{
			{ItemName: "wood", Weight: 10},
		}))
	})

	t.Run("multiple drops statistical distribution", func(t *testing.T) {
		drops := []RegionDrop{
			{ItemName: "common", Weight: 70}, // 70%
			{ItemName: "uncommon", Weight: 25}, // 25%
			{ItemName: "rare", Weight: 5},    // 5%
		}

		counts := map[string]int{
			"common":   0,
			"uncommon": 0,
			"rare":     0,
		}

		// Run enough times to get a statistically significant distribution
		// but keep it fast enough for unit tests
		iterations := 10000
		for i := 0; i < iterations; i++ {
			item := rollRegionItemDrop(drops)
			counts[item]++
		}

		// Allow some margin of error for randomness (e.g., +/- 3% absolute)
		assert.InDelta(t, 7000, counts["common"], 300)
		assert.InDelta(t, 2500, counts["uncommon"], 300)
		assert.InDelta(t, 500, counts["rare"], 150)
	})
}
