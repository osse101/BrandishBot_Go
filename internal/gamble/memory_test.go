package gamble

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/testing/leaktest"
)

//  =============================================================================
// Memory Leak Tests
// =============================================================================
// NOTE: Most mocks are defined in service_test.go and reused here

func TestStartGamble_NoGoroutineLeak(t *testing.T) {
	// Setup mocks (defined in service_test.go)
	repo := new(MockRepository)
	lootboxSvc := new(MockLootboxService)
	statsSvc := new(MockStatsService)

	user := &domain.User{ID: "user123", Username: "tester"}
	tx := new(MockTx)
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 10}, // lootbox
		},
	}

	repo.On("GetUserByPlatformID", mock.Anything, domain.PlatformDiscord, "user123").Return(user, nil)
	repo.On("GetInventory", mock.Anything, user.ID).Return(inventory, nil)
	repo.On("GetActiveGamble", mock.Anything).Return(nil, nil)
	// Bug #8 Fix requires item validation - mock lootbox item
	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	repo.On("GetItemByName", mock.Anything, domain.ItemLootbox1).Return(lootboxItem, nil)
	repo.On("GetItemByID", mock.Anything, 1).Return(lootboxItem, nil)
	repo.On("BeginTx", mock.Anything).Return(tx, nil)
	tx.On("GetInventory", mock.Anything, user.ID).Return(inventory, nil)
	tx.On("UpdateInventory", mock.Anything, user.ID, mock.Anything).Return(nil)
	tx.On("Commit", mock.Anything).Return(nil)
	tx.On("Rollback", mock.Anything).Return(nil)
	repo.On("CreateGamble", mock.Anything, mock.Anything).Return(nil)
	repo.On("JoinGamble", mock.Anything, mock.Anything).Return(nil)

	svc := NewService(repo, nil, nil, lootboxSvc, statsSvc, 30*time.Second, nil, nil, nil)
	checker := leaktest.NewGoroutineChecker(t)

	// Execute
	ctx := context.Background()
	bets := []domain.LootboxBet{{ItemName: "lootbox_tier1", Quantity: 1}}
	_, err := svc.StartGamble(ctx, domain.PlatformDiscord, "user123", "tester", bets)

	if err != nil {
		t.Logf("StartGamble error (may be expected): %v", err)
	}

	// Check for leaks (allow small tolerance)
	checker.Check(1)
}

func TestExecuteGamble_NoGoroutineLeak(t *testing.T) {
	// Setup
	repo := new(MockRepository)
	lootboxSvc := new(MockLootboxService)
	statsSvc := new(MockStatsService)

	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateCreated,
		Participants: []domain.Participant{
			{UserID: "user1", Username: "player1"},
		},
	}

	repo.On("GetGamble", mock.Anything, gambleID).Return(gamble, nil)
	repo.On("UpdateGambleState", mock.Anything, gambleID, mock.Anything).Return(nil)
	repo.On("SaveOpenedItems", mock.Anything, mock.Anything).Return(nil)
	repo.On("CompleteGamble", mock.Anything, mock.Anything).Return(nil)
	lootboxSvc.On("OpenLootbox", mock.Anything, mock.Anything, mock.Anything).Return([]lootbox.DroppedItem{}, nil).Maybe()

	svc := NewService(repo, nil, nil, lootboxSvc, statsSvc, 30*time.Second, nil, nil, nil)
	checker := leaktest.NewGoroutineChecker(t)

	// Execute
	ctx := context.Background()
	_, err := svc.ExecuteGamble(ctx, gambleID)

	if err != nil {
		t.Logf("ExecuteGamble error (may be expected): %v", err)
	}

	// Allow time for any async operations
	time.Sleep(50 * time.Millisecond)

	// Check for leaks (allow small tolerance)
	checker.Check(1)
}
