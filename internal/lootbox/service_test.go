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

// ============================================================================
// Test doubles
// ============================================================================

// mockItemRepo is a thread-safe in-memory item repository.
type mockItemRepo struct {
	sync.RWMutex
	items map[string]*domain.Item
}

func (m *mockItemRepo) GetItemByName(_ context.Context, name string) (*domain.Item, error) {
	m.RLock()
	defer m.RUnlock()
	return m.items[name], nil
}

func (m *mockItemRepo) GetItemsByNames(_ context.Context, names []string) ([]domain.Item, error) {
	m.RLock()
	defer m.RUnlock()
	var result []domain.Item
	for _, name := range names {
		if item, ok := m.items[name]; ok {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *mockItemRepo) GetAllItems(_ context.Context) ([]domain.Item, error) {
	m.RLock()
	defer m.RUnlock()
	result := make([]domain.Item, 0, len(m.items))
	for _, item := range m.items {
		result = append(result, *item)
	}
	return result, nil
}

type mockProgression struct{ unlocked bool }

func (m *mockProgression) IsNodeUnlocked(_ context.Context, _ string, _ int) (bool, error) {
	return m.unlocked, nil
}

// ============================================================================
// Helpers
// ============================================================================

// createTempConfigV2 writes a LootTableConfig to a temp file and returns its path.
func createTempConfigV2(t *testing.T, pools map[string]PoolDef, lootboxes map[string]Def) string {
	t.Helper()
	config := LootTableConfig{
		Version:   ConfigVersion2,
		Pools:     pools,
		Lootboxes: lootboxes,
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	file, err := os.CreateTemp("", "loot_*.json")
	require.NoError(t, err)

	_, err = file.Write(data)
	require.NoError(t, err)
	file.Close()

	t.Cleanup(func() { os.Remove(file.Name()) })
	return file.Name()
}

// moneyItem returns a basic money item for use in tests.
func moneyItem() *domain.Item {
	return &domain.Item{ID: 1, InternalName: domain.ItemMoney, BaseValue: 1, Types: []string{"currency"}}
}

// swordItem returns a basic sword item for use in tests.
func swordItem(id int, name string, value int) *domain.Item {
	return &domain.Item{ID: id, InternalName: name, BaseValue: value}
}

// buildSimpleService creates a service with one lootbox and one pool.
// rndVals is an optional sequence of deterministic rnd values.
func buildSimpleService(t *testing.T, repo *mockItemRepo, itemDropRate float64, moneyMin, moneyMax int, poolItems []PoolItemDef, rndVals []float64) (*service, error) {
	t.Helper()
	pools := map[string]PoolDef{
		"pool_a": {Items: poolItems},
	}
	lootboxes := map[string]Def{
		"box": {
			ItemDropRate: itemDropRate,
			FixedMoney:   MoneyRange{Min: moneyMin, Max: moneyMax},
			Pools:        []PoolRef{{PoolName: "pool_a", Weight: 1}},
		},
	}
	path := createTempConfigV2(t, pools, lootboxes)
	svc, err := NewService(repo, &mockProgression{unlocked: true}, nil, path)
	if err != nil {
		return nil, err
	}

	s := svc.(*service)
	if len(rndVals) > 0 {
		idx := 0
		s.rnd = func() float64 {
			v := rndVals[idx%len(rndVals)]
			idx++
			return v
		}
	}
	return s, nil
}

// ============================================================================
// Gatekeeper tests
// ============================================================================

func TestGatekeeperPass_ItemDropped(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"sword":          swordItem(2, "sword", 10),
	}}

	// ItemDropRate=1.0 → gatekeeper always passes → item always drops.
	// rnd sequence: [gatekeeper=0.5, pool=0.5, item=0.5, quality=0.5, upgradeCheck=0.9]
	s, err := buildSimpleService(t, repo, 1.0, 0, 10,
		[]PoolItemDef{{ItemName: "sword", Weight: 1}},
		[]float64{0.5, 0.5, 0.5, 0.5, 0.9},
	)
	require.NoError(t, err)

	drops, err := s.OpenLootbox(context.Background(), "box", 1, domain.QualityCommon)
	require.NoError(t, err)
	require.Len(t, drops, 1)
	assert.Equal(t, "sword", drops[0].ItemName)
}

func TestGatekeeperFail_MoneyOnly(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"sword":          swordItem(2, "sword", 10),
	}}

	// ItemDropRate=0.0 → gatekeeper always fails (0.5 >= 0.0) → consolation money always.
	// rnd per open: [gatekeeper=0.5, base=0.5, jitter=0.5] → base=50*(1)+0=0... wait.
	// moneyMin=100, moneyMax=100 → base=0.5*(100-100)+100=100, jitter=1+(0.5-0.5)*1=1, amount=100
	s, err := buildSimpleService(t, repo, 0.0, 100, 100,
		[]PoolItemDef{{ItemName: "sword", Weight: 1}},
		[]float64{0.5},
	)
	require.NoError(t, err)

	drops, err := s.OpenLootbox(context.Background(), "box", 1, domain.QualityCommon)
	require.NoError(t, err)
	require.Len(t, drops, 1)
	assert.Equal(t, domain.ItemMoney, drops[0].ItemName)
	assert.Equal(t, domain.QualityCommon, drops[0].QualityLevel)
	assert.Equal(t, 100, drops[0].Quantity)
}

