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
				assert.NotEmpty(t, d.ShineLevel)
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

		// Set RNG to high value to prevent lucky upgrade (0.9 > 0.01)
		// This ensures we stay at basic shine level for the chance.
		// Chance 1.0 > 0.9 (Junk) -> Cursed Shine (Mult = 0.4)
		s := svc.(*service)
		s.rnd = func() float64 { return 0.9 }

		drops, err := svc.OpenLootbox(context.Background(), "money_box", 1, "")
		require.NoError(t, err)
		assert.Len(t, drops, 1)

		// Cursed Shine (0.4) * 100 Quantity = 40 Quantity
		// Value should remain 1 (base value)
		assert.Equal(t, "money", drops[0].ItemName)
		assert.Equal(t, 40, drops[0].Quantity)
		assert.Equal(t, 1, drops[0].Value)
	})

	t.Run("Shine Shift Verification", func(t *testing.T) {
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

		// Set RNG to avoid lucky upgrade (0.9 > 0.01)
		s := svc.(*service)
		s.rnd = func() float64 { return 0.9 }

		// Case 1: Common Box
		// Use high quantity to ensure drop
		dropsCommon, err := svc.OpenLootbox(context.Background(), "box", 1000, ShineCommon)
		require.NoError(t, err)
		require.NotEmpty(t, dropsCommon)
		assert.Equal(t, ShineEpic, dropsCommon[0].ShineLevel)

		// Case 2: Uncommon Box
		dropsUncommon, err := svc.OpenLootbox(context.Background(), "box", 1000, ShineUncommon)
		require.NoError(t, err)
		require.NotEmpty(t, dropsUncommon)
		assert.Equal(t, ShineLegendary, dropsUncommon[0].ShineLevel)
	})

	t.Run("Negative Shine Shift Verification", func(t *testing.T) {
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
		// Force RNG to return 0.9 to avoid any "Lucky!" critical upgrades logic interfering
		s.rnd = func() float64 { return 0.9 }

		// Sub-Test 1: Poor Box (-3% Shift)
		// Legendary Threshold becomes 1% - 3% = -2%.
		// Epic Threshold becomes 5% - 3% = 2%.
		// Rare Threshold becomes 15% - 3% = 12%.
		// Item (0.5%) is NOT <= -2%.
		// Item (0.5%) IS <= 2% (Epic).
		// Result: Legendary Item becomes Epic.
		dropsPoor, err := svc.OpenLootbox(context.Background(), "box", 1000, ShinePoor)
		require.NoError(t, err)
		for _, d := range dropsPoor {
			if d.ItemName == "legendary_item" {
				// Verify Poor box downgraded Legendary to Epic
				assert.Equal(t, ShineEpic, d.ShineLevel, "Legendary item in Poor box should downgrade to Epic")
			}
		}

		// Sub-Test 2: Junk Box (-6% Shift)
		// Legendary Threshold: 1% - 6% = -5%
		// Epic Threshold: 5% - 6% = -1%
		// Rare Threshold: 15% - 6% = 9%
		// Item (4.0% Epic) is NOT <= -5%
		// Item (4.0% Epic) is NOT <= -1%
		// Item (4.0% Epic) IS <= 9% (Rare)
		// Result: Epic Item becomes Rare.
		dropsJunk, err := svc.OpenLootbox(context.Background(), "box", 1000, ShineJunk)
		require.NoError(t, err)
		for _, d := range dropsJunk {
			if d.ItemName == "epic_item" {
				// Verify Junk box downgraded Epic to Rare
				assert.Equal(t, ShineRare, d.ShineLevel, "Epic item in Junk box should downgrade to Rare")
			}
		}
	})
}
