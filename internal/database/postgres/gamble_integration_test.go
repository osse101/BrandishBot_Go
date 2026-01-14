package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
)

// setupGambleIntegrationTest sets up the dependencies for gamble integration tests
// It reuses setupIntegrationTest for the DB container but builds its own services
func setupGambleIntegrationTest(t *testing.T) (*pgxpool.Pool, *UserRepository, gamble.Service, func()) {
	// 1. Setup DB using existing helper (returns pool, repo, and a user service we ignore)
	pool, repo, _ := setupIntegrationTest(t)
	if pool == nil {
		// setupIntegrationTest skips if docker fails
		return nil, nil, nil, nil
	}

	// 2. Setup Lootbox Service with deterministic loot table
	lootTable := map[string][]lootbox.LootItem{
		"lootbox_tier1": {
			{ItemName: "money", Min: 100, Max: 100, Chance: 1.0},
		},
	}
	lootTableData, err := json.Marshal(map[string]interface{}{"tables": lootTable})
	require.NoError(t, err)

	tmpDir := t.TempDir()
	lootTablePath := filepath.Join(tmpDir, "loot_tables.json")
	err = os.WriteFile(lootTablePath, lootTableData, 0644)
	require.NoError(t, err)

	// Since we are reusing the pool/repo which uses database connection,
	// and lootbox service needs repo to look up items "money".
	// The real lootbox service uses a repo interface.
	// postgres.UserRepository implements it (GetItemByName, GetItemsByNames).
	// We need to make sure the item "money" exists in the DB (seeded by migrations).

	lootRepo := NewUserRepository(pool)
	lootSvc, err := lootbox.NewService(lootRepo, lootTablePath)
	require.NoError(t, err)

	// 3. Setup Gamble Service
	gambleRepo := NewGambleRepository(pool)
	eventBus := event.NewMemoryBus()
	statsSvc := &MockStatsService{}
	jobSvc := &MockJobService{}

	// mock progression service
	progressionSvc := &MockProgressionService{}

	gambleSvc := gamble.NewService(
		gambleRepo,
		eventBus,
		lootSvc,
		statsSvc,
		1*time.Minute, // Join duration
		jobSvc,
		progressionSvc,
	)

	cleanup := func() {
		// Pool is cleaned up by setupIntegrationTest t.Cleanup
	}

	return pool, repo, gambleSvc, cleanup
}

type MockProgressionService struct{}

func (m *MockProgressionService) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	return baseValue, nil
}

func (m *MockProgressionService) IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error) {
	return true, nil
}