// ============================================================================
// Money jitter tests
// ============================================================================

func TestMoneyJitter_Proportional(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
	}}

	// ItemDropRate=0.5 → gatekeeper fails when rnd >= 0.5.
	// We want to test jitter: rnd=[0.9(gate fail), 0.8(base), 0.8(jitter)]
	// base = 0.8*(100-50)+50 ≈ 89.999... (0.8 is not exact in float64)
	// jitter = 1+(0.8-0.5)*(1-0.5) ≈ 1.1499...
	// amount = round(89.999... * 1.1499...) = round(103.499...) = 103
	rolls := []float64{0.9, 0.8, 0.8}
	s, err := buildSimpleService(t, repo, 0.5, 50, 100,
		[]PoolItemDef{{ItemName: domain.ItemMoney, Weight: 1}},
		nil,
	)
	require.NoError(t, err)
	idx := 0
	s.rnd = func() float64 { v := rolls[idx%len(rolls)]; idx++; return v }

	drops, err := s.OpenLootbox(context.Background(), "box", 1, domain.QualityCommon)
	require.NoError(t, err)
	require.Len(t, drops, 1)
	assert.Equal(t, domain.ItemMoney, drops[0].ItemName)
	assert.Equal(t, 103, drops[0].Quantity)
}

func TestMoneyJitter_FloorsAtOne(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
	}}

	// moneyMin=moneyMax=0 → base always 0 → amount rounds to 0 → clamped to 1.
	// gatekeeper: any value >= 0.0 when ItemDropRate=0.0.
	s, err := buildSimpleService(t, repo, 0.0, 0, 0,
		[]PoolItemDef{{ItemName: domain.ItemMoney, Weight: 1}},
		[]float64{0.5},
	)
	require.NoError(t, err)

	drops, err := s.OpenLootbox(context.Background(), "box", 1, domain.QualityCommon)
	require.NoError(t, err)
	require.Len(t, drops, 1)
	assert.Equal(t, domain.ItemMoney, drops[0].ItemName)
	assert.GreaterOrEqual(t, drops[0].Quantity, 1)
}

// ============================================================================
// Aggregation tests
// ============================================================================

func TestMultiQuantity_Aggregated(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"sword":          swordItem(2, "sword", 10),
	}}

	// ItemDropRate=1.0 → always item. 5 opens → 5 sword drops aggregated.
	// rnd sequence cycling: [gate=0.5, pool=0.5, item=0.5, quality=0.5, upgrade=0.9]
	s, err := buildSimpleService(t, repo, 1.0, 0, 10,
		[]PoolItemDef{{ItemName: "sword", Weight: 1}},
		[]float64{0.5, 0.5, 0.5, 0.5, 0.9},
	)
	require.NoError(t, err)

	drops, err := s.OpenLootbox(context.Background(), "box", 5, domain.QualityCommon)
	require.NoError(t, err)
	require.Len(t, drops, 1) // one unique item, aggregated
	assert.Equal(t, "sword", drops[0].ItemName)
	assert.Equal(t, 5, drops[0].Quantity)
}

