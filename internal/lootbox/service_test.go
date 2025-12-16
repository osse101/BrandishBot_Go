package lootbox

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

type mockItemRepo struct {
	items map[string]*domain.Item
}

func (m *mockItemRepo) GetItemByName(ctx context.Context, name string) (*domain.Item, error) {
	return m.items[name], nil
}

func TestOpenLootbox(t *testing.T) {
	// Setup loot table file
	lootTable := map[string][]LootItem{
		"box1": {
			{ItemName: "common_sword", Min: 1, Max: 1, Chance: 1.0},
			{ItemName: "rare_sword", Min: 1, Max: 1, Chance: 0.1},
		},
	}
	file, _ := os.CreateTemp("", "loot.json")
	defer os.Remove(file.Name())

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(lootTable); err != nil {
		t.Fatalf("Failed to encode loot table: %v", err)
	}
	file.Close()

	// Setup mock repo
	repo := &mockItemRepo{
		items: map[string]*domain.Item{
			"common_sword": {ID: 1, Name: "common_sword", BaseValue: 10},
			"rare_sword":   {ID: 2, Name: "rare_sword", BaseValue: 100},
		},
	}

	// Create service
	svc, err := NewService(repo, file.Name())
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test
	// We run 1 iteration to make it simple, but since chance is 1.0 for common, we should get at least that.
	drops, err := svc.OpenLootbox(context.Background(), "box1", 1)
	if err != nil {
		t.Fatalf("OpenLootbox failed: %v", err)
	}

	if len(drops) == 0 {
		t.Errorf("Expected drops, got none")
	}

	foundCommon := false
	for _, d := range drops {
		if d.ItemName == "common_sword" {
			foundCommon = true
			if d.ShineLevel != ShineCommon && d.ShineLevel != ShineUncommon { // Allow for Crit Upgrade
				t.Errorf("Expected Common or Uncommon shine for common item, got %s", d.ShineLevel)
			}
		}
		if d.ItemName == "rare_sword" {
			if d.ShineLevel != ShineRare && d.ShineLevel != ShineEpic { // Allow for Crit Upgrade
				t.Errorf("Expected Rare or Epic shine for rare item, got %s", d.ShineLevel)
			}
		}
		// Check that other fields are populated (pre-existing behavior)
		if d.ItemID == 0 {
			t.Errorf("ItemID not populated")
		}
		if d.Value == 0 {
			t.Errorf("Value not populated")
		}
		if d.ShineLevel == "" {
			t.Errorf("ShineLevel not populated")
		}
	}
	if !foundCommon {
		t.Errorf("Expected common_sword")
	}
}
