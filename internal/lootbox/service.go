package lootbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// LootItem defines an item that can be dropped from a lootbox
type LootItem struct {
	ItemName string  `json:"item_name"`
	Min      int     `json:"min"`
	Max      int     `json:"max"`
	Chance   float64 `json:"chance"`
}

// Shine levels
const (
	ShineCommon    = "COMMON"
	ShineUncommon  = "UNCOMMON"
	ShineRare      = "RARE"
	ShineEpic      = "EPIC"
	ShineLegendary = "LEGENDARY"
)

// DroppedItem represents an item generated from opening a lootbox
type DroppedItem struct {
	ItemID     int
	ItemName   string
	Quantity   int
	Value      int
	ShineLevel string
}

// ItemRepository defines the interface for fetching item data
type ItemRepository interface {
	GetItemByName(ctx context.Context, name string) (*domain.Item, error)
}

// Service defines the lootbox opening interface
type Service interface {
	OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]DroppedItem, error)
}

type service struct {
	repo       ItemRepository
	lootTables map[string][]LootItem
}

// NewService creates a new lootbox service
func NewService(repo ItemRepository, lootTablesPath string) (Service, error) {
	svc := &service{
		repo:       repo,
		lootTables: make(map[string][]LootItem),
	}

	// Load loot tables from JSON file
	if err := svc.loadLootTables(lootTablesPath); err != nil {
		return nil, fmt.Errorf("failed to load loot tables: %w", err)
	}

	return svc, nil
}

func (s *service) loadLootTables(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read loot tables file: %w", err)
	}

	var tables map[string][]LootItem
	if err := json.Unmarshal(data, &tables); err != nil {
		return fmt.Errorf("failed to parse loot tables: %w", err)
	}

	s.lootTables = tables
	return nil
}

// OpenLootbox simulates opening lootboxes and returns the dropped items
func (s *service) OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]DroppedItem, error) {
	log := logger.FromContext(ctx)

	table, ok := s.lootTables[lootboxName]
	if !ok {
		log.Warn("No loot table found for lootbox", "lootbox", lootboxName)
		return nil, nil // Empty result, not an error
	}

	// Generate drops based on probability
	type dropInfo struct {
		Qty    int
		Chance float64
	}
	dropCounts := make(map[string]dropInfo)

	for i := 0; i < quantity; i++ {
		for _, loot := range table {
			if utils.RandomFloat() <= loot.Chance {
				qty := loot.Min
				if loot.Max > loot.Min {
					qty = utils.RandomInt(loot.Min, loot.Max)
				}

				info, exists := dropCounts[loot.ItemName]
				if !exists {
					info = dropInfo{Qty: 0, Chance: loot.Chance}
				} else if loot.Chance < info.Chance {
					// Keep the rarest chance if item appears in multiple entries
					info.Chance = loot.Chance
				}
				info.Qty += qty
				dropCounts[loot.ItemName] = info
			}
		}
	}

	if len(dropCounts) == 0 {
		return nil, nil // No drops
	}

	// Convert to DroppedItem with item IDs
	var drops []DroppedItem
	for itemName, info := range dropCounts {
		item, err := s.repo.GetItemByName(ctx, itemName)
		if err != nil {
			log.Error("Failed to get dropped item", "item", itemName, "error", err)
			continue
		}
		if item == nil {
			log.Warn("Dropped item not found in DB", "item", itemName)
			continue
		}

		drops = append(drops, DroppedItem{
			ItemID:     item.ID,
			ItemName:   item.Name,
			Quantity:   info.Qty,
			Value:      item.BaseValue,
			ShineLevel: calculateShine(info.Chance),
		})
	}

	return drops, nil
}

// calculateShine determines the visual rarity "shine" of a drop based on its chance
func calculateShine(chance float64) string {
	shine := ShineCommon
	if chance <= 0.01 {
		shine = ShineLegendary
	} else if chance <= 0.05 {
		shine = ShineEpic
	} else if chance <= 0.15 {
		shine = ShineRare
	} else if chance <= 0.30 {
		shine = ShineUncommon
	}

	// Critical Shine Upgrade: 1% chance to upgrade the shine level
	// This adds a fun "Lucky!" moment for players
	if utils.RandomFloat() < 0.01 {
		switch shine {
		case ShineCommon:
			shine = ShineUncommon
		case ShineUncommon:
			shine = ShineRare
		case ShineRare:
			shine = ShineEpic
		case ShineEpic:
			shine = ShineLegendary
		}
	}

	return shine
}