func TestSameItem_QuantitySummed(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"sword":          swordItem(2, "sword", 10),
	}}

	// Two pool entries for the same item → both select "sword" → quantities sum.
	// With rnd=0.5 and equal weights (1,1), roll=0 for pool, item selection varies.
	// Actually, same item regardless of which "slot" selects, so still aggregated.
	pools := map[string]PoolDef{
		"pool_a": {Items: []PoolItemDef{
			{ItemName: "sword", Weight: 1},
			{ItemName: "sword", Weight: 1},
		}},
	}
	lootboxes := map[string]Def{
		"box": {
			ItemDropRate: 1.0,
			FixedMoney:   MoneyRange{Min: 0, Max: 0},
			Pools:        []PoolRef{{PoolName: "pool_a", Weight: 1}},
		},
	}
	path := createTempConfigV2(t, pools, lootboxes)
	svc, err := NewService(repo, &mockProgression{unlocked: true}, nil, path)
	require.NoError(t, err)

	drops, err := svc.OpenLootbox(context.Background(), "box", 3, domain.QualityCommon)
	require.NoError(t, err)
	require.Len(t, drops, 1)
	assert.Equal(t, "sword", drops[0].ItemName)
	assert.Equal(t, 3, drops[0].Quantity)
}

// ============================================================================
// Weighted selection tests
// ============================================================================

func TestPoolSelection_WeightedRoll(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"sword_a":        swordItem(2, "sword_a", 10),
		"sword_b":        swordItem(3, "sword_b", 20),
	}}

	// pool_a weight=30, pool_b weight=70. TotalPoolWeight=100.
	// With rnd=0.1 for pool → roll=10 → cumul[0]=30 > 10 → pool_a (sword_a).
	// With rnd=0.5 for pool → roll=50 → cumul[0]=30 <= 50, cumul[1]=100 > 50 → pool_b (sword_b).
	pools := map[string]PoolDef{
		"pool_a": {Items: []PoolItemDef{{ItemName: "sword_a", Weight: 1}}},
		"pool_b": {Items: []PoolItemDef{{ItemName: "sword_b", Weight: 1}}},
	}
	lootboxes := map[string]Def{
		"box": {
			ItemDropRate: 1.0,
			FixedMoney:   MoneyRange{Min: 0, Max: 0},
			Pools: []PoolRef{
				{PoolName: "pool_a", Weight: 30},
				{PoolName: "pool_b", Weight: 70},
			},
		},
	}
	path := createTempConfigV2(t, pools, lootboxes)
	svc, err := NewService(repo, &mockProgression{unlocked: true}, nil, path)
	require.NoError(t, err)
	s := svc.(*service)

	// Roll low → pool_a (sword_a).
	// rnd sequence: [gate=0.0(pass), pool=0.1, item=0.0, quality=0.5, upgrade=0.9]
	rolls := []float64{0.0, 0.1, 0.0, 0.5, 0.9}
	idx := 0
	s.rnd = func() float64 { v := rolls[idx]; idx++; return v }

	drops, err := s.OpenLootbox(context.Background(), "box", 1, domain.QualityCommon)
	require.NoError(t, err)
	require.Len(t, drops, 1)
	assert.Equal(t, "sword_a", drops[0].ItemName)

	// Roll high → pool_b (sword_b).
	rolls2 := []float64{0.0, 0.5, 0.0, 0.5, 0.9}
	idx2 := 0
	s.rnd = func() float64 { v := rolls2[idx2]; idx2++; return v }

	drops2, err := s.OpenLootbox(context.Background(), "box", 1, domain.QualityCommon)
	require.NoError(t, err)
	require.Len(t, drops2, 1)
	assert.Equal(t, "sword_b", drops2[0].ItemName)
}

