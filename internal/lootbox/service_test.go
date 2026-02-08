package lootbox

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Thread-safe mock repo
type mockItemRepo struct {
	sync.RWMutex
	items map[string]*domain.Item
}

func (m *mockItemRepo) GetItemByName(ctx context.Context, name string) (*domain.Item, error) {
	m.RLock()
	defer m.RUnlock()
	return m.items[name], nil
}

func (m *mockItemRepo) GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error) {
	m.RLock()
	defer m.RUnlock()
	var items []domain.Item
	for _, name := range names {
		if item, ok := m.items[name]; ok {
			items = append(items, *item)
		}
	}
	return items, nil
}

type mockProgression struct {
	unlocked bool
}

func (m *mockProgression) IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error) {
	return m.unlocked, nil
}

// Helper to create temp config file
func createTempConfig(t *testing.T, tables map[string][]LootItem) string {
	config := struct {
		Version string                `json:"version"`
		Tables  map[string][]LootItem `json:"tables"`
	}{
		Version: "1.0",
		Tables:  tables,
	}

	file, err := os.CreateTemp("", "loot_*.json")
	require.NoError(t, err)

	encoder := json.NewEncoder(file)
	err = encoder.Encode(config)
	require.NoError(t, err)
	file.Close()

	t.Cleanup(func() {
		os.Remove(file.Name())
	})

	return file.Name()
}

