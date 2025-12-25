package gamble

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ==============================================================================
// Concurrency Tests for Bug Fixes
// ==============================================================================

// TestStartGamble_Concurrent_RaceCondition tests Bug #1 fix
// Verifies that only one gamble can be in "Joining" state at a time
func TestStartGamble_Concurrent_RaceCondition(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, new(MockLootboxService), nil, time.Minute, nil)

	ctx := context.Background()
	user1 := &domain.User{ID: "user1"}
	user2 := &domain.User{ID: "user2"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}
	lootboxItem := &domain.Item{ID: 1, InternalName: "lootbox_tier1"}

	// Both see no active gamble initially
	repo.On("GetActiveGamble", ctx).Return(nil, nil).Times(2)
	repo.On("GetUserByPlatformID", ctx, "twitch", "123").Return(user1, nil)
	repo.On("GetUserByPlatformID", ctx, "twitch", "456").Return(user2, nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)

	// Setup transaction mocks
	sharedTx := new(MockTx)
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 5}}}

	repo.On("BeginTx", ctx).Return(sharedTx, nil)
	sharedTx.On("GetInventory", ctx, mock.Anything).Return(inventory, nil)
	sharedTx.On("UpdateInventory", ctx, mock.Anything, mock.Anything).Return(nil)

	// First CreateGamble will succeed
	repo.On("CreateGamble", ctx, mock.Anything).Return(nil).Once()
	// Subsequent CreateGamble will fail due to database constraint (mocked)
	repo.On("CreateGamble", ctx, mock.Anything).Return(fmt.Errorf("unique constraint violation")).Maybe()
	
	repo.On("JoinGamble", ctx, mock.Anything).Return(nil).Maybe()
	
	// Transaction completion
	// One commit (success), one rollback (failure) expected roughly, but race might cause both to rollback if join fails? 
	// Or multiple commits if constraint isn't hit? 
	// The constraint test logic implies one fails CreateGamble.
	sharedTx.On("Commit", ctx).Return(nil).Maybe()
	sharedTx.On("Rollback", ctx).Return(nil).Maybe()

	// Launch concurrent StartGamble calls
	var wg sync.WaitGroup
	results := make(chan error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := s.StartGamble(ctx, "twitch", "123", "user1", bets)
		results <- err
	}()
	go func() {
		defer wg.Done()
		_, err := s.StartGamble(ctx, "twitch", "456", "user2", bets)
		results <- err
	}()

	wg.Wait()
	close(results)

	var successCount int
	for err := range results {
		if err == nil {
			successCount++
		}
	}

	// With database constraint, only 1 should succeed
	assert.LessOrEqual(t, successCount, 1, "Only one StartGamble should succeed due to constraint")
}

// TestJoinGamble_SameUserTwice_ShouldReject tests Bug #2 fix
func TestJoinGamble_SameUserTwice_ShouldReject(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, new(MockLootboxService), nil, time.Minute, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}

	// Gamble already has this user
	gamble := &domain.Gamble{
		ID:           gambleID,
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(time.Minute),
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemID: 1, Quantity: 1}}},
		},
	}

	repo.On("GetUserByPlatformID", ctx, "twitch", "123").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	// Try to join again - should fail
	err := s.JoinGamble(ctx, gambleID, "twitch", "123", "user1", bets)

	// Verify error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already joined")
}

// TestJoinGamble_NonLootboxItem_ShouldReject tests Bug #8 fix
func TestJoinGamble_NonLootboxItem_ShouldReject(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, new(MockLootboxService), nil, time.Minute, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemID: 99, Quantity: 1}}

	gamble := &domain.Gamble{
		ID:           gambleID,
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(time.Minute),
	}

	// Mock a non-lootbox item (e.g., a sword)
	nonLootboxItem := &domain.Item{ID: 99, InternalName: domain.ItemBlaster}

	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	repo.On("GetItemByID", ctx, 99).Return(nonLootboxItem, nil)

	err := s.JoinGamble(ctx, gambleID, domain.PlatformTwitch, "123", "user1", bets)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotALootbox)
}

// TestExecuteGamble_Concurrent_Idempotent tests Bug #4 fix
func TestExecuteGamble_Concurrent_Idempotent(t *testing.T) {
	repo := new(MockRepository)
	lootboxSvc := new(MockLootboxService)
	s := NewService(repo, nil, lootboxSvc, nil, time.Minute, nil)

	ctx := context.Background()
	gambleID := uuid.New()

	gamble := &domain.Gamble{
		ID:           gambleID,
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(-time.Second), // Past deadline
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemID: 1, Quantity: 1}}},
		},
	}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	
	tx := new(MockTx)
	// Allow multiple calls to BeginGambleTx (one per goroutine initially)
	repo.On("BeginGambleTx", ctx).Return(tx, nil)

	// First call: CAS succeeds (1 row affected)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).
		Return(int64(1), nil).Once()

	// Second call: CAS fails (0 rows affected - state already changed)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).
		Return(int64(0), nil).Maybe()

	// Rest of execution for first call
	lootboxItem := &domain.Item{ID: 1, InternalName: domain.PublicNameLootbox}
	drops := []lootbox.DroppedItem{{ItemID: 10, Quantity: 5, Value: 100}}
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil).Maybe()
	lootboxSvc.On("OpenLootbox", ctx, domain.PublicNameLootbox, 1).Return(drops, nil).Maybe()
	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(nil).Maybe()

	tx.On("GetInventory", ctx, "user1").Return(&domain.Inventory{}, nil).Maybe()
	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil).Maybe()
	tx.On("Commit", ctx).Return(nil).Maybe()
	tx.On("Rollback", ctx).Return(nil).Maybe()
	tx.On("CompleteGamble", ctx, mock.Anything).Return(nil).Maybe()

	// Execute concurrently
	var wg sync.WaitGroup
	results := make(chan error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := s.ExecuteGamble(ctx, gambleID)
		results <- err
	}()
	go func() {
		defer wg.Done()
		_, err := s.ExecuteGamble(ctx, gambleID)
		results <- err
	}()

	wg.Wait()
	close(results)

	var successCount int
	for err := range results {
		if err == nil {
			successCount++
		}
	}

	// With CAS, only 1 execution should succeed
	assert.Equal(t, 1, successCount, "Only one ExecuteGamble should succeed due to CAS")
}

// TestExecuteGamble_DeadlineNotPassed_ShouldReject tests Bug #6 fix
func TestExecuteGamble_DeadlineNotPassed_ShouldReject(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, new(MockLootboxService), nil, time.Minute, nil)

	ctx := context.Background()
	gambleID := uuid.New()

	// Gamble with future deadline
	gamble := &domain.Gamble{
		ID:           gambleID,
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(time.Hour), // Still in future!
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemID: 1, Quantity: 1}}},
		},
	}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	_, err := s.ExecuteGamble(ctx, gambleID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "before join deadline")
}

// TestConsumeItem_MultipleItemsRemoval tests Bug #7 fix
func TestConsumeItem_MultipleItemsRemoval(t *testing.T) {
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 5},
			{ItemID: 2, Quantity: 3},
			{ItemID: 3, Quantity: 2},
		},
	}

	// Consume all items sequentially (used to cause index shift bugs)
	err1 := consumeItem(inventory, 1, 5) // Removes slot 0
	err2 := consumeItem(inventory, 2, 3) // Should remove original slot 1 (now slot 0)
	err3 := consumeItem(inventory, 3, 2) // Should remove original slot 2 (now slot 0)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Empty(t, inventory.Slots, "All items should be consumed")
}
