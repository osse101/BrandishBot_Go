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
	svc, err := NewService(repo, configPath)
	require.NoError(t, err)

	t.Run("Best Case: Success", func(t *testing.T) {
		drops, err := svc.OpenLootbox(context.Background(), "box1", 1)
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
		drops, err := svc.OpenLootbox(context.Background(), "multi_box", 1)
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
		drops, err := svc.OpenLootbox(context.Background(), "invalid_box", 1)
		assert.NoError(t, err)
		assert.Empty(t, drops)
	})

	t.Run("Error Case: Invalid Config File", func(t *testing.T) {
		_, err := NewService(repo, "non_existent_file.json")
		assert.Error(t, err)
	})

	t.Run("Nil/Empty Case: Zero Quantity", func(t *testing.T) {
		drops, err := svc.OpenLootbox(context.Background(), "box1", 0)
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
				_, err := svc.OpenLootbox(context.Background(), "box1", 1)
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
}
