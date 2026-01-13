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

// Shine multipliers (Boosts Gamble Score)
const (
	MultCommon    = 1.0
	MultUncommon  = 1.1
	MultRare      = 1.25
	MultEpic      = 1.5
	MultLegendary = 2.0
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
	GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error)
}

// Service defines the lootbox opening interface
type Service interface {
	OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]DroppedItem, error)
}

type service struct {
	repo       ItemRepository
	lootTables map[string][]LootItem
	// itemCache stores item definitions in memory to avoid DB lookups during lootbox opening.
	// Since item definitions (name, value, etc.) are static config, this is safe.
	itemCache map[string]domain.Item
}

// NewService creates a new lootbox service
func NewService(repo ItemRepository, lootTablesPath string) (Service, error) {
	svc := &service{
		repo:       repo,
		lootTables: make(map[string][]LootItem),
		itemCache:  make(map[string]domain.Item),
	}

	// Load loot tables from JSON file
	if err := svc.loadLootTables(lootTablesPath); err != nil {
		return nil, fmt.Errorf("failed to load loot tables: %w", err)
	}

	// Optimization: Preload all items mentioned in loot tables into cache
	// This avoids N+1 queries or batch queries during OpenLootbox, making it O(0) DB calls.
	if err := svc.preloadItems(context.Background()); err != nil {
		// We log but don't fail, in case DB is momentarily down or items are missing.
		// OpenLootbox handles missing items gracefully (logs/skips).
		// However, failing here might be better to signal config mismatch.
		// Given robust startup is preferred, we'll return error.
		return nil, fmt.Errorf("failed to preload lootbox items: %w", err)
	}

	return svc, nil
}

func (s *service) loadLootTables(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read loot tables file: %w", err)
	}

	// Parse the nested structure with "tables" key
	var config struct {
		Tables map[string][]LootItem `json:"tables"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse loot tables: %w", err)
	}

	s.lootTables = config.Tables
	return nil
}

func (s *service) preloadItems(ctx context.Context) error {
	// Collect all unique item names from all loot tables
	uniqueNamesMap := make(map[string]struct{})
	for _, items := range s.lootTables {
		for _, item := range items {
			uniqueNamesMap[item.ItemName] = struct{}{}
		}
	}

	if len(uniqueNamesMap) == 0 {
		return nil
	}

	names := make([]string, 0, len(uniqueNamesMap))
	for name := range uniqueNamesMap {
		names = append(names, name)
	}

	// Fetch items from repo
	items, err := s.repo.GetItemsByNames(ctx, names)
	if err != nil {
		return err
	}

	// Populate cache
	for _, item := range items {
		s.itemCache[item.InternalName] = item
	}

	return nil
}

// OpenLootbox simulates opening lootboxes and returns the dropped items
func (s *service) OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]DroppedItem, error) {
	if quantity <= 0 {
		return nil, nil
	}

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

	// Optimization: Instead of looping quantity * tableSize times (O(N*T)),
	// we loop over the table and use Geometric distribution to find how many boxes contain the item (O(T)).
	// This reduces RNG calls significantly for large quantities.
	for _, loot := range table {
		if loot.Chance <= 0 {
			continue
		}

		remaining := quantity

		// If chance is >= 1.0, every box drops it (guaranteed)
		if loot.Chance >= 1.0 {
			// All remaining boxes drop this
			count := remaining
			totalQty := 0
			// Calculate quantity
			if loot.Max > loot.Min {
				// Sum of 'count' random integers
				// Optimization: Approximate for large counts if needed, but for now exact loop
				// Since count can be large, we loop here.
				// Wait, if count is large, looping here is O(N) still for the quantity generation.
				// But we saved the "misses".
				for k := 0; k < count; k++ {
					totalQty += utils.SecureRandomIntRange(loot.Min, loot.Max)
				}
			} else {
				totalQty = count * loot.Min
			}

			info, exists := dropCounts[loot.ItemName]
			if !exists {
				info = dropInfo{Qty: 0, Chance: loot.Chance}
			} else if loot.Chance < info.Chance {
				info.Chance = loot.Chance
			}
			info.Qty += totalQty
			dropCounts[loot.ItemName] = info
			continue
		}

		// Standard case: Chance < 1.0
		// Use Geometric distribution to skip failures
		for remaining > 0 {
			// Geometric returns "failures before next success"
			// If skip >= remaining, we failed for all remaining boxes.
			skip := utils.Geometric(loot.Chance)
			if skip >= remaining {
				break
			}

			// Success found at index (current + skip)
			remaining -= (skip + 1) // Consume failures + the success

			// Generate quantity for this single drop
			qty := loot.Min
			if loot.Max > loot.Min {
				qty = utils.SecureRandomIntRange(loot.Min, loot.Max)
			}

			info, exists := dropCounts[loot.ItemName]
			if !exists {
				info = dropInfo{Qty: 0, Chance: loot.Chance}
			} else if loot.Chance < info.Chance {
				info.Chance = loot.Chance
			}
			info.Qty += qty
			dropCounts[loot.ItemName] = info
		}
	}

	if len(dropCounts) == 0 {
		return nil, nil // No drops
	}

	// Convert to DroppedItem with item IDs
	// Optimization: Use cached items to avoid DB lookup
	drops := make([]DroppedItem, 0, len(dropCounts))

	for itemName, info := range dropCounts {
		item, found := s.itemCache[itemName]
		if !found {
			// If item not in cache, log warning.
			// In strict mode we might error, but here we just skip.
			// This implies the loot table references an item not in DB.
			log.Warn("Dropped item not found in cache/DB", "item", itemName)
			continue
		}

		shine, mult := calculateShine(info.Chance)
		boostedValue := int(float64(item.BaseValue) * mult)

		drops = append(drops, DroppedItem{
			ItemID:     item.ID,
			ItemName:   item.InternalName,
			Quantity:   info.Qty,
			Value:      boostedValue,
			ShineLevel: shine,
		})
	}

	return drops, nil
}

// calculateShine determines the visual rarity "shine" and value multiplier of a drop based on its chance
func calculateShine(chance float64) (string, float64) {
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
	if utils.SecureRandomFloat() < 0.01 {
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

	var mult float64
	switch shine {
	case ShineLegendary:
		mult = MultLegendary
	case ShineEpic:
		mult = MultEpic
	case ShineRare:
		mult = MultRare
	case ShineUncommon:
		mult = MultUncommon
	default:
		mult = MultCommon
	}

	return shine, mult
}
