package lootbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// ============================================================================
// Runtime / cache types (built once at startup, read-only thereafter)
// ============================================================================

// FlatPoolEntry is one resolved item entry in a flattened pool.
type FlatPoolEntry struct {
	ItemName    string
	CumulWeight int          // cumulative weight up to and including this entry
	Item        *domain.Item // pre-fetched at startup
}

// FlatPool is a pool whose items have been resolved and sorted by cumulative weight.
type FlatPool struct {
	Entries     []FlatPoolEntry
	TotalWeight int
}

// flatPoolRef is a pool reference with a cumulative weight for weighted selection.
type flatPoolRef struct {
	PoolName    string
	CumulWeight int
}

// FlattenedLootbox is the pre-computed runtime representation of a lootbox.
type FlattenedLootbox struct {
	ItemDropRate    float64
	MoneyMin        int
	MoneyMax        int
	MoneyItem       *domain.Item // pre-fetched consolation money item (may be nil)
	PoolRefs        []flatPoolRef
	TotalPoolWeight int
	Pools           map[string]*FlatPool
}

// ============================================================================
// Cache builder
// ============================================================================

func (s *service) buildCache(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToReadLootFile, err)
	}

	if err := s.schemaValidator.ValidateBytes(data, LootTablesSchemaPath); err != nil {
		return fmt.Errorf("schema validation failed for %s: %w", path, err)
	}

	var config LootTableConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToParseLootFile, err)
	}

	if len(config.Lootboxes) == 0 {
		return fmt.Errorf("no lootboxes defined in configuration")
	}

	// Fetch all items from the database once at startup.
	allItems, err := s.repo.GetAllItems(context.Background())
	if err != nil {
		return fmt.Errorf("failed to fetch all items for lootbox cache: %w", err)
	}

	itemByName := make(map[string]*domain.Item, len(allItems))
	itemsByType := make(map[string][]*domain.Item)
	for i := range allItems {
		item := &allItems[i]
		itemByName[item.InternalName] = item
		for _, ct := range item.ContentType {
			itemsByType[ct] = append(itemsByType[ct], item)
		}
	}

	moneyItem := itemByName[domain.ItemMoney]

	// Orphan tracking — warn about items not referenced by any pool entry.
	s.checkOrphans(allItems, config.Pools)

	// Build flattened pools (filtering by progression unlock status).
	ctx := context.Background()
	flatPools := make(map[string]*FlatPool, len(config.Pools))
	for poolName, poolDef := range config.Pools {
		fp, err := buildFlatPool(ctx, poolDef, itemByName, itemsByType, s.progressionSvc)
		if err != nil {
			return fmt.Errorf("pool %q: %w", poolName, err)
		}
		flatPools[poolName] = fp
	}

	// Build flattened lootboxes.
	cache := make(map[string]*FlattenedLootbox, len(config.Lootboxes))
	for lbName, lbDef := range config.Lootboxes {
		flb, err := buildFlattenedLootbox(lbDef, flatPools, moneyItem)
		if err != nil {
			return fmt.Errorf("lootbox %q: %w", lbName, err)
		}
		cache[lbName] = flb
	}

	s.cache = cache
	return nil
}

// checkOrphans logs a warning for every item that is not referenced by any pool entry.
// Money (domain.ItemMoney) is excluded because it is handled via the consolation path.
func (s *service) checkOrphans(allItems []domain.Item, pools map[string]PoolDef) {
	namedRefs := make(map[string]bool)
	typeRefs := make(map[string]bool)

	for _, poolDef := range pools {
		for _, entry := range poolDef.Items {
			if entry.ItemName != "" {
				namedRefs[entry.ItemName] = true
			}
			if entry.ItemType != "" {
				typeRefs[entry.ItemType] = true
			}
		}
	}

	for _, item := range allItems {
		if item.InternalName == domain.ItemMoney {
			continue
		}

		if namedRefs[item.InternalName] {
			continue
		}

		// Check if any content type of this item is covered by a type-ref entry.
		covered := false
		for _, ct := range item.ContentType {
			if typeRefs[ct] {
				covered = true
				break
			}
		}
		if !covered {
			logger.Warn(LogMsgOrphanedItem, LogFieldItem, item.InternalName)
		}
	}
}