func TestItemSelection_WeightedRoll(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"item_common":    swordItem(2, "item_common", 10),
		"item_rare":      swordItem(3, "item_rare", 100),
	}}

	// item_common weight=70, item_rare weight=30. TotalWeight=100.
	// rnd=0.1 for item → roll=10 → item_common (cumul=70 > 10).
	// rnd=0.8 for item → roll=80 → item_rare  (cumul=70 <= 80, cumul=100 > 80).
	pools := map[string]PoolDef{
		"pool_a": {Items: []PoolItemDef{
			{ItemName: "item_common", Weight: 70},
			{ItemName: "item_rare", Weight: 30},
		}},
	}
	lootboxes := map[string]Def{
		"box": {
			ItemDropRate: 1.0,
			FixedMoney:   MoneyRange{Min: 0, Max: 0},
			Pools:        []PoolRef{{PoolName: "pool_a", Weight: 1}},
		},
	}
	path := createTempConfigV2(t, pools, lootboxes)
	svc, err := NewService(repo, &mockProgression{unlocked: true}, nil, path)
	require.NoError(t, err)
	s := svc.(*service)

	// Select common.
	rollsCommon := []float64{0.0, 0.0, 0.1, 0.5, 0.9}
	idx := 0
	s.rnd = func() float64 { v := rollsCommon[idx]; idx++; return v }
	drops, err := s.OpenLootbox(context.Background(), "box", 1, domain.QualityCommon)
	require.NoError(t, err)
	require.Len(t, drops, 1)
	assert.Equal(t, "item_common", drops[0].ItemName)

	// Select rare.
	rollsRare := []float64{0.0, 0.0, 0.8, 0.5, 0.9}
	idx2 := 0
	s.rnd = func() float64 { v := rollsRare[idx2]; idx2++; return v }
	drops2, err := s.OpenLootbox(context.Background(), "box", 1, domain.QualityCommon)
	require.NoError(t, err)
	require.Len(t, drops2, 1)
	assert.Equal(t, "item_rare", drops2[0].ItemName)
}

// ============================================================================
// Type expansion tests
// ============================================================================

func TestTypeExpansion_Explosive(t *testing.T) {
	// 4 explosive items — each gets weight 25 individually → TotalWeight=100.
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"explosive_mine": {ID: 10, InternalName: "explosive_mine", ContentType: []string{"explosive"}},
		"explosive_trap": {ID: 11, InternalName: "explosive_trap", ContentType: []string{"explosive"}},
		"explosive_tnt":  {ID: 12, InternalName: "explosive_tnt", ContentType: []string{"explosive"}},
		"item_grenade":   {ID: 13, InternalName: "item_grenade", ContentType: []string{"explosive"}},
	}}

	pools := map[string]PoolDef{
		"pool_a": {Items: []PoolItemDef{
			{ItemType: "explosive", Weight: 25},
		}},
	}
	lootboxes := map[string]Def{
		"box": {
			ItemDropRate: 1.0,
			FixedMoney:   MoneyRange{Min: 0, Max: 0},
			Pools:        []PoolRef{{PoolName: "pool_a", Weight: 1}},
		},
	}
	path := createTempConfigV2(t, pools, lootboxes)
	svc, err := NewService(repo, &mockProgression{unlocked: true}, nil, path)
	require.NoError(t, err)

	// Verify cache: pool_a should have 4 entries with TotalWeight=100.
	s := svc.(*service)
	flat := s.cache["box"]
	require.NotNil(t, flat)
	pool := flat.Pools["pool_a"]
	require.NotNil(t, pool)
	assert.Equal(t, 4, len(pool.Entries))
	assert.Equal(t, 100, pool.TotalWeight)
}

func TestTypeExpansion_Unknown_Error(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
	}}
	// "unknown_type" has no items in the mock — should cause NewService to fail.
	pools := map[string]PoolDef{
		"pool_a": {Items: []PoolItemDef{
			{ItemType: "unknown_type", Weight: 1},
		}},
	}
	lootboxes := map[string]Def{
		"box": {
			ItemDropRate: 0.5,
			FixedMoney:   MoneyRange{Min: 1, Max: 10},
			Pools:        []PoolRef{{PoolName: "pool_a", Weight: 1}},
		},
	}
	path := createTempConfigV2(t, pools, lootboxes)
	_, err := NewService(repo, &mockProgression{unlocked: true}, nil, path)
	require.Error(t, err)
}

// ============================================================================
// Error / validation tests
// ============================================================================

func TestUnknownPool_Error(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
	}}
	// Lootbox references "pool_nonexistent" which isn't defined.
	pools := map[string]PoolDef{
		"pool_a": {Items: []PoolItemDef{{ItemName: domain.ItemMoney, Weight: 1}}},
	}
	lootboxes := map[string]Def{
		"box": {
			ItemDropRate: 0.5,
			FixedMoney:   MoneyRange{Min: 1, Max: 10},
			Pools:        []PoolRef{{PoolName: "pool_nonexistent", Weight: 1}},
		},
	}
	path := createTempConfigV2(t, pools, lootboxes)
	_, err := NewService(repo, &mockProgression{unlocked: true}, nil, path)
	require.Error(t, err)
}