func TestOpenLootbox(t *testing.T) {
	// Common Setup
	repo := &mockItemRepo{
		items: map[string]*domain.Item{
			"common_sword": {ID: 1, InternalName: "common_sword", BaseValue: 10},
			"rare_sword":   {ID: 2, InternalName: "rare_sword", BaseValue: 100},
			"epic_sword":   {ID: 3, InternalName: "epic_sword", BaseValue: 1000},
		},
	}

	lootTable := map[string][]LootItem{
		"box1": {
			{ItemName: "common_sword", Min: 1, Max: 1, Chance: 1.0},
			{ItemName: "rare_sword", Min: 1, Max: 1, Chance: 0.5},
		},
		"empty_box": {},
		"multi_box": {
			{ItemName: "common_sword", Min: 2, Max: 5, Chance: 1.0},
		},
	}

	configPath := createTempConfig(t, lootTable)
	svc, err := NewService(repo, &mockProgression{unlocked: true}, configPath)
	require.NoError(t, err)

	t.Run("Best Case: Success", func(t *testing.T) {
		drops, err := svc.OpenLootbox(context.Background(), "box1", 1, "")
		require.NoError(t, err)
		assert.NotEmpty(t, drops)

		// Should always have common_sword (chance 1.0)
		foundCommon := false
		for _, d := range drops {
			if d.ItemName == "common_sword" {
				foundCommon = true
				assert.NotZero(t, d.Value)
				assert.NotEmpty(t, d.QualityLevel)
			}
		}
		assert.True(t, foundCommon, "Should have dropped common_sword")
	})

	t.Run("Boundary Case: Min/Max Quantity", func(t *testing.T) {
		// multi_box drops 2-5 common_swords
		drops, err := svc.OpenLootbox(context.Background(), "multi_box", 1, "")
		require.NoError(t, err)

		count := 0
		for _, d := range drops {
			if d.ItemName == "common_sword" {
				count += d.Quantity
			}
		}
		assert.GreaterOrEqual(t, count, 2)
		assert.LessOrEqual(t, count, 5)
	})

	t.Run("Error Case: Box Not Found", func(t *testing.T) {
		drops, err := svc.OpenLootbox(context.Background(), "invalid_box", 1, "")
		assert.NoError(t, err)
		assert.Empty(t, drops)
	})

	t.Run("Error Case: Invalid Config File", func(t *testing.T) {
		repo := &mockItemRepo{}
		_, err := NewService(repo, &mockProgression{}, "non_existent_path")
		require.Error(t, err)
	})

	t.Run("Nil/Empty Case: Zero Quantity", func(t *testing.T) {
		drops, err := svc.OpenLootbox(context.Background(), "box1", 0, "")
		require.NoError(t, err)
		assert.Empty(t, drops, "Should return empty drops for 0 quantity")
	})

	t.Run("Concurrent Case: Parallel Open", func(t *testing.T) {
		var wg sync.WaitGroup
		errChan := make(chan error, 20)

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := svc.OpenLootbox(context.Background(), "box1", 1, "")
				if err != nil {
					errChan <- err
				}
			}()
		}
		wg.Wait()
		close(errChan)

		for err := range errChan {
			assert.NoError(t, err)
		}
	})

	t.Run("Special Case: Money Scaling", func(t *testing.T) {
		repo := &mockItemRepo{
			items: map[string]*domain.Item{
				"money": {ID: 10, InternalName: "money", BaseValue: 1},
			},
		}
		lootTable := map[string][]LootItem{
			"money_box": {
				{ItemName: "money", Min: 100, Max: 100, Chance: 1.0},
			},
		}
		configPath := createTempConfig(t, lootTable)
		svc, err := NewService(repo, &mockProgression{unlocked: true}, configPath)
		require.NoError(t, err)

		// Set RNG to high value: 0.98 for Cursed quality, 0.9 for no lucky upgrade
		s := svc.(*service)
		var rolls = []float64{0.98, 0.9} // 1st: Cursed, 2nd: No upgrade
		var rollIdx int
		s.rnd = func() float64 {
			val := rolls[rollIdx]
			rollIdx = (rollIdx + 1) % len(rolls)
			return val
		}

		drops, err := svc.OpenLootbox(context.Background(), "money_box", 1, "")
		require.NoError(t, err)
		assert.Len(t, drops, 1)

		// Cursed Quality (0.4) * 100 Quantity = 40 Quantity
		assert.Equal(t, "money", drops[0].ItemName)
		assert.Equal(t, 40, drops[0].Quantity)
		assert.Equal(t, 1, drops[0].Value)
	})

	t.Run("Quality Shift Verification", func(t *testing.T) {
		repo := &mockItemRepo{
			items: map[string]*domain.Item{
				"test_item": {ID: 1, InternalName: "test_item", BaseValue: 100},
			},
		}
		// Create a loot table with an item that has exactly 4% chance (0.04)
		// With Common Box (bonus 0), 0.04 > 0.01 (Legendary), 0.04 <= 0.05 (Epic). Should be Epic.
		// With Uncommon Box (bonus 0.03), Legendary Thresh = 0.01 + 0.03 = 0.04.
		// 0.04 <= 0.04. Should be Legendary.
		lootTable := map[string][]LootItem{
			"box": {
				{ItemName: "test_item", Min: 1, Max: 1, Chance: 0.04},
			},
		}

		configPath := createTempConfig(t, lootTable)
		svc, err := NewService(repo, &mockProgression{unlocked: false}, configPath)
		require.NoError(t, err)

		// Set RNG: 0.04 for Legendary/Epic quality, 0.9 for no lucky upgrade
		s := svc.(*service)

		// Case 1: Common Box
		// Use high quantity to ensure drop (though not strictly needed with 0.04 < 0.70)
		rollIdx := 0
		s.rnd = func() float64 {
			rolls := []float64{0.04, 0.9} // 1: Epic (<=0.05), 2: No upgrade
			val := rolls[rollIdx%2]
			rollIdx++
			return val
		}

		dropsCommon, err := svc.OpenLootbox(context.Background(), "box", 1000, domain.QualityCommon)
		require.NoError(t, err)
		require.NotEmpty(t, dropsCommon)
		assert.Equal(t, domain.QualityEpic, dropsCommon[0].QualityLevel)

		// Case 2: Uncommon Box
		// Reset roll index
		rollIdx = 0
		// With Uncommon Box (bonus 0.03), Legendary Thresh = 0.01 + 0.03 = 0.04.
		// Roll 0.04 <= 0.04. Should be Legendary.
		dropsUncommon, err := svc.OpenLootbox(context.Background(), "box", 1000, domain.QualityUncommon)
		require.NoError(t, err)
		require.NotEmpty(t, dropsUncommon)
		assert.Equal(t, domain.QualityLegendary, dropsUncommon[0].QualityLevel)
	})

	t.Run("Negative Quality Shift Verification", func(t *testing.T) {
		repo := &mockItemRepo{
			items: map[string]*domain.Item{
				"legendary_item": {ID: 1, InternalName: "legendary_item", BaseValue: 1000},
				"epic_item":      {ID: 2, InternalName: "epic_item", BaseValue: 500},
			},
		}

		lootTable := map[string][]LootItem{
			"box": {
				{ItemName: "legendary_item", Min: 1, Max: 1, Chance: 0.005}, // 0.5% - Normally Legendary (<= 1%)
				{ItemName: "epic_item", Min: 1, Max: 1, Chance: 0.04},       // 4.0% - Normally Epic (<= 5%)
			},
		}

		configPath := createTempConfig(t, lootTable)
		// No progression needed for this test
		svc, err := NewService(repo, nil, configPath)
		require.NoError(t, err)

		s := svc.(*service)

		// Sub-Test 1: Poor Box (-3% Shift)
		// We want a roll that is normally Legendary (<= 0.01) but becomes Epic with -0.03 bonus.
		// Roll 0.005: 0.005 > -0.02 (Legendary thresh) but 0.005 <= 0.02 (Epic thresh).
		s.rnd = func() float64 { return 0.005 }
		dropsPoor, err := svc.OpenLootbox(context.Background(), "box", 1000, domain.QualityPoor)
		require.NoError(t, err)
		require.NotEmpty(t, dropsPoor)
		for _, d := range dropsPoor {
			if d.ItemName == "legendary_item" {
				assert.Equal(t, domain.QualityEpic, d.QualityLevel, "Legendary item in Poor box should downgrade to Epic")
			}
		}

		// Sub-Test 2: Junk Box (-6% Shift)
		// We want a roll that is normally Epic (<= 0.05) but becomes Rare with -0.06 bonus.
		// Roll 0.04: 0.04 > -0.01 (Epic thresh) but 0.04 <= 0.09 (Rare thresh).
		s.rnd = func() float64 { return 0.04 }
		dropsJunk, err := svc.OpenLootbox(context.Background(), "box", 1000, domain.QualityJunk)
		require.NoError(t, err)
		require.NotEmpty(t, dropsJunk)
		for _, d := range dropsJunk {
			if d.ItemName == "epic_item" {
				assert.Equal(t, domain.QualityRare, d.QualityLevel, "Epic item in Junk box should downgrade to Rare")
			}
		}
	})
}