func TestGambleLifecycle_Integration(t *testing.T) {
	_, repo, svc, _ := setupGambleIntegrationTest(t)
	if svc == nil {
		return // Skipped
	}

	ctx := context.Background()

	// --- Step 1: Prep Users ---
	userA := &domain.User{Username: "UserA", TwitchID: "twitchA"}
	userB := &domain.User{Username: "UserB", TwitchID: "twitchB"}

	require.NoError(t, repo.UpsertUser(ctx, userA))
	require.NoError(t, repo.UpsertUser(ctx, userB))

	// Refresh to get IDs
	userA, _ = repo.GetUserByUsername(ctx, "UserA")
	userB, _ = repo.GetUserByUsername(ctx, "UserB")

	// Verify Item "lootbox_tier1" exists
	// GetItemByName in UserRepository looks up by internal name (despite the name)
	lbItem, err := repo.GetItemByName(ctx, "lootbox_tier1")
	require.NoError(t, err)
	require.NotNil(t, lbItem)

	// Verify Item "money" exists
	moneyItem, err := repo.GetItemByName(ctx, "money")
	require.NoError(t, err)
	require.NotNil(t, moneyItem)

	// Seed Inventory
	// User A: 5 lootboxes
	// User B: 5 lootboxes
	invA := domain.Inventory{Slots: []domain.InventorySlot{{ItemID: lbItem.ID, Quantity: 5}}}
	invB := domain.Inventory{Slots: []domain.InventorySlot{{ItemID: lbItem.ID, Quantity: 5}}}

	require.NoError(t, repo.UpdateInventory(ctx, userA.ID, invA))
	require.NoError(t, repo.UpdateInventory(ctx, userB.ID, invB))

	// --- Step 2: Start Gamble ---
	// User A starts gamble betting 2 lootboxes
	// Lootbox returns 100 money each. Total value = 200.
	betsA := []domain.LootboxBet{{ItemID: lbItem.ID, Quantity: 2}}
	gamble, err := svc.StartGamble(ctx, domain.PlatformTwitch, "twitchA", "UserA", betsA)
	require.NoError(t, err)
	require.NotNil(t, gamble)
	assert.Equal(t, domain.GambleStateJoining, gamble.State)

	// Verify inventory deduction A
	invAAfterStart, err := repo.GetInventory(ctx, userA.ID)
	require.NoError(t, err)
	require.Equal(t, 3, getQty(invAAfterStart, lbItem.ID))

	// --- Step 3: Join Gamble ---
	// User B joins betting 1 lootbox
	// Lootbox returns 100 money. Total value = 100.
	betsB := []domain.LootboxBet{{ItemID: lbItem.ID, Quantity: 1}}
	err = svc.JoinGamble(ctx, gamble.ID, domain.PlatformTwitch, "twitchB", "UserB", betsB)
	require.NoError(t, err)

	// Verify inventory deduction B
	invBAfterJoin, err := repo.GetInventory(ctx, userB.ID)
	require.NoError(t, err)
	require.Equal(t, 4, getQty(invBAfterJoin, lbItem.ID))

	// --- Step 4: Execute Gamble ---
	// Expected Outcome:
	// User A Value: 200
	// User B Value: 100
	// Winner: User A
	// Winner receives: (2+1) * 100 money = 300 money.

	result, err := svc.ExecuteGamble(ctx, gamble.ID)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify Result
	assert.Equal(t, userA.ID, result.WinnerID)
	assert.Equal(t, int64(300), result.TotalValue)

	// Verify Gamble State in DB
	finalGamble, err := svc.GetGamble(ctx, gamble.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.GambleStateCompleted, finalGamble.State)

	// Verify Winner Inventory (User A)
	// Should have 3 lootboxes left + 300 money
	invAFinal, err := repo.GetInventory(ctx, userA.ID)
	require.NoError(t, err)
	require.Equal(t, 3, getQty(invAFinal, lbItem.ID))
	require.Equal(t, 300, getQty(invAFinal, moneyItem.ID))

	// Verify Loser Inventory (User B)
	// Should have 4 lootboxes left + 0 money
	invBFinal, err := repo.GetInventory(ctx, userB.ID)
	require.NoError(t, err)
	require.Equal(t, 4, getQty(invBFinal, lbItem.ID))
	require.Equal(t, 0, getQty(invBFinal, moneyItem.ID))

	// Wait for async stats/xp if needed (shutdown handles this usually, but we check values directly)
	// We mocked job/stats so no side effects to check there.
}

func getQty(inv *domain.Inventory, itemID int) int {
	for _, s := range inv.Slots {
		if s.ItemID == itemID {
			return s.Quantity
		}
	}
	return 0
}

func TestGamble_Concurrency_Join(t *testing.T) {
	_, repo, svc, _ := setupGambleIntegrationTest(t)
	if svc == nil {
		return
	}
	ctx := context.Background()

	// Setup host
	host := &domain.User{Username: "Host", TwitchID: "twitchHost"}
	require.NoError(t, repo.UpsertUser(ctx, host))
	host, _ = repo.GetUserByUsername(ctx, "Host")

	lbItem, _ := repo.GetItemByName(ctx, "lootbox_tier1")
	invHost := domain.Inventory{Slots: []domain.InventorySlot{{ItemID: lbItem.ID, Quantity: 10}}}
	repo.UpdateInventory(ctx, host.ID, invHost)

	// Start gamble
	bets := []domain.LootboxBet{{ItemID: lbItem.ID, Quantity: 1}}
	gamble, err := svc.StartGamble(ctx, domain.PlatformTwitch, "twitchHost", "Host", bets)
	require.NoError(t, err)

	// Concurrent Joiners
	count := 10
	var wg sync.WaitGroup
	errChan := make(chan error, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			uname := fmt.Sprintf("User%d", idx)
			pid := fmt.Sprintf("twitch%d", idx)
			u := &domain.User{Username: uname, TwitchID: pid}

			// Setup user with inventory
			// We need a separate connection/context for setup to avoid contention?
			// No, repository is thread safe.
			if err := repo.UpsertUser(ctx, u); err != nil {
				errChan <- err
				return
			}
			u, _ = repo.GetUserByUsername(ctx, uname)
			inv := domain.Inventory{Slots: []domain.InventorySlot{{ItemID: lbItem.ID, Quantity: 1}}}
			repo.UpdateInventory(ctx, u.ID, inv)

			// Join
			err := svc.JoinGamble(ctx, gamble.ID, domain.PlatformTwitch, pid, uname, bets)
			if err != nil {
				errChan <- fmt.Errorf("user %d failed to join: %w", idx, err)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent join error: %v", err)
	}

	// Verify all joined
	finalGamble, _ := svc.GetGamble(ctx, gamble.ID)
	assert.Equal(t, count+1, len(finalGamble.Participants)) // +1 for host
}