// buildFlatPool resolves a PoolDef into a FlatPool with cumulative weights.
// Items that are locked via progression are excluded, and weights are adjusted accordingly.
func buildFlatPool(ctx context.Context, def PoolDef, itemByName map[string]*domain.Item, itemsByType map[string][]*domain.Item, progressionSvc ProgressionService) (*FlatPool, error) {
	fp := &FlatPool{}

	for _, entry := range def.Items {
		switch {
		case entry.ItemName != "":
			item, ok := itemByName[entry.ItemName]
			if !ok {
				return nil, fmt.Errorf("item %q not found in database", entry.ItemName)
			}

			// Check if item is unlocked via progression
			if !isItemUnlocked(ctx, item.InternalName, progressionSvc) {
				logger.Debug("Excluding locked item from pool", "item", item.InternalName)
				continue
			}

			fp.TotalWeight += entry.Weight
			fp.Entries = append(fp.Entries, FlatPoolEntry{
				ItemName:    entry.ItemName,
				CumulWeight: fp.TotalWeight,
				Item:        item,
			})

		case entry.ItemType != "":
			items, ok := itemsByType[entry.ItemType]
			if !ok || len(items) == 0 {
				return nil, fmt.Errorf("item_type %q has no matching items in database", entry.ItemType)
			}
			for _, item := range items {
				// Check if item is unlocked via progression
				if !isItemUnlocked(ctx, item.InternalName, progressionSvc) {
					logger.Debug("Excluding locked item from pool", "item", item.InternalName)
					continue
				}

				fp.TotalWeight += entry.Weight
				fp.Entries = append(fp.Entries, FlatPoolEntry{
					ItemName:    item.InternalName,
					CumulWeight: fp.TotalWeight,
					Item:        item,
				})
			}
		}
	}

	// Empty pools are valid (all items may be locked via progression)
	return fp, nil
}

// isItemUnlocked checks if an item is unlocked via the progression system.
// Returns true if progressionSvc is nil (no progression checks) or if the item is unlocked.
func isItemUnlocked(ctx context.Context, itemInternalName string, progressionSvc ProgressionService) bool {
	if progressionSvc == nil {
		return true // No progression service = all items unlocked
	}

	// Item progression nodes follow the pattern "item_{internal_name}"
	nodeKey := fmt.Sprintf("item_%s", itemInternalName)
	unlocked, err := progressionSvc.IsNodeUnlocked(ctx, nodeKey, 1)
	if err != nil {
		// On error, default to unlocked to avoid breaking loot system
		logger.Warn("Failed to check item unlock status, defaulting to unlocked", "item", itemInternalName, "error", err)
		return true
	}

	return unlocked
}

// buildFlattenedLootbox resolves a Def into a FlattenedLootbox.
func buildFlattenedLootbox(def Def, flatPools map[string]*FlatPool, moneyItem *domain.Item) (*FlattenedLootbox, error) {
	flb := &FlattenedLootbox{
		ItemDropRate: def.ItemDropRate,
		MoneyMin:     def.FixedMoney.Min,
		MoneyMax:     def.FixedMoney.Max,
		MoneyItem:    moneyItem,
		Pools:        make(map[string]*FlatPool, len(def.Pools)),
	}

	// Ensure MoneyMax >= MoneyMin.
	if flb.MoneyMax < flb.MoneyMin {
		flb.MoneyMax = flb.MoneyMin
	}

	for _, ref := range def.Pools {
		fp, ok := flatPools[ref.PoolName]
		if !ok {
			return nil, fmt.Errorf("pool %q referenced but not defined", ref.PoolName)
		}
		flb.TotalPoolWeight += ref.Weight
		flb.PoolRefs = append(flb.PoolRefs, flatPoolRef{
			PoolName:    ref.PoolName,
			CumulWeight: flb.TotalPoolWeight,
		})
		flb.Pools[ref.PoolName] = fp
	}

	if len(flb.PoolRefs) == 0 {
		return nil, fmt.Errorf("lootbox has no pool references")
	}

	return flb, nil
}