func TestUnknownItem_Error(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		// "ghost_item" intentionally absent
	}}
	pools := map[string]PoolDef{
		"pool_a": {Items: []PoolItemDef{{ItemName: "ghost_item", Weight: 1}}},
	}
	lootboxes := map[string]Def{
		"box": {
			ItemDropRate: 0.5,
			FixedMoney:   MoneyRange{Min: 1, Max: 10},
			Pools:        []PoolRef{{PoolName: "pool_a", Weight: 1}},
		},
	}
	path := createTempConfigV2(t, pools, lootboxes)
	_, err := NewService(repo, &mockProgression{unlocked: true}, nil, path)
	require.Error(t, err)
}

func TestBoxNotFound_NilNoError(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"sword":          swordItem(2, "sword", 10),
	}}
	s, err := buildSimpleService(t, repo, 1.0, 0, 10,
		[]PoolItemDef{{ItemName: "sword", Weight: 1}},
		nil,
	)
	require.NoError(t, err)

	drops, err := s.OpenLootbox(context.Background(), "nonexistent_box", 1, domain.QualityCommon)
	assert.NoError(t, err)
	assert.Nil(t, drops)
}

func TestZeroQuantity_NilNoError(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"sword":          swordItem(2, "sword", 10),
	}}
	s, err := buildSimpleService(t, repo, 1.0, 0, 10,
		[]PoolItemDef{{ItemName: "sword", Weight: 1}},
		nil,
	)
	require.NoError(t, err)

	drops, err := s.OpenLootbox(context.Background(), "box", 0, domain.QualityCommon)
	assert.NoError(t, err)
	assert.Nil(t, drops)
}

func TestInvalidConfig_Error(t *testing.T) {
	repo := &mockItemRepo{}
	_, err := NewService(repo, &mockProgression{unlocked: true}, nil, "nonexistent_path.json")
	require.Error(t, err)
}

// ============================================================================
// Quality tests
// ============================================================================

func TestQualityApplied_PerItem(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"sword":          swordItem(2, "sword", 100),
	}}

	s, err := buildSimpleService(t, repo, 1.0, 0, 10,
		[]PoolItemDef{{ItemName: "sword", Weight: 1}},
		nil,
	)
	require.NoError(t, err)

	// rnd sequence: [gate=0.0(pass), pool=0.0, item=0.0, quality=0.005(Legendary), upgrade=0.9(no)]
	rolls := []float64{0.0, 0.0, 0.0, 0.005, 0.9}
	idx := 0
	s.rnd = func() float64 { v := rolls[idx]; idx++; return v }

	drops, err := s.OpenLootbox(context.Background(), "box", 1, domain.QualityCommon)
	require.NoError(t, err)
	require.Len(t, drops, 1)
	assert.Equal(t, domain.QualityLegendary, drops[0].QualityLevel)
}

// ============================================================================
// Concurrent access test
// ============================================================================

func TestConcurrentOpen_NoRace(t *testing.T) {
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"sword":          swordItem(2, "sword", 10),
	}}

	s, err := buildSimpleService(t, repo, 0.5, 10, 100,
		[]PoolItemDef{{ItemName: "sword", Weight: 1}},
		nil,
	)
	require.NoError(t, err)

	var wg sync.WaitGroup
	errChan := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.OpenLootbox(context.Background(), "box", 1, domain.QualityCommon)
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
}

// ============================================================================
// Orphan tracking (no error, just a warning)
// ============================================================================

func TestOrphanTracking_NoError(t *testing.T) {
	// "orphan_sword" is in the repo but not in any pool — should warn but not error.
	repo := &mockItemRepo{items: map[string]*domain.Item{
		domain.ItemMoney: moneyItem(),
		"sword":          swordItem(2, "sword", 10),
		"orphan_sword":   swordItem(3, "orphan_sword", 50),
	}}

	_, err := buildSimpleService(t, repo, 1.0, 0, 10,
		[]PoolItemDef{{ItemName: "sword", Weight: 1}},
		nil,
	)
	// Orphaned items emit warnings, not errors.
	assert.NoError(t, err)
}
