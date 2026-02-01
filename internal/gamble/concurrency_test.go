package gamble

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
)

func TestStartGamble_Concurrent_RaceCondition(t *testing.T) {
	// Logic: We simulate the DB constraint by making the second CreateGamble call fail.
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	user1 := &domain.User{ID: "user1"}
	user2 := &domain.User{ID: "user2"}
	bets := []domain.LootboxBet{{ItemName: "lootbox_tier1", Quantity: 1}}

	// Both see no active gamble
	repo.On("GetActiveGamble", ctx).Return(nil, nil)

	repo.On("GetUserByPlatformID", ctx, "twitch", "123").Return(user1, nil)
	repo.On("GetUserByPlatformID", ctx, "twitch", "456").Return(user2, nil)

	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	repo.On("GetItemByName", ctx, "lootbox_tier1").Return(lootboxItem, nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)

	tx := new(MockTx)
	inv1 := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 5}}}
	inv2 := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 5}}}

	repo.On("GetInventory", ctx, "user1").Return(inv1, nil).Maybe()
	repo.On("GetInventory", ctx, "user2").Return(inv2, nil).Maybe()

	repo.On("BeginTx", ctx).Return(tx, nil).Twice()

	tx.On("GetInventory", ctx, "user1").Return(inv1, nil)
	tx.On("GetInventory", ctx, "user2").Return(inv2, nil)

	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx.On("UpdateInventory", ctx, "user2", mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	// Simulate one success and one failure due to constraint
	repo.On("CreateGamble", ctx, mock.Anything).Return(nil).Once()
	repo.On("CreateGamble", ctx, mock.Anything).Return(domain.ErrGambleAlreadyActive).Once()

	repo.On("JoinGamble", ctx, mock.Anything).Return(nil).Maybe()

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
	var errorCount int
	for err := range results {
		if err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	assert.Equal(t, 1, successCount, "Only one gamble start should succeed")
	assert.Equal(t, 1, errorCount, "One gamble start should fail")
}

func TestJoinGamble_SameUserTwice_ShouldReject(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user1"}

	gamble := &domain.Gamble{
		ID:           gambleID,
		InitiatorID:  "initiator_user",
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(time.Minute),
		Participants: []domain.Participant{
			{UserID: "initiator_user", GambleID: gambleID, LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}

	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)

	repo.On("GetUserByPlatformID", ctx, "twitch", "123").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	tx := new(MockTx)
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 5}}}
	repo.On("GetInventory", ctx, "user1").Return(inventory, nil).Maybe()
	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("GetInventory", ctx, "user1").Return(inventory, nil)
	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx.On("Rollback", ctx).Return(nil)

	// Simulate DB Constraint Violation
	repo.On("JoinGamble", ctx, mock.Anything).Return(domain.ErrUserAlreadyJoined)

	err := s.JoinGamble(ctx, gambleID, domain.PlatformTwitch, "123", "user1")

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUserAlreadyJoined)
}

func TestExecuteGamble_Concurrent_Idempotent(t *testing.T) {
	repo := new(MockRepository)
	lootboxSvc := new(MockLootboxService)
	s := NewService(repo, nil, nil, lootboxSvc, nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()

	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
		JoinDeadline: time.Now().Add(-time.Minute), // Deadline PASSED, ready to execute
	}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	tx1, tx2 := new(MockTx), new(MockTx)
	repo.On("BeginGambleTx", ctx).Return(tx1, nil).Once()
	repo.On("BeginGambleTx", ctx).Return(tx2, nil).Once()

	// One succeeds, one fails
	tx1.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)
	tx2.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(0), nil)

	// Failed one rolls back
	tx2.On("Rollback", ctx).Return(nil).Maybe()

	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	drops := []lootbox.DroppedItem{{ItemID: 10, Quantity: 5, Value: 100}}
	repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)
	lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, 1).Return(drops, nil)
	tx1.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)

	tx1.On("GetInventory", ctx, "user1").Return(&domain.Inventory{}, nil)
	tx1.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx1.On("CompleteGamble", ctx, mock.Anything).Return(nil)
	tx1.On("Commit", ctx).Return(nil)
	tx1.On("Rollback", ctx).Return(nil).Maybe()

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

	assert.Equal(t, 1, successCount)
}

func TestConsumeItem_MultipleItemsRemoval(t *testing.T) {
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 5},
			{ItemID: 2, Quantity: 3},
			{ItemID: 3, Quantity: 2},
		},
	}

	err := consumeItem(inventory, 1, 5)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(inventory.Slots))

	err = consumeItem(inventory, 2, 3)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(inventory.Slots))

	err = consumeItem(inventory, 3, 2)
	assert.NoError(t, err)
	assert.Empty(t, inventory.Slots)
}
