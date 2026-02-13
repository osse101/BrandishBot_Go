package lootbox

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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

// dropInfo accumulates one unique item's drops during a multi-open run.
type dropInfo struct {
	Qty  int
	Item *domain.Item
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

	// Build flattened pools.
	flatPools := make(map[string]*FlatPool, len(config.Pools))
	for poolName, poolDef := range config.Pools {
		fp, err := buildFlatPool(poolDef, itemByName, itemsByType)
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
func buildFlatPool(def PoolDef, itemByName map[string]*domain.Item, itemsByType map[string][]*domain.Item) (*FlatPool, error) {
	fp := &FlatPool{}

	for _, entry := range def.Items {
		switch {
		case entry.ItemName != "":
			item, ok := itemByName[entry.ItemName]
			if !ok {
				return nil, fmt.Errorf("item %q not found in database", entry.ItemName)
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
				fp.TotalWeight += entry.Weight
				fp.Entries = append(fp.Entries, FlatPoolEntry{
					ItemName:    item.InternalName,
					CumulWeight: fp.TotalWeight,
					Item:        item,
				})
			}
		}
	}

	if len(fp.Entries) == 0 || fp.TotalWeight == 0 {
		return nil, fmt.Errorf("pool is empty after expansion")
	}

	return fp, nil
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

// ============================================================================
// 3-stage drop pipeline
// ============================================================================

// processLootTable runs the 3-stage pipeline for `quantity` box opens.
// Returns accumulated item drops and total consolation money.
func (s *service) processLootTable(flat *FlattenedLootbox, quantity int) (map[string]*dropInfo, int) {
	dropCounts := make(map[string]*dropInfo)
	consolationMoney := 0

	for i := 0; i < quantity; i++ {
		// Stage 1 — Gatekeeper roll.
		if s.rnd() >= flat.ItemDropRate {
			// Gatekeeper failed: award consolation money.
			base := s.rnd()*float64(flat.MoneyMax-flat.MoneyMin) + float64(flat.MoneyMin)
			jitter := 1.0 + (s.rnd()-0.5)*(1.0-flat.ItemDropRate)
			amount := int(math.Round(base * jitter))
			if amount < 1 {
				amount = 1
			}
			consolationMoney += amount
			continue
		}

		// Stage 2 — Pool selection (weighted).
		poolName := selectPool(flat, s.rnd())
		pool := flat.Pools[poolName]

		// Stage 3 — Item selection (weighted).
		entry := selectItem(pool, s.rnd())

		if info, ok := dropCounts[entry.ItemName]; ok {
			info.Qty++
		} else {
			dropCounts[entry.ItemName] = &dropInfo{Qty: 1, Item: entry.Item}
		}
	}

	return dropCounts, consolationMoney
}

// selectPool returns the pool name chosen by a weighted roll in [0, TotalPoolWeight).
func selectPool(flat *FlattenedLootbox, rnd float64) string {
	roll := int(rnd * float64(flat.TotalPoolWeight))
	lo, hi := 0, len(flat.PoolRefs)-1
	for lo < hi {
		mid := (lo + hi) / 2
		if flat.PoolRefs[mid].CumulWeight <= roll {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return flat.PoolRefs[lo].PoolName
}

// selectItem returns the pool entry chosen by a weighted roll in [0, TotalWeight).
func selectItem(pool *FlatPool, rnd float64) *FlatPoolEntry {
	roll := int(rnd * float64(pool.TotalWeight))
	lo, hi := 0, len(pool.Entries)-1
	for lo < hi {
		mid := (lo + hi) / 2
		if pool.Entries[mid].CumulWeight <= roll {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return &pool.Entries[lo]
}
