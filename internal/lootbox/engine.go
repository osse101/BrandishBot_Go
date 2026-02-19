package lootbox

import (
	"math"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// ============================================================================
// 3-stage drop pipeline
// ============================================================================

// dropInfo accumulates one unique item's drops during a multi-open run.
type dropInfo struct {
	Qty  int
	Item *domain.Item
}

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

		// If pool is empty (all items locked), treat as gatekeeper failure.
		if len(pool.Entries) == 0 {
			base := s.rnd()*float64(flat.MoneyMax-flat.MoneyMin) + float64(flat.MoneyMin)
			jitter := 1.0 + (s.rnd()-0.5)*(1.0-flat.ItemDropRate)
			amount := int(math.Round(base * jitter))
			if amount < 1 {
				amount = 1
			}
			consolationMoney += amount
			continue
		}

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
